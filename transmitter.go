package tt

import (
	"context"

	"github.com/gregoryv/mq"
)

// NewTransmitter returns a handler as a combination of the given
// handlers and or last io.Writer. Same as
//
//   middleware0.Out(middleware1.Out(...handlerN))
//
func NewTransmitter(v ...any) Handler {
	switch m := v[0].(type) {
	case Outer:
		return m.Out(NewTransmitter(v[1:]...))
	case Handler:
		return m
	case func(context.Context, mq.Packet) error:
		return m
	default:
		panic("NewTransmitter only accepts tt.Outer | tt.Handler")
	}
}
