package tt

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gregoryv/mq"
)

func TestLogger(t *testing.T) {
	l := NewLogger()
	var buf bytes.Buffer
	l.SetOutput(&buf)
	cid := "1bbde752-5161-11ed-a94b-675e009b6f46"
	l.SetLogPrefix(cid)

	p := mq.NewConnect()
	p.SetClientID(cid)

	// trimmed client id
	l.Print(p)
	if v := buf.String(); !strings.HasPrefix(v, "~75e009b6f46") {
		t.Error(v)
	}
}
