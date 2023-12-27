package tt

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gregoryv/mq"
)

// newRouter returns a router for handling the given subscriptions.
func newRouter() *router {
	return &router{
		log:     log.New(log.Writer(), "router ", log.Flags()),
		filtSub: make(map[string][]*subscription),
	}
}

type router struct {
	log *log.Logger

	// topic filter -> subscription
	m       sync.RWMutex
	filtSub map[string][]*subscription
}

func (r *router) String() string {
	var c int
	for _, s := range r.filtSub {
		c += len(s)
	}
	return plural(c, "subscription")
}

func plural(v int, word string) string {
	if v > 1 {
		word = word + "s"
	}
	return fmt.Sprintf("%v %s", v, word)
}

func (r *router) AddSubscriptions(v ...*subscription) {
	r.m.Lock()
	defer r.m.Unlock()
	for _, s := range v {
		for _, f := range s.filters {
			r.filtSub[f] = append(r.filtSub[f], s)
		}
	}
}

func (r *router) removeFilters(sc *sclient, filters []string) {
	r.m.Lock()
	defer r.m.Unlock()

	// todo remove filters in router
}

// Route routes mq.Publish packets by topic name.
func (r *router) Route(ctx context.Context, p mq.Packet) error {
	r.m.RLock()
	defer r.m.RUnlock()
	switch p := p.(type) {
	case *mq.Publish:
		for filter, subscriptions := range r.filtSub {
			if match(filter, p.TopicName()) {
				for _, s := range subscriptions {
					for _, h := range s.handlers {
						h(ctx, p)
					}
				}
			}
		}

	}
	return ctx.Err()
}
