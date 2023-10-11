package tree

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
)

func ExampleTree() {
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
	t := NewTree()
	t.Handle("#")
	t.Handle("+/tennis/#")
	t.Handle("sport/#")
	t.Handle("sport/tennis/player1/#")

	topics := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}
	var res []int
	for _, topic := range topics {
		res = append(res, t.Route(topic))
	}
	fmt.Println(res)
	fmt.Println(t.root)
	// output:
	// [4 4 4]
}

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
		n.Next = append(n.Next, x)
		t.handle(filter[i+1:], x)
		return
	}
	x := n.Find(filter)
	if x == nil {
		x = NewNode(filter)
	}
	n.Next = append(n.Next, x)
}

func (t *Tree) Route(topic string) int {
	return t.root.route(topic)
}

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
	End  // todo ie. sub scribing clients
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

type End interface{}
