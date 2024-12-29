// Package tt provides components for writing mqtt-v5 clients.
package tt

import (
	"bytes"
	"encoding/hex"

	"github.com/gregoryv/mq"
)

func dump(debug bool, p mq.Packet) string {
	if !debug {
		return ""
	}
	var buf bytes.Buffer
	p.WriteTo(&buf)
	return "\n" + hex.Dump(buf.Bytes()) + "\n"
}

func trimID(s string, width uint) string {
	if v := uint(len(s)); v > width {
		return prefixStr + s[v-width:]
	}
	return s
}

const prefixStr = "~"
