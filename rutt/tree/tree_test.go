/*
Filters

	#
	+/tennis/#
	sport/#
	sport/tennis/player1/#

should all match the following topics

	sport/tennis/player1
	sport/tennis/player1/ranking
	sport/tennis/player1/score/wimbledon

https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
*/
package tree

import (
	"fmt"
	"testing"
)

func TestTree(t *testing.T) {
	x := NewTree()
	x.AddFilter("")
	x.AddFilter("#")
	x.AddFilter("+/tennis/#")
	x.AddFilter("sport/#")
	x.AddFilter("sport/tennis/player1/#")
	fmt.Println(x)
}
