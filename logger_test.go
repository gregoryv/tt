package tt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt/ttx"
)

func ExampleLogger_In() {
	l := NewLogger()
	l.SetOutput(os.Stdout)
	l.SetRemote("1.2.3.4:0001")

	l.In(ttx.NoopHandler)(nil, mq.Pub(0, "a/b", "gopher"))

	// output:
	// in  PUBLISH ---- p0 a/b 14 bytes
}

func ExampleLogger_Out() {
	l := NewLogger()
	l.SetOutput(os.Stdout)

	p := mq.Pub(0, "a/b", "gopher")
	l.Out(ttx.NoopHandler)(nil, p)

	// output:
	// out PUBLISH ---- p0 a/b 14 bytes
}

func ExampleLogger_DumpPacket() {
	l := NewLogger()
	l.SetOutput(os.Stdout)
	l.SetDebug(true)

	p := mq.Pub(0, "a/b", "gopher")
	l.In(ttx.NoopHandler)(nil, p)

	// output:
	// in  PUBLISH ---- p0 a/b 14 bytes
	// 00000000  30 0c 00 03 61 2f 62 00  67 6f 70 68 65 72        |0...a/b.gopher|
}

func ExampleLogger_SetMaxIDLen() {
	l := NewLogger()
	l.SetOutput(os.Stdout)
	l.SetRemote("1.2.3.4:0001")
	l.SetMaxIDLen(6)
	{
		p := mq.NewConnect()
		p.SetClientID("short")
		l.Out(ttx.NoopHandler)(nil, p)
	}
	{
		p := mq.NewConnAck()
		p.SetAssignedClientID("1bbde752-5161-11ed-a94b-675e009b6f46")
		l.In(ttx.NoopHandler)(nil, p)
	}
	// output:
	// short out CONNECT ---- -------- MQTT5 short 0s 20 bytes
	// ~9b6f46 in  CONNACK ---- -------- 1bbde752-5161-11ed-a94b-675e009b6f46 44 bytes
}

func ExampleLogger_errors() {
	l := NewLogger()
	l.SetOutput(os.Stdout)

	broken := func(context.Context, mq.Packet) error {
		return fmt.Errorf("broken")
	}
	p := mq.Pub(0, "a/b", "gopher")
	l.In(broken)(nil, p)
	l.Out(broken)(nil, p)
	// output:
	// in  PUBLISH ---- p0 a/b 14 bytes
	// broken
	// out PUBLISH ---- p0 a/b 14 bytes
	// broken
}

func BenchmarkLogger_Out(b *testing.B) {
	l := NewLogger()
	p := mq.NewConnect()
	p.SetClientID("1bbde752-5161-11ed-a94b-675e009b6f46")
	ctx := context.Background()

	b.Run("Out", func(b *testing.B) {
		out := l.Out(ttx.NoopHandler)
		for i := 0; i < b.N; i++ {
			out(ctx, p)
		}
	})

	b.Run("In", func(b *testing.B) {
		in := l.In(ttx.NoopHandler)
		for i := 0; i < b.N; i++ {
			in(ctx, p)
		}
	})
}

func TestLogger(t *testing.T) {
	l := NewLogger()
	var buf bytes.Buffer
	l.SetOutput(&buf)
	cid := "1bbde752-5161-11ed-a94b-675e009b6f46"
	p := mq.NewConnect()
	p.SetClientID(cid)

	// trimmed client id
	l.Out(ttx.NoopHandler)(nil, p)
	if v := buf.String(); !strings.HasPrefix(v, "~75e009b6f46") {
		t.Error(v)
	}

	// subsequent
	l.Out(ttx.NoopHandler)(nil, p)
	if v := buf.String(); !strings.HasPrefix(v, "~75e009b6f46") {
		t.Error(v)
	}

	// debug
	l = NewLogger()
	l.SetOutput(&buf)
	l.SetDebug(true)

	buf.Reset()
	l.Out(ttx.NoopHandler)(nil, p)
	if v := buf.String(); !strings.Contains(v, "|f46|") {
		t.Error(v)
	}
}
