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
	_ = NewTransmitter(pool, logger, Send(ioutil.Discard))
	_ = NewTransmitter(pool.Out, logger, Send(ioutil.Discard))
	_ = NewTransmitter(pool, logger, NoopHandler)
}

func TestNewTransmitter_panics(t *testing.T) {
	t.Run("Handler func", func(t *testing.T) {
		defer expPanic(t)
		NewTransmitter(1, NoopHandler)
	})

	t.Run("Handler func", func(t *testing.T) {
		defer expPanic(t)
		NewTransmitter(NoopHandler, 1)
	})

	t.Run("", func(t *testing.T) {
		defer expPanic(t)
		m := NewIDPool(3)
		NewTransmitter(m.Out, m)
	})

	t.Run("", func(t *testing.T) {
		defer expPanic(t)
		m := NewIDPool(3)
		NewTransmitter(m, 1)
	})
}
