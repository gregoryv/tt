package arn

// NewNode returns a new node using txt as level value. E.g. +, # or a
// word.
func NewNode(txt string) *Node {
	return &Node{
		txt: txt,
	}
}

type Node struct {
	// Value is controlled by the caller.
	Value any

	txt      string
	parent   *Node
	children []*Node
}

func (n *Node) match(result *[]*Node, parts []string, i int) {
	switch {
	case i > len(parts)-1:
		*result = append(*result, n)
		return

	case n.txt == "#":
		if parts[0][0] != '$' {
			// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901246
			*result = append(*result, n)
		}
		return

	case n.txt != "+" && n.txt != parts[i]:
		return
	}

	for _, child := range n.children {
		child.match(result, parts, i+1)
	}
}

func (n *Node) Find(parts []string) (*Node, bool) {
	if len(parts) == 0 {
		return n, true
	}
	c := n.FindChild(parts[0])
	if c == nil {
		return nil, false
	}
	return c.Find(parts[1:])
}

func (n *Node) FindChild(txt string) *Node {
	for _, child := range n.children {
		if child.txt == txt {
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
		return n.txt
	}
	return n.parent.Filter() + "/" + n.txt
}

func (n *Node) Leafs() []*Node {
	var leafs []*Node
	for _, c := range n.children {
		if c.IsLeaf() {
			leafs = append(leafs, c)
			continue
		}
		leafs = append(leafs, c.Leafs()...)
	}
	return leafs
}

func (n *Node) IsLeaf() bool {
	return len(n.children) == 0
}
