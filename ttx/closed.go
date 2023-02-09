package ttx

import (
	"fmt"
	"net"
)

type ClosedConn struct{}

func (c *ClosedConn) Read(_ []byte) (int, error) {
	return 0, &net.OpError{Op: "read", Err: fmt.Errorf("ttx closed conn")}
}

func (c *ClosedConn) Write(_ []byte) (int, error) {
	return 0, &net.OpError{Op: "write", Err: fmt.Errorf("ttx closed conn")}
}
