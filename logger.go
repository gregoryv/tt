package tt

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/gregoryv/mq"
)

func init() {
	log.SetFlags(0) // quiet by default
}

// NewLogger returns a logger with max id len 11
func NewLogger() *Logger {
	l := &Logger{
		Logger: log.New(ioutil.Discard, "", log.Flags()),
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

// In logs incoming packets. Log prefix is based on
// mq.ConnAck.AssignedClientID.
func (f *Logger) In(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		var msg string
		switch p := p.(type) {
		case *mq.ConnAck:
			if v := p.AssignedClientID(); v != "" {
				f.SetLogPrefix(v)
			}
		case *mq.Connect:
			msg = fmt.Sprintf("in  %v %s", p, f.remote)
		}
		if msg == "" {
			msg = fmt.Sprint("in  ", p)
		}
		// double spaces to align in/out. Usually this is not advised
		// but in here it really does aid when scanning for patterns
		// of packets.
		if f.debug {
			f.Print(msg, "\n", dumpPacket(p))
		} else {
			f.Print(msg)
		}
		err := next(ctx, p)
		if err != nil {
			f.Print(err)
		}
		// return error just incase this middleware is not the first
		return err
	}
}

// Out logs outgoing packets. Log prefix is based on
// mq.Connect.ClientID.
func (f *Logger) Out(next Handler) Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(*mq.Connect); ok {
			f.SetLogPrefix(p.ClientID())
		}
		if f.debug {
			f.Print("out ", p, "\n", dumpPacket(p))
		} else {
			f.Print("out ", p)
		}
		err := next(ctx, p)
		if err != nil {
			f.Print(err)
		}
		return err
	}
}

func (f *Logger) SetLogPrefix(v string) {
	v = newPrefix(v, f.maxLen)
	f.SetPrefix(v + " ")
}

func dumpPacket(p mq.Packet) string {
	var buf bytes.Buffer
	p.WriteTo(&buf)
	return hex.Dump(buf.Bytes())
}

// ----------------------------------------

func newPrefix(s string, width uint) string {
	if v := uint(len(s)); v > width {
		return prefixStr + s[v-width:]
	}
	return s
}

const prefixStr = "~"
