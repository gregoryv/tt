package tt

import (
	"io"
	"io/ioutil"
	"testing"

	"github.com/gregoryv/mq"
)

// Dial returns a test connection where writes are discarded. The
// returned writer is used to inject responses from the connected
// destiantion.
func Dial() *TestConn {
	fromServer, toClient := io.Pipe()
	toServer := ioutil.Discard
	c := &TestConn{
		Reader: fromServer,
		Writer: toServer,
		client: toClient,
	}
	return c
}

type TestConn struct {
	io.Reader // incoming from server
	io.Writer // outgoing to server

	client io.Writer
}

func (t *TestConn) Responds(p mq.Packet) {
	p.WriteTo(t.client)
}

func expPanic(t *testing.T) {
	t.Helper()
	if e := recover(); e == nil {
		t.Fatal("expect panic")
	}
}
