// Package spec implements implementation verifiers against the MQTTv5 specification
//
// [4.7 Topic names and filters]
//
// [4.7 Topic names and filters]: https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
package spec

import (
	"errors"
	"fmt"
)

// VerifyFilterFormat verifies topic filter implementation against the
// given rules. If no rules are given, default set of rules are used.
func VerifyFilterFormat(fn func(filter string) bool, rules ...RuleFilterFormat) error {
	var all []error
	if len(rules) == 0 {
		rules = RulesFilterFormat
	}
	for _, rule := range rules {
		if err := rule.Verify(fn); err != nil {
			all = append(all, err)
		}
	}
	return errors.Join(all...)
}

var RulesFilterFormat = []RuleFilterFormat{
	{true, "#"},
	{true, "/#"},
	{true, "a/b/#"},
	{true, "+"},
	{true, "+/"},
	{true, "+/+"},
	{true, "/+/+"},
	{true, "a/+/c/+"},

	{false, " #"},
	{false, "#/"},
	{false, " +"},
	{false, "++/+"},
	{false, "/a+"},
}

type RuleFilterFormat struct {
	Exp    bool
	Filter string
}

func (r *RuleFilterFormat) Verify(fn func(string) bool) error {
	got := fn(r.Filter)
	if r.Exp && !got {
		return fmt.Errorf("%q should be acceptable format", r.Filter)
	}
	if !r.Exp && got {
		return fmt.Errorf("%q should NOT be acceptable format", r.Filter)
	}
	return nil
}

// ----------------------------------------

// VerifyFilterMatching verifies match func against the given
// rules. If no rules are given, default set of rules is used.
func VerifyFilterMatching(fn MatchFunc, rules ...RuleFilterMatch) error {
	var all []error
	if len(rules) == 0 {
		rules = RulesTopic
	}
	for _, topic := range rules {
		if err := topic.Verify(fn); err != nil {
			all = append(all, err)
		}
	}
	return errors.Join(all...)
}

var RulesTopic = []RuleFilterMatch{
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
	{true, "+/+", "/finance"},
	{true, "/+", "/finance"},

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
	{false, "+", "/finance"},
}

type RuleFilterMatch struct {
	Exp    bool
	Filter string
	Name   string
}

func (r *RuleFilterMatch) Verify(fn MatchFunc) error {
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
