package ttx

import "testing"

func Test_NoopPub(t *testing.T) {
	if err := NoopPub(nil, nil); err != nil {
		t.Fatal(err)
	}
}
