package tt

import (
	"context"
	"time"

	"github.com/gregoryv/mq"
)

// NewKeepAlive returns a middleware which keeps connection alive by
// sending pings within the given interval. The interval is rounded to
// seconds.
func newKeepAlive(interval time.Duration) *keepAlive {
	return &keepAlive{
		interval:   uint16(interval.Seconds()),
		tick:       time.NewTicker(time.Second),
		packetSent: make(chan struct{}, 1),
	}
}

// keepAlive implements client logic for keeping connection alive.
//
// See [3.1.2.10 Keep Alive]
//
// [3.1.2.10 Keep Alive]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Keep_Alive_1
type keepAlive struct {
	transmit Handler
	interval uint16 // seconds
	tick     *time.Ticker

	packetSent   chan struct{}
	disconnected chan struct{}
}

func (k *keepAlive) SetTransmitter(v Handler) { k.transmit = v }

func (k *keepAlive) run(ctx context.Context) {
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
