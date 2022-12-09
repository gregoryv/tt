package tt

import (
	"testing"
)

func TestSubscription(t *testing.T) {
	// todo move this on TopicFilter instead
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
	}{
		{true, spec, MustNewSubscription("sport/tennis/player1/#")},
		{true, spec, MustNewSubscription("sport/#")},
		{true, spec, MustNewSubscription("#")},
		{true, spec, MustNewSubscription("+/tennis/#")},

		{true, []string{"a/b/c"}, MustNewSubscription("a/+/+")},
		{true, []string{"a/b/c"}, MustNewSubscription("a/+/c")},

		{false, spec, MustNewSubscription("+")},
		{false, spec, MustNewSubscription("tennis/player1/#")},
		{false, spec, MustNewSubscription("sport/tennis#")},
	}

	for _, c := range cases {
		for _, name := range c.names {
			words, match := c.Subscription.Match(name)

			if match != c.expMatch {
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
	if v := MustNewSubscription("sport/#").String(); v != "sport/#" {
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
