package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

func NewSubscriber(r *Router, transmit Handler) *Subscriber {
	return &Subscriber{
		Router: r,
		PubHandler: func(ctx context.Context, p *mq.Publish) error {
			return transmit(ctx, p)
		},
		transmit: transmit,
	}
}

// Subscriber adds routes to a routes to a router on incomming
// subscribe packets.
type Subscriber struct {
	*Router

	// PubHandler is used in the routes to transmit packets to a
	// specific client
	PubHandler

	transmit func(ctx context.Context, p mq.Packet) error
}

func (s *Subscriber) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Subscribe:
			a := mq.NewSubAck()
			a.SetPacketID(p.PacketID())
			for _, f := range p.Filters() {
				r := NewRoute(f.Filter(), s.PubHandler)
				s.Router.AddRoute(r)
				// 3.9.3 SUBACK Payload
				a.AddReasonCode(mq.Success)
			}
			s.transmit(ctx, a)
			// todo should we return here, ie. not let subsequent
			// handlers do their thing? I guess it's an optimization thing
		}
		return next(ctx, p)
	}
}
