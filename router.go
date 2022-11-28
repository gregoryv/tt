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
					// todo how to handle errors when routing to clients?
					// this will depend on QoS aswell
					_ = h(ctx, p)
				}
			}
		}
	}
	return ctx.Err()
}

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}
