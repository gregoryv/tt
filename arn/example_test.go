package arn_test

import (
	"fmt"

	"github.com/gregoryv/tt/arn"
)

func Example_tree() {
	x := arn.NewTree()
	x.AddFilter("#")
	x.AddFilter("#")
	x.AddFilter("#")
	x.AddFilter("+/tennis/#")
	x.AddFilter("sport/#")
	x.AddFilter("sport/tennis/player1/#")
	fmt.Println("filters:", x.Filters())

	var result []*arn.Node
	topic := "sport/golf"
	x.Match(&result, topic)
	fmt.Println("topic:", topic)
	for _, n := range result {
		fmt.Println(n.Filter())
	}
	// output:
	// filters: [# +/tennis/# sport/# sport/tennis/player1/#]
	// topic: sport/golf
	// #
	// sport/#
}
