package tt

import (
	"fmt"
	"testing"
)

func ExampleTopicFilter() {
	tf := NewTopicFilter("/a/+/b/+/+")
	groups, _ := tf.Match("/a/gopher/b/is/cute")
	fmt.Println(groups)
	// output:
	// [gopher is cute]
}

func TestTopicFilter(t *testing.T) {
	okcases := []struct {
		exp    bool
		filter string
		name   string
	}{
		{
			exp:    true,
			filter: "#",
			name:   "/a/b",
		},
	}
	for _, c := range okcases {
		tf := NewTopicFilter(c.filter)
		if _, ok := tf.Match(c.name); !ok {
			t.Errorf("%s expected to match %s", c.filter, c.name)
		}
	}

	badcases := []struct {
		exp    bool
		filter string
		name   string
	}{
		{
			exp:    false,
			filter: "a/#/c",
			name:   "/a/b/c",
		},
	}
	for _, c := range badcases {
		tf := NewTopicFilter(c.filter)
		if _, ok := tf.Match(c.name); ok {
			t.Errorf("%s matched %s", c.filter, c.name)
		}
	}

}
