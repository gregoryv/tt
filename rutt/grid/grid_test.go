package grid

import (
	"sync"
	"testing"
)

func BenchmarkWindow(b *testing.B) {
	var windows = sync.Pool{
		New: func() any {
			// The Pool's New function should generally only return pointer
			// types, since a pointer can be put into the return interface
			// value without an allocation:
			return make([]int, 1000)
		},
	}
	// https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901241
	topics := []string{
		"sport/tennis/player1",
		"sport/tennis/player1/ranking",
		"sport/tennis/player1/score/wimbledon",
	}

	filters := []string{
		"sport/tennis/player1/#",
		"sport/#",
		"#",
		"+/tennis/#",
		// non matching below
		"+",
		"tennis/player1/#",
		"sport/tennis#",
	}
	win := windows.Get().([]int)

	// reset
	for i := 0; i < len(win); i++ {
		win[i] = i
	}

	for i := 0; i < b.N; i++ {
		for _, topic := range topics {
			match(0, win[:len(filters)], filters, topic)
		}
	}
	windows.Put(win)
}

// ----------------------------------------

func match(k int, window []int, filters []string, topic string) {
	for _, v := range topic {
		for j, filter := range filters {
			if window[j] == -1 {
				continue
			}
			_ = v
			_ = filter
		}
	}
}
