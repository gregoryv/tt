package spec

import (
	"errors"
	"fmt"
)

// VerifyFilterMatching verifies the given implementation against the
// given rules. If no rules are given, default set of rules is used.
func VerifyFilterMatching(impl MatchFunc, rules ...RuleTopic) error {
	var all []error
	if len(rules) == 0 {
		rules = RulesTopic
	}
	for _, topic := range rules {
		if err := topic.Verify(impl); err != nil {
			all = append(all, err)
		}
	}
	return errors.Join(all...)
}

var RulesTopic = []RuleTopic{
	{true, "#", "/"},
	{true, "#", "word"},
	{true, "#", "a/b"},
	{true, "a/#", "a/b"},
	{true, "a/#", "a"},

	{true, "a/+/#", "a/b/c"},
	{true, "a/+/#", "a/b"},
	{true, "a/+/+", "a/b/c"},

	{true, "word", "word"},
	{true, "$sys", "$sys"},
	{true, "a/b", "a/b"},
	{true, "hello/ Åke/", "hello/ Åke/"},

	{false, "#", "$sys"},
	{false, "+", "$sys"},
	{false, "a/b/+/#", "a/b"},
	{false, "b/+/#", "a/b/c"},
	{false, "a/+/+", "a/b/c/d"},
	{false, "a/+/+", "b/c/d"},
	{false, "a/+/+", "a/b"},
	{false, "a/B", "a/b"},
	{false, "/a", "a"},
	{false, "a", "/a"},
}

type RuleTopic struct {
	Exp    bool
	Filter string
	Name   string
}

func (r *RuleTopic) Verify(fn MatchFunc) error {
	got := fn(r.Filter, r.Name)
	if r.Exp && !got {
		return fmt.Errorf("%s should match %s", r.Filter, r.Name)
	}
	if !r.Exp && got {
		return fmt.Errorf("%s should NOT match %s", r.Filter, r.Name)
	}
	return nil
}

type MatchFunc func(filter, name string) bool
