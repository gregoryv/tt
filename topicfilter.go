package tt

import (
	"fmt"
	"regexp"
	"strings"
)

func MustParseTopicFilter(v string) *TopicFilter {
	re, err := ParseTopicFilter(v)
	if err != nil {
		panic(err.Error())
	}
	return re
}

// lets try regexp
func ParseTopicFilter(v string) (*TopicFilter, error) {
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

	tf := &TopicFilter{
		re:     re,
		filter: v,
	}
	return tf, nil
}

type TopicFilter struct {
	re     *regexp.Regexp
	filter string
}

// Match topic name and return any wildcard words.
func (r *TopicFilter) Match(name string) ([]string, bool) {
	res := r.re.FindAllStringSubmatch(name, -1)
	if len(res) == 0 {
		return nil, false
	}
	// skip the entire match, ie. the first element
	return res[0][1:], true
}
