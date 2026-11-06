package checker

import "slices"

// Helpers over lists of type ids whose callbacks take the types themselves. They follow the semantics of the
// generic helpers in core.

func (c *Checker) typesSome(ts []TypeId, f func(*Type) bool) bool {
	for _, id := range ts {
		if f(c.typ(id)) {
			return true
		}
	}
	return false
}

func (c *Checker) typesEvery(ts []TypeId, f func(*Type) bool) bool {
	for _, id := range ts {
		if !f(c.typ(id)) {
			return false
		}
	}
	return true
}

func (c *Checker) typesFilter(ts []TypeId, f func(*Type) bool) []TypeId {
	for i, id := range ts {
		if !f(c.typ(id)) {
			result := slices.Clone(ts[:i])
			for i++; i < len(ts); i++ {
				if id := ts[i]; f(c.typ(id)) {
					result = append(result, id)
				}
			}
			return result
		}
	}
	return ts
}

func (c *Checker) typesFind(ts []TypeId, f func(*Type) bool) *Type {
	for _, id := range ts {
		if t := c.typ(id); f(t) {
			return t
		}
	}
	return nil
}

func (c *Checker) typesFindIndex(ts []TypeId, f func(*Type) bool) int {
	for i, id := range ts {
		if f(c.typ(id)) {
			return i
		}
	}
	return -1
}

func (c *Checker) typesSameMap(ts []TypeId, f func(*Type) *Type) []TypeId {
	for i, id := range ts {
		mapped := f(c.typ(id)).ref()
		if mapped != id {
			result := make([]TypeId, len(ts))
			copy(result, ts[:i])
			result[i] = mapped
			for j := i + 1; j < len(ts); j++ {
				result[j] = f(c.typ(ts[j])).ref()
			}
			return result
		}
	}
	return ts
}

func (c *Checker) typesSameMapIndex(ts []TypeId, f func(*Type, int) *Type) []TypeId {
	for i, id := range ts {
		mapped := f(c.typ(id), i).ref()
		if mapped != id {
			result := make([]TypeId, len(ts))
			copy(result, ts[:i])
			result[i] = mapped
			for j := i + 1; j < len(ts); j++ {
				result[j] = f(c.typ(ts[j]), j).ref()
			}
			return result
		}
	}
	return ts
}

func (c *Checker) typesMap(ts []TypeId, f func(*Type) *Type) []TypeId {
	if ts == nil {
		return nil
	}
	result := make([]TypeId, len(ts))
	for i, id := range ts {
		result[i] = f(c.typ(id)).ref()
	}
	return result
}

func (c *Checker) typesMapIndex(ts []TypeId, f func(*Type, int) *Type) []TypeId {
	if ts == nil {
		return nil
	}
	result := make([]TypeId, len(ts))
	for i, id := range ts {
		result[i] = f(c.typ(id), i).ref()
	}
	return result
}

func (c *Checker) typesMapNonNil(ts []TypeId, f func(*Type) *Type) []TypeId {
	var result []TypeId
	for _, id := range ts {
		if mapped := f(c.typ(id)); mapped != nil {
			result = append(result, mapped.ref())
		}
	}
	return result
}

func typesMapTo[U any](c *Checker, ts []TypeId, f func(*Type) U) []U {
	if ts == nil {
		return nil
	}
	result := make([]U, len(ts))
	for i, id := range ts {
		result[i] = f(c.typ(id))
	}
	return result
}

func typesMapIndexTo[U any](c *Checker, ts []TypeId, f func(*Type, int) U) []U {
	if ts == nil {
		return nil
	}
	result := make([]U, len(ts))
	for i, id := range ts {
		result[i] = f(c.typ(id), i)
	}
	return result
}

func typesMapNonNilTo[U comparable](c *Checker, ts []TypeId, f func(*Type) U) []U {
	var result []U
	for _, id := range ts {
		mapped := f(c.typ(id))
		if mapped != *new(U) {
			result = append(result, mapped)
		}
	}
	return result
}

func mapToTypeIds[T any](s []T, f func(T) *Type) []TypeId {
	if s == nil {
		return nil
	}
	result := make([]TypeId, len(s))
	for i, v := range s {
		result[i] = f(v).ref()
	}
	return result
}

func mapIndexToTypeIds[T any](s []T, f func(T, int) *Type) []TypeId {
	if s == nil {
		return nil
	}
	result := make([]TypeId, len(s))
	for i, v := range s {
		result[i] = f(v, i).ref()
	}
	return result
}

func mapNonNilToTypeIds[T any](s []T, f func(T) *Type) []TypeId {
	var result []TypeId
	for _, v := range s {
		if mapped := f(v); mapped != nil {
			result = append(result, mapped.ref())
		}
	}
	return result
}

// typesOf resolves a list of type ids, for callers outside the checker.
func (c *Checker) typesOf(ts []TypeId) []*Type {
	if ts == nil {
		return nil
	}
	result := make([]*Type, len(ts))
	for i, id := range ts {
		result[i] = c.typ(id)
	}
	return result
}

// typeIdsOf returns the ids of a list of types.
func typeIdsOf(ts []*Type) []TypeId {
	if ts == nil {
		return nil
	}
	result := make([]TypeId, len(ts))
	for i, t := range ts {
		result[i] = t.ref()
	}
	return result
}

func (c *Checker) typesCount(ts []TypeId, f func(*Type) bool) int {
	n := 0
	for _, id := range ts {
		if f(c.typ(id)) {
			n++
		}
	}
	return n
}

// TypesOf resolves a list of type ids returned by a Type accessor.
func (c *Checker) TypesOf(ts []TypeId) []*Type {
	return c.typesOf(ts)
}

// TypeOf resolves a type id; 0 resolves to nil.
func (c *Checker) TypeOf(id TypeId) *Type {
	return c.typ(id)
}
