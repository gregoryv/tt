package tt

import (
	"fmt"
	"testing"
)

func ExampleTopicFilter() {
	tf := MustParseFilterExpr("/a/+/b/+/+")
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
		_, err := ParseFilterExpr(filter)
		if err != nil {
			t.Fatal(err)
		}
	}

	badcases := []string{
		"a/#/c",
		"#/",
	}
	for _, filter := range badcases {
		_, err := ParseFilterExpr(filter)
		if err == nil {
			t.Fatalf("%s should fail", filter)
		}
	}

}

func TestMustParseFilterExpr(t *testing.T) {
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
	spec := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}

	cases := []struct {
		expMatch bool
		names    []string
		*FilterExpr
	}{
		{true, spec, MustParseFilterExpr("sport/tennis/player1/#")},
		{true, spec, MustParseFilterExpr("sport/#")},
		{true, spec, MustParseFilterExpr("#")},
		{true, spec, MustParseFilterExpr("+/tennis/#")},

		{false, spec, MustParseFilterExpr("+")},
		{false, spec, MustParseFilterExpr("tennis/player1/#")},
		{false, spec, MustParseFilterExpr("sport/tennis#")},
	}

	for _, c := range cases {
		for _, name := range c.names {
			words, match := c.FilterExpr.Match(name)

			if match != c.expMatch {
				t.Errorf("%s %s exp:%v got:%v %q",
					name, c.FilterExpr.Filter(), c.expMatch, match, words,
				)
			}

			if v := c.FilterExpr.Filter(); v == "" {
				t.Error("no subscription")
			}
		}
	}

	// check String
	if v := MustParseFilterExpr("sport/#").Filter(); v != "sport/#" {
		t.Error("Route.String missing filter", v)
	}
}

func TestMustParseFilterExpr_panics(t *testing.T) {
	defer catchPanic(t)
	MustParseFilterExpr("sport/(.")
}

func catchPanic(t *testing.T) {
	if e := recover(); e == nil {
		t.Error("expected panic")
	}
}
