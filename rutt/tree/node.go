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
	children []*Node
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
	n.children = append(n.children, c)
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
