package tree

import (
	"bytes"
	"fmt"
)

func NewNode(part string) *Node {
	return &Node{
		part: part,
	}
}

type Node struct {
	part     string
	parent   *Node
	children []*Node
}

func (n *Node) Match(parts []string, i int) []*Node {
	var filters []*Node
	switch {
	case i > len(parts)-1:
		return append(filters, n)

	case n.part == "#":
		return append(filters, n)

	case n.part != "+" && n.part != parts[i]:
		return nil
	}

	for _, child := range n.children {
		filters = append(filters, child.Match(parts, i+1)...)
	}

	return filters
}

func (n *Node) FindChild(part string) *Node {
	for _, child := range n.children {
		if child.part == part {
			return child
		}
	}
	return nil
}

func (n *Node) AddChild(c *Node) {
	c.parent = n
	n.children = append(n.children, c)
}

func (n *Node) Filter() string {
	if n.parent == nil {
		return n.part
	}
	return n.parent.Filter() + "/" + n.part
}

func (n *Node) String() string {
	var buf bytes.Buffer
	n.sprint(&buf, "")
	return buf.String()
}

func (n *Node) sprint(buf *bytes.Buffer, indent string) {
	fmt.Fprintln(buf, indent, n.part)
	for _, n := range n.children {
		n.sprint(buf, indent+"  ")
	}
}
