package tt

import (
	"context"
	"fmt"
	"log"

	"github.com/gregoryv/mq"
)

// NewRouter returns a router for handling the given subscriptions.
func NewRouter(v ...*Subscription) *Router {
	return &Router{
		subs: v,
		log:  log.New(log.Writer(), "router ", log.Flags()),
	}
}

type Router struct {
	subs []*Subscription

	log *log.Logger
}

func (r *Router) String() string {
	return plural(len(r.subs), "subscription")
}

func (r *Router) AddRoute(v *Subscription) {
	r.subs = append(r.subs, v)
}

// In forwards routes mq.Publish packets by topic name.
func (r *Router) Handle(ctx context.Context, p mq.Packet) error {
	switch p := p.(type) {
	case *mq.Publish:
		// naive implementation looping over each route, improve at
		// some point
		for _, s := range r.subs {
			if _, ok := s.Match(p.TopicName()); ok {
				for _, h := range s.handlers {
					// maybe we'll have to have a different routing mechanism for
					// client side handling subscriptions compared to server side.
					// As server may have to adapt packages before sending and
					// there will be a QoS on each subscription that we need to consider.
					if err := h(ctx, p); err != nil {
						r.log.Println("handle", p, err)
					}
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
