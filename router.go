package tt

import (
	"context"
	"log"
	"sync"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/arn"
)

// newRouter returns a router for handling the given subscriptions.
func newRouter() *router {
	return &router{
		rut: arn.NewTree(),
		log: log.New(log.Writer(), "router ", log.Flags()),
	}
}

type router struct {
	m   sync.Mutex
	rut *arn.Tree
	log *log.Logger
}

func (r *router) String() string {
	return plural(len(r.rut.Leafs()), "subscription")
}

func (r *router) AddSubscriptions(v ...*subscription) {
	r.m.Lock()
	defer r.m.Unlock()
	for _, s := range v {
		for _, f := range s.filters {
			n := r.rut.AddFilter(f)
			if n.Value == nil {
				n.Value = v
			} else {
				n.Value = append(n.Value.([]*subscription), s)
			}
		}
	}
}

func (r *router) removeFilters(sc *sclient, filters []string) {
	r.m.Lock()
	defer r.m.Unlock()

	// wip remove filters in router
}

// Route routes mq.Publish packets by topic name.
func (r *router) Route(ctx context.Context, p mq.Packet) error {
	switch p := p.(type) {
	case *mq.Publish:
		// optimization opportunity by pooling a set of results
		var result []*arn.Node
		r.rut.Match(&result, p.TopicName())
		for _, n := range result {
			for _, s := range n.Value.([]*subscription) {
				for _, h := range s.handlers {
					if err := h(ctx, p); err != nil {
						r.log.Println("handle", p, err)
					}
				}
			}
		}
	}
	return ctx.Err()
}
