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
