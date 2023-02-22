// Package tt provides components for writing mqtt-v5 clients.
package tt

import (
	"bytes"
	"context"
	"encoding/hex"

	"github.com/gregoryv/mq"
)

// Handler handles a mqtt control packet
type Handler func(context.Context, mq.Packet) error

func dumpPacket(p mq.Packet) string {
	var buf bytes.Buffer
	p.WriteTo(&buf)
	return hex.Dump(buf.Bytes())
}

func trimID(s string, width uint) string {
	if v := uint(len(s)); v > width {
		return prefixStr + s[v-width:]
	}
	return s
}

const prefixStr = "~"
