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

func (t *Tree) Match(topic string) []*Node {
	parts := strings.Split(topic, "/")

	var filters []*Node
	for _, child := range t.root.children {
		filters = append(filters, child.Match(parts, 0)...)
	}

	return filters
}

func (t *Tree) AddFilter(filter string) {
	if filter == "" {
		return
	}
	t.m.Lock()
	defer t.m.Unlock()
	parts := strings.Split(filter, "/")
	t.addParts(t.root, parts)
	// t.root is just a virtual parent
	for _, top := range t.root.children {
		top.parent = nil
	}
}

func (t *Tree) addParts(n *Node, parts []string) {
	if len(parts) == 0 {
		return
	}
	parent := n.FindChild(parts[0])
	if parent == nil {
		parent = NewNode(parts[0])
		n.AddChild(parent)
	}
	// add rest
	t.addParts(parent, parts[1:])
}
