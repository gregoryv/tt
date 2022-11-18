package tt

// NewTransmitter returns a handler as a combination of the given
// handlers and or last io.Writer. Same as
//
// handler0(handler1(...Send(writerN)))
//
// or
//
// outer0(outer1(...handlerN()))
func NewTransmitter(v ...any) Handler {
	switch m := v[0].(type) {
	case Outer:
		return m.Out(NewTransmitter(v[1:]...))
	case Handler:
		return m
	default:
		panic("NewTransmitter only accepts tt.Outer | tt.Handler")
	}
}
