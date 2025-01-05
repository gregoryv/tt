/*
Load tests are used to flush out CPU and memory bottlenecks.
Profile reports are generated.
*/

package tt

import (
	"context"
	"errors"
	"net"
	"strconv"
	"testing"

	"github.com/gregoryv/mq"
)

func TestLoad(t *testing.T) {
	srv := NewServer()
	conn, _ := testServer(t, srv) // this calls starts one connection
	defer mq.NewDisconnect().WriteTo(conn)

	total := 10_000 - 1 // minus one already started above
	var connects int
	ctx := context.Background()
	for i := 0; i < total; i++ {
		conn, servConn := net.Pipe()
		go srv.Handle(ctx, servConn)

		p := mq.NewConnect()
		p.SetClientID("load" + strconv.Itoa(i))
		p.SetReceiveMax(1)
		p.SetKeepAlive(uint16(300))
		p.WriteTo(conn)

		if err := expectConnAck(conn); err != nil {
			t.Fatal(err)
		}
		connects++
		defer mq.NewDisconnect().WriteTo(conn)
	}

	if v := connects; v != total {
		t.Errorf("%v connects", v)
	}
}

func expectConnAck(conn Connection) error {
	a, err := mq.ReadPacket(conn)
	if err != nil {
		return err
	}
	ack := a.(*mq.ConnAck)
	switch ack.ReasonCode() {
	case mq.Success:
		return nil
	default:
		return errors.New(ack.ReasonString())
	}
}
