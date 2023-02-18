package tt

import (
	"context"
	"time"

	"github.com/gregoryv/mq"
)

// NewKeepAlive returns a middleware which keeps connection alive by
// sending pings within the given interval. The interval is rounded to
// seconds.
func NewKeepAlive(interval time.Duration) *KeepAlive {
	return &KeepAlive{
		interval:   uint16(interval.Seconds()),
		tick:       time.NewTicker(time.Second),
		packetSent: make(chan struct{}, 1),
	}
}

// KeepAlive implements client logic for keeping connection alive.
//
// See [3.1.2.10 Keep Alive]
//
// [3.1.2.10 Keep Alive]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Keep_Alive_1
type KeepAlive struct {
	transmit Handler
	interval uint16 // seconds
	tick     *time.Ticker

	packetSent   chan struct{}
	disconnected chan struct{}
}

func (k *KeepAlive) SetTransmitter(v Handler) { k.transmit = v }

// In adjusts interval based on ConnAck.ServerKeepAlive
func (k *KeepAlive) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.ConnAck:
			// do as the server says
			if v := p.ServerKeepAlive(); v > 0 {
				k.interval = v
			}
		}
		return next(ctx, p)
	}
}

// Out sets keep alive on connect packets.
func (k *KeepAlive) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Connect:
			p.SetKeepAlive(k.interval)
			k.tick.Reset(time.Second)
			go k.run(ctx)
		}
		err := next(ctx, p)
		if err == nil {
			k.packetSent <- struct{}{}
		}
		return err
	}
}

func (k *KeepAlive) run(ctx context.Context) {
	last := time.Now()
	for {
		select {
		case <-ctx.Done():
			k.tick.Stop()
			return

		case <-k.packetSent:
			last = time.Now()

		case <-k.tick.C:
			// check if its time to send a ping
			if time.Since(last).Round(time.Second) == time.Duration(k.interval)*time.Second-time.Second {
				k.transmit(ctx, mq.NewPingReq())
			}
		}
	}
}
