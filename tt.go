// Package tt provides components for writing mqtt-v5 clients.
package tt

import (
	"bytes"
	"encoding/hex"
	"runtime/debug"

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

func trimID(s string, width int) string {
	if v := len(s); v > width {
		return prefixStr + s[v-width:]
	}
	return s
}

const prefixStr = "~"

func short(s string, width int) string {
	if v := len(s); v > width {
		return s[v-width:]
	}
	return s
}

func revision() string {
	// this is a module so it will always work
	info, _ := debug.ReadBuildInfo()
	return findBuildValue(info.Settings, "vcs.revision")
}

func findBuildValue(bs []debug.BuildSetting, key string) string {
	for _, s := range bs {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}
