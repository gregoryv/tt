package tt

import (
	"fmt"
	"testing"
)

func ExampleTopicFilter() {
	tf := MustParseTopicFilter("/a/+/b/+/+")
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

func TestTopicFilter_Match(t *testing.T) {
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
	spec := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}

	cases := []struct {
		expMatch bool
		names    []string
		*TopicFilter
	}{
		{true, spec, MustParseTopicFilter("sport/tennis/player1/#")},
		{true, spec, MustParseTopicFilter("sport/#")},
		{true, spec, MustParseTopicFilter("#")},
		{true, spec, MustParseTopicFilter("+/tennis/#")},

		{false, spec, MustParseTopicFilter("+")},
		{false, spec, MustParseTopicFilter("tennis/player1/#")},
		{false, spec, MustParseTopicFilter("sport/tennis#")},
	}

	for _, c := range cases {
		for _, name := range c.names {
			words, match := c.TopicFilter.Match(name)

			if match != c.expMatch {
				t.Errorf("%s %s exp:%v got:%v %q",
					name, c.TopicFilter.Filter(), c.expMatch, match, words,
				)
			}

			if v := c.TopicFilter.Filter(); v == "" {
				t.Error("no subscription")
			}
		}
	}

	// check String
	if v := MustParseTopicFilter("sport/#").Filter(); v != "sport/#" {
		t.Error("Route.String missing filter", v)
	}
}

func TestMustParseTopicFilter_panics(t *testing.T) {
	defer catchPanic(t)
	MustParseTopicFilter("sport/(.")
}
