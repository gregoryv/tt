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

// VerifyTopicNameFormat returns error if below requirements are not
// implemented.
//
//   - The wildcard characters can be used in Topic Filters, but MUST NOT
//     be used within a Topic Name [MQTT-4.7.0-1].
func VerifyTopicNameFormat(fn func(filter string) bool) error {
	var all []error
	for _, r := range RulesTopicNameFormat {
		got := fn(r.Name)
		if r.Exp && !got {
			all = append(all, fmt.Errorf("topic name %q MUST be accepted", r.Name))
			continue
		}
		if !r.Exp && got {
			all = append(all, fmt.Errorf("topic name %q MUST NOT be accepted", r.Name))
		}
	}
	if len(all) > 0 {
		all = append(all, fmt.Errorf(`The wildcard characters can be used in Topic Filters,
but MUST NOT be used within a Topic Name [MQTT-4.7.0-1]`))
	}

	return errors.Join(all...)
}

var RulesTopicNameFormat = []struct {
	Exp  bool
	Name string
}{
	{true, "/"},
	{true, "a/b"},
	{true, "a/b/ "},

	{false, "#"},
	{false, "/#"},
	{false, "+"},
	{false, "/+"},
	{false, "/a/+/c"},
}

// ----------------------------------------

// VerifyFilterFormat verifies topic filter implementation against the
// given rules.
func VerifyFilterFormat(fn func(filter string) bool) error {
	var all []error
	for _, r := range RulesFilterFormat {
		got := fn(r.Filter)
		if r.Exp && !got {
			all = append(all, fmt.Errorf("%q should be acceptable format", r.Filter))
			continue
		}
		if !r.Exp && got {
			all = append(all, fmt.Errorf("%q should NOT be acceptable format", r.Filter))
		}
	}
	return errors.Join(all...)
}

var RulesFilterFormat = []struct {
	Exp    bool
	Filter string
}{
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

// ----------------------------------------

// VerifyFilterMatching verifies match func against the given
// rules.
func VerifyFilterMatching(fn func(filter, name string) bool) error {
	var all []error
	for _, r := range RulesFilterMatching {
		got := fn(r.Filter, r.Name)
		if r.Exp && !got {
			all = append(all, fmt.Errorf("%s should match %s", r.Filter, r.Name))
			continue
		}
		if !r.Exp && got {
			all = append(all, fmt.Errorf("%s should NOT match %s", r.Filter, r.Name))
		}
	}
	return errors.Join(all...)
}

var RulesFilterMatching = []struct {
	Exp    bool
	Filter string
	Name   string
}{
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
