package tt

import (
	"io/ioutil"
	"testing"
)

func TestTransmitter(t *testing.T) {
	var (
		pool   = NewIDPool(5)
		logger = NewLogger(LevelInfo)
	)
	_ = NewTransmitter(pool, logger, Send(ioutil.Discard))
}
