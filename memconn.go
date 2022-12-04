package tt

import (
	"io"
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
		},
		client: &conn{
			Reader:      fromServer,
			WriteCloser: toServer,
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
}

func (t *conn) RemoteAddr() net.Addr {
	return t
}

// Network and String are used as implementation of net.Addr
func (t *conn) Network() string { return "tcp" }
func (t *conn) String() string  { return "testconn:0000" }

func (c *conn) Responds(p mq.Packet) { p.WriteTo(c) }
