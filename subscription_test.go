package tt

import (
	"testing"
)

func TestSubscription(t *testing.T) {
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
	spec := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}

	cases := []struct {
		expMatch bool
		names    []string
		*Subscription
		expWords []string
	}{
		{true, spec, NewSubscription("sport/tennis/player1/#"), nil},
		{true, spec, NewSubscription("sport/#"), nil},
		{true, spec, NewSubscription("#"), nil},
		{true, spec, NewSubscription("+/tennis/#"), []string{"sport"}},

		{true, []string{"a/b/c"}, NewSubscription("a/+/+"), []string{"b", "c"}},
		{true, []string{"a/b/c"}, NewSubscription("a/+/c"), []string{"b"}},

		{false, spec, NewSubscription("+"), nil},
		{false, spec, NewSubscription("tennis/player1/#"), nil},
		{false, spec, NewSubscription("sport/tennis#"), nil},
	}

	for _, c := range cases {
		for _, name := range c.names {
			words, match := c.Subscription.Match(name)

			if !equal(words, c.expWords) || match != c.expMatch {
				t.Errorf("%s %s exp:%v got:%v %q",
					name, c.Subscription, c.expMatch, match, words,
				)
			}

			if v := c.Subscription.Filter(); v == "" {
				t.Error("no subscription")
			}
		}
	}

	// check String
	if v := NewSubscription("sport/#").String(); v != "sport/#" {
		t.Error("Route.String missing filter", v)
	}
}

func equal[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
