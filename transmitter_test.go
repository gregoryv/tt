package tt

import (
	"io/ioutil"
	"testing"
)

func TestTransmitter(t *testing.T) {
	var (
		pool   = NewIDPool(5)
		logger = NewLogger()
	)
	_ = CombineOut(Send(ioutil.Discard), pool, logger)
}
