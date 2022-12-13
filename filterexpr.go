package tt

import (
	"fmt"
	"regexp"
	"strings"
)

func MustParseFilterExpr(v string) *FilterExpr {
	re, err := ParseFilterExpr(v)
	if err != nil {
		panic(err.Error())
	}
	return re
}

// lets try regexp
func ParseFilterExpr(v string) (*FilterExpr, error) {
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

	tf := &FilterExpr{
		re:     re,
		filter: v,
	}
	return tf, nil
}

type FilterExpr struct {
	re     *regexp.Regexp
	filter string
}

// Match topic name and return any wildcard words.
func (r *FilterExpr) Match(name string) ([]string, bool) {
	res := r.re.FindAllStringSubmatch(name, -1)
	if len(res) == 0 {
		return nil, false
	}
	// skip the entire match, ie. the first element
	return res[0][1:], true
}

func (r *FilterExpr) Filter() string {
	return r.filter
}
