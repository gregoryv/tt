package tt

import (
	"io"
	"io/ioutil"
	"net"
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

func (t *TestConn) RemoteAddr() net.Addr {
	return t
}

// Network and String are used as implementation of net.Addr
func (t *TestConn) Network() string { return "tcp" }
func (t *TestConn) String() string  { return "testconn:0000" }

func expPanic(t *testing.T) {
	t.Helper()
	if e := recover(); e == nil {
		t.Fatal("expect panic")
	}
}
