package arn

import (
	"strings"
	"sync"
)

// NewTree returns a new empty topic filter tree.
func NewTree() *Tree {
	return &Tree{
		root: NewNode(""),
	}
}

type Tree struct {
	m    sync.RWMutex
	root *Node
}

// Match populates result with leaf nodes matching the given topic
// name.
func (t *Tree) Match(result *[]*Node, topic string) {
	t.m.RLock()
	defer t.m.RUnlock()
	parts := strings.Split(topic, "/")
	for _, child := range t.root.children {
		child.Match(result, parts, 0)
	}
}

// Filters returns all topic filters in the tree
func (t *Tree) Filters() []string {
	var filters []string
	for _, l := range t.Leafs() {
		filters = append(filters, l.Filter())
	}
	return filters
}

// Leafs returns all topic filters in the tree as nodes.
func (t *Tree) Leafs() []*Node {
	t.m.RLock()
	defer t.m.RUnlock()
	return t.root.Leafs()
}

// AddFilter adds the topic filter to the tree. Returns existing or
// new node for that filter. Returns nil on empty filter.
func (t *Tree) AddFilter(filter string) *Node {
	if filter == "" {
		return nil
	}
	t.m.Lock()
	defer t.m.Unlock()
	parts := strings.Split(filter, "/")
	n := t.addParts(t.root, parts)
	// t.root is just a virtual parent
	for _, top := range t.root.children {
		top.parent = nil
	}
	return n
}

// Find returns node matching the given filter. If not found, nil and
// false is returned.
func (t *Tree) Find(filter string) (*Node, bool) {
	if filter == "" {
		return nil, false
	}
	t.m.Lock()
	defer t.m.Unlock()
	parts := strings.Split(filter, "/")
	return t.root.Find(parts)
}

func (t *Tree) addParts(n *Node, parts []string) *Node {
	if len(parts) == 0 {
		return n
	}
	parent := n.FindChild(parts[0])
	if parent == nil {
		parent = NewNode(parts[0])
		n.AddChild(parent)
	}
	// add rest
	return t.addParts(parent, parts[1:])
}
