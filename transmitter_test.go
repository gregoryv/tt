package tt

import (
	"bytes"
	"testing"
)

func TestTransmitter(t *testing.T) {
	var (
		pool   = NewIDPool(5)
		logger = NewLogger(LevelInfo)
		buf    bytes.Buffer
	)
	_ = NewTransmitter(pool, logger, &buf)
}
