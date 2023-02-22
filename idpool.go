package tt

import (
	"context"
	"fmt"
	"time"

	"github.com/gregoryv/mq"
)

// NewIDPool returns a iDPool of reusable id's from 1..max, 0 is not used
func newIDPool(max uint16) *iDPool {
	o := iDPool{
		nextTimeout: 3 * time.Second,
		max:         max,
		used:        make([]time.Time, max+1),
		values:      make(chan uint16, max),
	}
	for i := uint16(1); i <= max; i++ {
		o.values <- i
	}
	return &o
}

type iDPool struct {
	nextTimeout time.Duration
	max         uint16
	used        []time.Time // flags id that has been used in Out handler
	values      chan uint16
}

// In checks if incoming packet has a packet ID, if so it's
// returned to the pool before next handler is called.
func (o *iDPool) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(mq.HasPacketID); ok {
			_ = o.reuse(p.PacketID())
		}
		return next(ctx, p)
	}
}

// reuse returns the given value to the pool, returns the reused value
// or 0 if ignored
func (o *iDPool) reuse(v uint16) uint16 {
	if v == 0 || v > o.max {
		return 0
	}
	if o.used[v].IsZero() {
		return 0
	}
	o.values <- v
	o.used[v] = zero
	return v
}

// Out on outgoing packets, refs MQTT-2.2.1-3
func (o *iDPool) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if err := o.SetPacketID(ctx, p); err != nil {
			return err
		}
		return next(ctx, p)
	}
}

func (o *iDPool) SetPacketID(ctx context.Context, p mq.Packet) error {
	if p, ok := p.(mq.HasPacketID); ok {
		switch p := p.(type) {
		case *mq.Publish:
			if p.QoS() > 0 && p.PacketID() == 0 {
				id, err := o.next(ctx)
				if err != nil {
					return err
				}
				p.SetPacketID(id)
			}

		case *mq.Subscribe:
			id, err := o.next(ctx)
			if err != nil {
				return err
			}
			p.SetPacketID(id)

		case *mq.Unsubscribe:
			id, err := o.next(ctx)
			if err != nil {
				return err
			}
			p.SetPacketID(id)
		}
	}
	return nil
}

// next returns the next available ID, blocks until one is available
// or context is canceled. next is safe for concurrent use by multiple
// goroutines.
func (o *iDPool) next(ctx context.Context) (uint16, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()

	case <-time.After(o.nextTimeout):
		return 0, ErrIDPoolEmpty

	case v := <-o.values:
		o.used[v] = time.Now()
		return v, nil
	}
}

var ErrIDPoolEmpty = fmt.Errorf("no available packet ids")

var zero = time.Time{}
