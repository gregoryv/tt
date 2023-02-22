package tt

import (
	"bytes"
	"encoding/hex"
	"log"
	"os"

	"github.com/gregoryv/mq"
)

func init() {
	log.SetFlags(0) // quiet by default
}

// NewLogger returns a client side logger with max id len 11.
func NewLogger() *Logger {
	l := &Logger{
		Logger: log.New(os.Stderr, "", log.Flags()),
	}
	l.SetMaxIDLen(11)
	return l
}

type Logger struct {
	*log.Logger
	debug bool
	// client ids
	maxLen uint
	remote string
}

func (l *Logger) SetDebug(v bool)    { l.debug = v }
func (l *Logger) SetRemote(v string) { l.remote = v }

// SetMaxIDLen configures the logger to trim the client id to number of
// characters. Use 0 to not trim.
func (l *Logger) SetMaxIDLen(max uint) {
	l.maxLen = max
}

func (f *Logger) SetLogPrefix(v string) {
	v = trimID(v, f.maxLen)
	f.SetPrefix(v + " ")
}

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
