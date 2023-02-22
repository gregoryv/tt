package tt

import (
	"fmt"
	"testing"
)

func ExampleTopicFilter() {
	tf := mustParseTopicFilter("/a/+/b/+/+")
	groups, _ := tf.Match("/a/gopher/b/is/cute")
	fmt.Println(groups)
	// output:
	// [gopher is cute]
}

func TestParseTopicFilter(t *testing.T) {
	okcases := []string{
		"#",
	}
	for _, filter := range okcases {
		_, err := ParseTopicFilter(filter)
		if err != nil {
			t.Fatal(err)
		}
	}

	badcases := []string{
		"a/#/c",
		"#/",
		"",
	}
	for _, filter := range badcases {
		_, err := ParseTopicFilter(filter)
		if err == nil {
			t.Fatalf("%s should fail", filter)
		}
	}

}

func Test_topicFilter_Match(t *testing.T) {
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
	spec := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}

	cases := []struct {
		expMatch bool
		names    []string
		*topicFilter
	}{
		{true, spec, mustParseTopicFilter("sport/tennis/player1/#")},
		{true, spec, mustParseTopicFilter("sport/#")},
		{true, spec, mustParseTopicFilter("#")},
		{true, spec, mustParseTopicFilter("+/tennis/#")},

		{false, spec, mustParseTopicFilter("+")},
		{false, spec, mustParseTopicFilter("tennis/player1/#")},
		{false, spec, mustParseTopicFilter("sport/tennis#")},
	}

	for _, c := range cases {
		for _, name := range c.names {
			words, match := c.topicFilter.Match(name)

			if match != c.expMatch {
				t.Errorf("%s %s exp:%v got:%v %q",
					name, c.topicFilter.Filter(), c.expMatch, match, words,
				)
			}

			if v := c.topicFilter.Filter(); v == "" {
				t.Error("no subscription")
			}
		}
	}

	// check String
	if v := mustParseTopicFilter("sport/#").Filter(); v != "sport/#" {
		t.Error("Route.String missing filter", v)
	}
}

func TestMustParseTopicFilter_panics(t *testing.T) {
	defer catchPanic(t)
	mustParseTopicFilter("sport/(.")
}
