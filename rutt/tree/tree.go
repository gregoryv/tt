package tree

import (
	"strings"
	"sync"
)

func NewTree() *Tree {
	return &Tree{
		root: NewNode(""),
	}
}

type Tree struct {
	m    sync.Mutex
	root *Node
}

func (t *Tree) String() string {
	return t.root.String()
}

func (t *Tree) Handle(filter string) {
	t.m.Lock()
	defer t.m.Unlock()
	t.handle(filter, t.root)
}

func (t *Tree) handle(filter string, n *Node) {
	if filter == "" {
		return
	}
	i := strings.Index(filter, "/")
	if i > 0 {
		part := filter[:i]
		x := n.Find(part)
		if x == nil {
			x = NewNode(part)
		}
		n.Children = append(n.Children, x)
		t.handle(filter[i+1:], x)
		return
	}
	x := n.Find(filter)
	if x == nil {
		x = NewNode(filter)
	}
	n.Children = append(n.Children, x)
}

func (t *Tree) Route(topic string) int {
	return t.root.route(topic)
}
