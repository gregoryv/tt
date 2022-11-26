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
		// todo handle subscriptions
	default:
		return fmt.Errorf("%T unhandled!", p)
	}
	return ctx.Err()
}

// ----------------------------------------

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}
