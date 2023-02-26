package ttx

import "testing"

func Test_Noops(t *testing.T) {
	NoopPub(nil, nil)
	NoopHandler(nil, nil)
}

func TestCalled(t *testing.T) {
	c := NewCalled()

	go c.Handler(nil, nil)
	<-c.Done()
}

func TestClosedConn(t *testing.T) {
	var c ClosedConn
	buf := make([]byte, 10)
	if _, err := c.Read(buf); err == nil {
		t.Fatal("Read should faild")
	}
	if _, err := c.Write(buf); err == nil {
		t.Fatal("Read should faild")
	}
}
