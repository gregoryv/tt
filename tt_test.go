package tt

import (
	"testing"

	"github.com/gregoryv/mq"
)

func Test_dump_debug_false(t *testing.T) {
	v := dump(false, mq.Pub(1, "room/a", "hello"))
	if v != "" {
		t.Error(v)
	}
}

func Test_dump_debug_true(t *testing.T) {
	v := dump(true, mq.Pub(1, "room/a", "hello"))
	if v == "" {
		t.Fail()
	}
}
