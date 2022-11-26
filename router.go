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

// SubscribeHandler should be used for incoming packets.
//
// We cannot use Inner as the connection information is missing.
func (r *Router) NewSubscriber(transmit Handler) *Subscriber {
	return &Subscriber{
		Router:   r,
		transmit: transmit,
	}
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

// Subscriber keeps track of subscriptions
type Subscriber struct {
	*Router
	transmit Handler
}

func (s *Subscriber) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		return fmt.Errorf(": todo")
	}
}

// ----------------------------------------

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}
