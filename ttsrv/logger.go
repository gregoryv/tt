package ttsrv

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/gregoryv/mq"
	"github.com/gregoryv/tt"
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
	maxLen   uint
	remote   string
	clientID string
}

func (l *Logger) SetDebug(v bool)      { l.debug = v }
func (l *Logger) SetRemote(v string)   { l.remote = v }
func (l *Logger) SetClientID(v string) { l.remote = v }

// SetMaxIDLen configures the logger to trim the client id to number of
// characters. Use 0 to not trim.
func (l *Logger) SetMaxIDLen(max uint) {
	l.maxLen = max
}

// In logs incoming packets.
func (f *Logger) In(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(*mq.Connect); ok {
			f.clientID = trimID(p.ClientID(), f.maxLen)
		}
		// double spaces to align in/out. Usually this is not advised
		// but in here it really does aid when scanning for patterns
		// of packets.
		msg := fmt.Sprintf("in  %v <- %s:%s", p, f.remote, f.clientID)		
		if f.debug {
			f.Print(msg, "\n", dumpPacket(p))
		} else {
			f.Print(msg)
		}
		err := next(ctx, p)
		if err != nil {
			f.Print(err)
		}
		return err
	}
}

// Out logs outgoing packets. Log prefix is based on
// mq.Connect.ClientID.
func (f *Logger) Out(next tt.Handler) tt.Handler {
	return func(ctx context.Context, p mq.Packet) error {
		if p, ok := p.(*mq.ConnAck); ok {
			if v := p.AssignedClientID(); v != "" {
				f.clientID = trimID(v, f.maxLen)
			}
		}
		if f.debug {
			f.Print("out ", p, "\n", dumpPacket(p))
		} else {
			f.Printf("out %v -> %s:%s", p, f.remote, f.clientID)
		}
		err := next(ctx, p)
		if err != nil {
			f.Print(err)
		}
		return err
	}
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
