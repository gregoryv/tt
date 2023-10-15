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
package ftree

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/gregoryv/golden"
)

func TestTree_NoMatch(t *testing.T) {
	x := newTestTree()
	x.AddFilter("garage/+", nil)

	var result []*Node
	topic := "store/fruit/apple"
	x.Match(&result, topic)
	if len(result) != 1 {
		t.Log("filters: ", x.Filters())
		t.Errorf("%s should only match one filter", topic)
	}
}

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
	x.AddFilter("", nil) // should result in a noop
	x.AddFilter("#", nil)
	x.AddFilter("+/tennis/#", nil)
	x.AddFilter("sport/#", nil)
	x.AddFilter("sport/tennis/player1/#", nil)
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
		r.AddFilter(filter, nil)
	}

	topics := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}
	var filters []string
	var result []*Node
	for _, topic := range topics {
		t.Run(topic, func(t *testing.T) {
			result = result[:0] // reset
			r.Match(&result, topic)
			filters = filters[:0] // reset
			for _, n := range result {
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
	AddFilter(string, any)
	Match(result *[]*Node, topic string)
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
		r.AddFilter(filter, nil)
	}
	topics := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}
	var result []*Node // could make result pooled
	for i := 0; i < b.N; i++ {
		for _, topic := range topics {
			result = result[:0] // reset
			r.Match(&result, topic)
		}
	}
}
