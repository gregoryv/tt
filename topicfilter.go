package tt

import (
	"fmt"
	"regexp"
	"strings"
)

func mustParseTopicFilter(v string) *topicFilter {
	re, err := ParseTopicFilter(v)
	if err != nil {
		panic(err.Error())
	}
	return re
}

func ParseTopicFilter(v string) (*topicFilter, error) {
	if len(v) == 0 {
		return nil, fmt.Errorf("empty filter")
	}
	if i := strings.Index(v, "#"); i >= 0 && i < len(v)-1 {
		// i.e. /a/#/b
		return nil, fmt.Errorf("%q # not allowed there", v)
	}

	// build regexp
	var expr string
	if v == "#" {
		expr = "^(.*)$"
	} else {
		expr = strings.ReplaceAll(v, "+", `([\w\s]+)`)
		expr = strings.ReplaceAll(expr, "/#", `(.*)`)
		expr = "^" + expr + "$"
	}
	re, err := regexp.Compile(expr)
	if err != nil {
		return nil, err
	}

	tf := &topicFilter{
		re:     re,
		filter: v,
	}
	return tf, nil
}

// topicFilter is used to match topic names as specified in [4.7 Topic
// Names and Topic Filters]
//
// [4.7 Topic Names and Topic Filters]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
type topicFilter struct {
	re     *regexp.Regexp
	filter string
}

// Match topic name and return any wildcard words.
func (r *topicFilter) Match(name string) ([]string, bool) {
	res := r.re.FindAllStringSubmatch(name, -1)
	if len(res) == 0 {
		return nil, false
	}
	// skip the entire match, ie. the first element
	return res[0][1:], true
}

func (r *topicFilter) Filter() string {
	return r.filter
}
