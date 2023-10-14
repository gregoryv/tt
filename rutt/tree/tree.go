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

func (t *Tree) AddFilter(filter string) {
	if filter == "" {
		return
	}
	t.m.Lock()
	defer t.m.Unlock()
	parts := strings.Split(filter, "/")
	t.addParts(t.root, parts)
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
