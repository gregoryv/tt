package tt

import (
	"context"
	"io"
	"sync"

	"github.com/gregoryv/mq"
)

// Send returns a handler that synchronizes writes of packets to the
// underlying writer. Safe for concurrent calls.
func Send(w io.Writer) Handler {
	var m sync.Mutex
	return func(ctx context.Context, p mq.Packet) error {
		m.Lock()
		_, err := p.WriteTo(w)
		m.Unlock()
		return err
	}
}
