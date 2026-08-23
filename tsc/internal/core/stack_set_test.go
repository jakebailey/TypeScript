package core_test

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"gotest.tools/v3/assert"
)

func TestStackSet(t *testing.T) {
	t.Parallel()

	var stack core.StackSet[int]
	const depth = 128
	for i := range depth {
		stack.Push(i)
		assert.Equal(t, stack.Len(), i+1)
		assert.Assert(t, stack.Has(i))
	}

	for i := depth - 1; i >= 0; i-- {
		assert.Equal(t, stack.Pop(), i)
		assert.Assert(t, !stack.Has(i))
		assert.Equal(t, stack.Len(), i)
	}

	assert.Assert(t, !stack.Has(0))
	func() {
		defer func() {
			assert.Assert(t, recover() != nil)
		}()
		stack.Pop()
	}()
}

func TestStackSetClear(t *testing.T) {
	t.Parallel()

	var stack core.StackSet[*int]
	values := make([]int, 10)
	for i := range values {
		stack.Push(&values[i])
	}

	stack.Clear()
	assert.Equal(t, stack.Len(), 0)
	for i := range values {
		assert.Assert(t, !stack.Has(&values[i]))
	}

	value := 42
	stack.Push(&value)
	assert.Assert(t, stack.Has(&value))
	assert.Equal(t, stack.Pop(), &value)
}

func TestStackSetShallowAllocations(t *testing.T) { //nolint:paralleltest
	const depth = 4

	before := testing.AllocsPerRun(100, func() {
		var stack legacyMaybeStack[int]
		pushAndPop(&stack, depth)
	})
	after := testing.AllocsPerRun(100, func() {
		var stack core.StackSet[int]
		pushAndPop(&stack, depth)
	})

	assert.Assert(t, after < before, "before = %v allocations, after = %v allocations", before, after)
}

type maybeStack interface {
	Has(int) bool
	Push(int)
	Pop() int
}

type legacyMaybeStack[T comparable] struct {
	values []T
	set    map[T]struct{}
}

func (s *legacyMaybeStack[T]) Has(value T) bool {
	_, ok := s.set[value]
	return ok
}

func (s *legacyMaybeStack[T]) Push(value T) {
	s.values = append(s.values, value)
	if s.set == nil {
		s.set = make(map[T]struct{})
	}
	s.set[value] = struct{}{}
}

func (s *legacyMaybeStack[T]) Pop() T {
	last := len(s.values) - 1
	value := s.values[last]
	s.values = s.values[:last]
	delete(s.set, value)
	return value
}

func pushAndPop[S maybeStack](stack S, depth int) {
	for i := range depth {
		stack.Push(i)
		if !stack.Has(i) {
			panic("missing pushed value")
		}
	}
	for range depth {
		stack.Pop()
	}
}

func BenchmarkMaybeStack(b *testing.B) {
	for _, test := range []struct {
		name  string
		depth int
	}{
		{name: "shallow", depth: 4},
		{name: "deep", depth: 32},
	} {
		b.Run(test.name, func(b *testing.B) {
			b.Run("before", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var stack legacyMaybeStack[int]
					pushAndPop(&stack, test.depth)
				}
			})
			b.Run("after", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					var stack core.StackSet[int]
					pushAndPop(&stack, test.depth)
				}
			})
		})
	}
}
