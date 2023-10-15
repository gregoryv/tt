package arn

import (
	"strings"
	"sync"
)

func NewTree() *Tree {
	return &Tree{
		root: NewNode("", nil),
	}
}

type Tree struct {
	m    sync.Mutex
	root *Node
}

func (t *Tree) Match(result *[]*Node, topic string) {
	parts := strings.Split(topic, "/")

	for _, child := range t.root.children {
		child.Match(result, parts, 0)
	}
}

func (t *Tree) Filters() []string {
	var filters []string
	for _, l := range t.root.Leafs() {
		filters = append(filters, l.Filter())
	}
	return filters
}

func (t *Tree) AddFilter(filter string, v any) *Node {
	if filter == "" {
		return nil
	}
	t.m.Lock()
	defer t.m.Unlock()
	parts := strings.Split(filter, "/")
	n := t.addParts(t.root, parts, v)
	// t.root is just a virtual parent
	for _, top := range t.root.children {
		top.parent = nil
	}
	return n
}

func (t *Tree) Find(filter string) (*Node, bool) {
	if filter == "" {
		return nil, false
	}
	t.m.Lock()
	defer t.m.Unlock()
	parts := strings.Split(filter, "/")
	return t.root.Find(parts)
}

func (t *Tree) addParts(n *Node, parts []string, v any) *Node {
	if len(parts) == 0 {
		return n
	}
	parent := n.FindChild(parts[0])
	if parent == nil {
		parent = NewNode(parts[0], v)
		n.AddChild(parent)
	}
	// add rest
	return t.addParts(parent, parts[1:], v)
}
