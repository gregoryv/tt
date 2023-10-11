package mix

import (
	"context"
	"strings"
	"sync"

	"github.com/gregoryv/mq"
)

func NewFmux() *Fmux {
	return &Fmux{
		hashEnd: make(map[Filter][]ForwardFunc),
	}
}

/*

Fmux is a mq.Publish packet router which groups topics for improved
performance.

*/
type Fmux struct {
	tex     sync.RWMutex
	hashEnd map[Filter][]ForwardFunc
}

func (m *Fmux) Handle(f Filter, fwd ForwardFunc) {
	m.tex.Lock()
	defer m.tex.Unlock()
	if f.HasHashSuffix() {
		// remove /# before storing
		prefix := f[:len(f)-2]
		m.hashEnd[prefix] = append(m.hashEnd[prefix], fwd)
	}
}

func (m *Fmux) Route(topic string) (n int) {
	m.tex.RLock()
	defer m.tex.RUnlock()
	for filter, _ := range m.hashEnd {
		if strings.HasPrefix(topic, string(filter)) {
			n++
		}
	}
	return
}

type ForwardFunc func(context.Context, *mq.Publish) error

type Filter string

func (f *Filter) HasHashSuffix() bool {
	return strings.HasSuffix(string(*f), "#")
}
