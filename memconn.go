package tt

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"

	"github.com/gregoryv/mq"
)

// dial opens a new in memory connection to the server, useful for testing.
func dial(s *Server) Remote {
	conn := NewMemConn()
	go s.AddConnection(context.Background(), conn.Server())
	return conn.Client()
}

// NewMemConn returns a duplex in memory connection.
func NewMemConn() *MemConn {
	fromServer, toClient := io.Pipe()
	fromClient, toServer := io.Pipe()

	c := &MemConn{
		client: &conn{
			Reader:      fromServer,
			WriteCloser: toServer,
			remote:      addr("tcp://:1234"),
		},
		server: &conn{
			Reader:      fromClient,
			WriteCloser: toClient,
			remote: addr(
				// port range 1024-2^16
				fmt.Sprintf("tcp://%s:%v", randIP(), 1024+rand.Int31n(64512)),
			),
		},
	}
	return c
}

type MemConn struct {
	*conn  // active side, set by Server() or Client() methods
	client *conn
	server *conn
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

func (c *conn) RemoteAddr() net.Addr { return c.remote }
func (c *conn) Responds(p mq.Packet) { p.WriteTo(c) }

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
