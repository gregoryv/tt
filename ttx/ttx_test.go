package ttx

import "testing"

func Test_NoopPub(t *testing.T) {
	if err := NoopPub(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func Test_NoopHandler(t *testing.T) {
	if err := NoopHandler(nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCalled(t *testing.T) {
	c := NewCalled()

	go c.Handler(nil, nil)
	<- c.Done()
}
