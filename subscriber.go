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
	}
}

// Subscriber adds routes to a routes to a router on incomming
// subscribe packets.
type Subscriber struct {
	*Router
	PubHandler
}

func (s *Subscriber) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Subscribe:
			for _, f := range p.Filters() {
				r := NewRoute(f.Filter(), s.PubHandler)
				s.Router.AddRoute(r)
			}
		}
		return next(ctx, p)
	}
}
