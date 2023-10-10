package grid

import "testing"

func BenchmarkWindow(b *testing.B) {
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

	for i := 0; i < b.N; i++ {
		for _, topic := range topics {
			match(filters, topic)
		}
	}
}

func match(subscriptions []string, topic string) {
	_ = topic
}
