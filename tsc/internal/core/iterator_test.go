package core_test

import (
	"iter"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"gotest.tools/v3/assert"
)

func TestSeqSearchStopsEarly(t *testing.T) {
	t.Parallel()

	visited := 0
	seq := func(yield func(int) bool) {
		for i := range 10 {
			visited++
			if !yield(i) {
				return
			}
		}
	}

	assert.Assert(t, core.SomeSeq(iter.Seq[int](seq), func(value int) bool { return value == 2 }))
	assert.Equal(t, visited, 3)

	visited = 0
	assert.Equal(t, core.FindSeq(iter.Seq[int](seq), func(value int) bool { return value == 4 }), 4)
	assert.Equal(t, visited, 5)
}
