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
		// remove hash before storing
		prefix := f[:len(f)-1]
		m.hashEnd[prefix] = append(m.hashEnd[prefix], fwd)
	}
}

type ForwardFunc func(context.Context, *mq.Publish) error

type Filter string

func (f *Filter) HasHashSuffix() bool {
	return strings.HasSuffix(string(*f), "#")
}
