package tt

import (
	"testing"

	"github.com/gregoryv/tt/spec"
)

func Test_Match(t *testing.T) {
	if err := spec.VerifyFilterMatching(match); err != nil {
		t.Error(err)
	}
}

func Benchmark_match(b *testing.B) {
	for i := 0; i < b.N; i++ {
		spec.VerifyFilterMatching(match)
	}
}
