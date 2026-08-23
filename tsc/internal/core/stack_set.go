package core

import "slices"

const stackSetMapThreshold = 8

// StackSet combines stack ordering with set membership checks.
// Small stacks use a linear scan and materialize a map only after the threshold.
type StackSet[T comparable] struct {
	values []T
	set    map[T]struct{}
}

func (s *StackSet[T]) Has(value T) bool {
	if s.set == nil {
		return slices.Contains(s.values, value)
	}
	_, ok := s.set[value]
	return ok
}

func (s *StackSet[T]) Push(value T) {
	s.values = append(s.values, value)
	if s.set == nil && len(s.values) > stackSetMapThreshold {
		s.set = make(map[T]struct{}, len(s.values))
		for _, value := range s.values {
			s.set[value] = struct{}{}
		}
	}
}

func (s *StackSet[T]) Pop() T {
	if len(s.values) == 0 {
		panic("stack is empty")
	}
	last := len(s.values) - 1
	value := s.values[last]
	var zero T
	s.values[last] = zero
	s.values = s.values[:last]
	delete(s.set, value)
	return value
}

func (s *StackSet[T]) Len() int {
	return len(s.values)
}

func (s *StackSet[T]) Clear() {
	clear(s.set)
	clear(s.values)
	s.values = s.values[:0]
}
