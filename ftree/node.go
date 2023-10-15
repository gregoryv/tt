package ftree

func NewNode(txt string, v any) *Node {
	return &Node{
		Value: v,
		txt:   txt,
	}
}

type Node struct {
	Value any

	txt      string
	parent   *Node
	children []*Node
}

func (n *Node) Match(filters *[]*Node, parts []string, i int) {
	switch {
	case i > len(parts)-1:
		*filters = append(*filters, n)
		return

	case n.txt == "#":
		*filters = append(*filters, n)
		return

	case n.txt != "+" && n.txt != parts[i]:
		return
	}

	for _, child := range n.children {
		child.Match(filters, parts, i+1)
	}
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
