package tt

import (
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"

	"github.com/gregoryv/mq"
)

// NewMemConn returns a duplex in memory connection.
func NewMemConn() *MemConn {
	fromServer, toClient := io.Pipe()
	fromClient, toServer := io.Pipe()

	c := &MemConn{
		server: &conn{
			Reader:      fromClient,
			WriteCloser: toClient,
			remote: addr(
				// port range 1024-2^16
				fmt.Sprintf("tcp://%s:%v", randIP(), 1024+rand.Int31n(64512)),
			),
		},
		client: &conn{
			Reader:      fromServer,
			WriteCloser: toServer,
			remote:      addr("tcp://:1234"),
		},
	}
	return c
}

type MemConn struct {
	*conn // active side, set by Server() or Client() methods

	server *conn
	client *conn
}

// Close closes both sides of the connection.
func (c *MemConn) Close() error {
	c.server.Close()
	c.client.Close()
	return nil
}

// Server returns the server side of the connection.
func (c *MemConn) Server() *MemConn {
	return &MemConn{
		conn:   c.server,
		client: c.client,
		server: c.server,
	}
}

// Client returns the client side of the connection.
func (c *MemConn) Client() *MemConn {
	return &MemConn{
		conn:   c.client,
		client: c.client,
		server: c.server,
	}
}

type conn struct {
	io.Reader
	io.WriteCloser

	remote addr
}

func (t *conn) RemoteAddr() net.Addr { return t.remote }

func (c *conn) Responds(p mq.Packet) { p.WriteTo(c) }

func (c *conn) String() string { return c.remote.String() }

func randIP() string {
	buf := make([]byte, 4)
	ip := rand.Uint32()
	binary.LittleEndian.PutUint32(buf, ip)
	return net.IP(buf).String()
}

// must be (tcp|udp)://([host]:port
type addr string

func (a addr) Network() string { return string(a)[:3] }
func (a addr) String() string  { return string(a)[7:] }
