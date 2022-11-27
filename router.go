package tt

import (
	"context"
	"fmt"

	"github.com/gregoryv/mq"
)

func NewRouter(v ...*Route) *Router {
	return &Router{routes: v}
}

type Router struct {
	routes []*Route
}

func (r *Router) String() string {
	return plural(len(r.routes), "route")
}

func (r *Router) AddRoute(v *Route) {
	r.routes = append(r.routes, v)
}

// In forwards routes mq.Publish packets by topic name.
func (r *Router) Handle(ctx context.Context, p mq.Packet) error {
	switch p := p.(type) {
	case *mq.Publish:
		// todo naive implementation looping over each route
		for _, route := range r.routes {
			if _, ok := route.Match(p.TopicName()); ok {
				for _, h := range route.handlers {
					_ = h(ctx, p) // todo how to handle errors
				}
			}
		}

	default:
		return fmt.Errorf("%T unhandled!", p)
	}
	return ctx.Err()
}

// ----------------------------------------

func NewSubscriber(r *Router, transmit Handler) *Subscriber {
	return &Subscriber{
		Router:   r,
		transmit: transmit,
	}
}

// Subscriber adds routes to a routes to a router on incomming
// subscribe packets.
type Subscriber struct {
	*Router
	transmit Handler
}

// todo mq.Subscriber has no WellFormed
func (s *Subscriber) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		switch p := p.(type) {
		case *mq.Subscribe:
			h := func(ctx context.Context, p *mq.Publish) error {
				return s.transmit(ctx, p)
			}
			f := p.Filters()[0] // todo add route for all
			_ = f               // todo cannot access TopicFilter.filter
			r := NewRoute("", h)
			s.Router.AddRoute(r)
			return fmt.Errorf("Subscriber.In: todo")
		}
		return next(ctx, p)
	}
}

// ----------------------------------------

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}
