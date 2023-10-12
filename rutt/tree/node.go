package tree

import (
	"bytes"
	"fmt"
	"strings"
)

func (n *Node) route(topic string) int {
	if topic == "" && n.part == "" {
		return 0
	}
	i := strings.Index(topic, "/")
	if i > 0 {
		part := topic[:i]
		_ = part // todo
	}
	return 0
}

func NewNode(part string) *Node {
	return &Node{
		part: part,
	}
}

type Node struct {
	part string
	Next []*Node
}

func (n *Node) Find(part string) *Node {
	for _, next := range n.Next {
		if next.part == part {
			return next
		}
	}
	return nil
}

func (n *Node) String() string {
	var buf bytes.Buffer
	n.sprint(&buf, "")
	return buf.String()
}

func (n *Node) sprint(buf *bytes.Buffer, indent string) {
	fmt.Fprintln(buf, indent, n.part)
	for _, n := range n.Next {
		n.sprint(buf, indent+"  ")
	}
}
