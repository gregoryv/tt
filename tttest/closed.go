package tttest

import "io"

type ClosedConn struct{}

func (c *ClosedConn) Read(_ []byte) (int, error) {
	return 0, io.EOF
}

func (c *ClosedConn) Write(_ []byte) (int, error) {
	return 0, io.EOF
}
