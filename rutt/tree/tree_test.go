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
	"bytes"
	"reflect"
	"testing"

	"github.com/gregoryv/golden"
)

func TestTree_Filters(t *testing.T) {
	x := newTestTree()
	var buf bytes.Buffer
	for _, f := range x.Filters() {
		buf.WriteString(f)
		buf.WriteString("\n")
	}
	golden.Assert(t, buf.String())
}

func newTestTree() *Tree {
	x := NewTree()
	x.AddFilter("") // should result in a noop
	x.AddFilter("#")
	x.AddFilter("+/tennis/#")
	x.AddFilter("sport/#")
	x.AddFilter("sport/tennis/player1/#")
	return x
}

func TestRouter(t *testing.T) {
	t.Run("Tree", func(t *testing.T) {
		testRouterMatch(t, NewTree())
	})
}

func testRouterMatch(t *testing.T, r Router) {
	t.Helper()
	exp := []string{
		"#",
		"+/tennis/#",
		"sport/#",
		"sport/tennis/player1/#",
	}
	for _, filter := range exp {
		r.AddFilter(filter)
	}

	topics := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}
	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			var filters []string
			for _, n := range r.Match(topic) {
				filters = append(filters, n.Filter())
			}
			if !reflect.DeepEqual(filters, exp) {
				t.Log(r)
				t.Error("\ntopic: ", topic, "matched by\n", filters, "\nexpected\n", exp)
			}
		})
	}
}

type Router interface {
	AddFilter(string)
	Match(topic string) []*Node
}

func BenchmarkTree_Match(b *testing.B) {
	benchmarkRouterMatch(b, NewTree())
}

func benchmarkRouterMatch(b *testing.B, r Router) {
	b.Helper()
	exp := []string{
		"#",
		"+/tennis/#",
		"sport/#",
		"sport/tennis/player1/#",
	}
	for _, filter := range exp {
		r.AddFilter(filter)
	}
	topics := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}
	for i := 0; i < b.N; i++ {
		for _, topic := range topics {
			r.Match(topic)
		}
	}
}
