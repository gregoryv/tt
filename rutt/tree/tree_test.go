package tree

import (
	"fmt"
	"testing"
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
	// output:
	// [4 4 4]
}

func TestTree(t *testing.T) {
	x := NewTree()
	x.Handle("#")
	x.Handle("+/tennis/#")
	x.Handle("sport/#")
	x.Handle("sport/tennis/player1/#")
	fmt.Println(x)
}
