package checker

import (
	"cmp"
	"slices"
	"sync/atomic"
	"unsafe"
)

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

func typeLookup[K comparable](c *Checker, m map[K]TypeId, k K) (*Type, bool) {
	id, ok := m[k]
	return c.typ(id), ok
}

const (
	listPageShift = 14
	listPageSize  = 1 << listPageShift
)

// listStore holds immutable lists in large pages, so a list costs no allocation of its own and is named by an 8-byte
// handle: its position and length. Position 0 is the nil list and position 1 an empty one. A list never straddles two
// pages; a list longer than a page gets a contiguous block of whole pages. The page directory is replaced rather than
// grown in place, so a reader holding a handle from an earlier allocation can read concurrently with later ones.
type listStore[T any] struct {
	dir   atomic.Pointer[[][]T]
	next  uint32
	bases []listPageBase // page start addresses, sorted, for find
}

type listPageBase struct {
	addr uintptr
	page uint32
}

// alloc reserves room for a list of n elements and returns its position and the room.
func (s *listStore[T]) alloc(n int) (uint32, []T) {
	if s.next == 0 {
		s.grow(1)
		s.next = 2
	}
	if n == 0 {
		return 1, (*s.dir.Load())[0][1:1:1]
	}
	off := s.next & (listPageSize - 1)
	if n > listPageSize {
		pages := len(*s.dir.Load())
		k := (n + listPageSize - 1) >> listPageShift
		s.grow(k)
		pos := uint32(pages) << listPageShift
		s.next = uint32(pages+k) << listPageShift
		return pos, (*s.dir.Load())[pages][:n:n]
	}
	if int(off)+n > listPageSize {
		s.next = (s.next | (listPageSize - 1)) + 1
		off = 0
	}
	page := s.next >> listPageShift
	if int(page) >= len(*s.dir.Load()) {
		s.grow(1)
	}
	pos := s.next
	s.next += uint32(n)
	return pos, (*s.dir.Load())[page][off : int(off)+n : int(off)+n]
}

// grow appends k pages, allocated as one block so that a list longer than a page is contiguous.
func (s *listStore[T]) grow(k int) {
	block := make([]T, k*listPageSize)
	var grown [][]T
	if d := s.dir.Load(); d != nil {
		grown = slices.Clip(*d)
	}
	for i := range k {
		page := block[i*listPageSize:]
		// SAFETY: the address is only used as a sort key by find.
		base := listPageBase{uintptr(unsafe.Pointer(unsafe.SliceData(page))), uint32(len(grown))}
		at, _ := slices.BinarySearchFunc(s.bases, base.addr, func(b listPageBase, a uintptr) int { return cmp.Compare(b.addr, a) })
		s.bases = slices.Insert(s.bases, at, base)
		grown = append(grown, page)
	}
	s.dir.Store(&grown)
}

// find returns the handle of a non-empty list that was returned by slice, so that storing it again keeps its
// identity, as sharing the slice did. It reports false for a list held anywhere else.
func (s *listStore[T]) find(list []T) (uint32, bool) {
	if len(list) == 0 || len(list) != cap(list) {
		return 0, false
	}
	var zero T
	size := unsafe.Sizeof(zero)
	// SAFETY: addresses are only compared, never turned back into pointers. Pages are never freed while the store
	// lives and Go does not move heap objects, so an address inside a page always names that page's element.
	addr := uintptr(unsafe.Pointer(unsafe.SliceData(list)))
	at, found := slices.BinarySearchFunc(s.bases, addr, func(b listPageBase, a uintptr) int { return cmp.Compare(b.addr, a) })
	if !found {
		if at == 0 {
			return 0, false
		}
		at--
	}
	base := s.bases[at]
	off := (addr - base.addr) / size
	if off >= listPageSize {
		return 0, false
	}
	return base.page<<listPageShift + uint32(off), true
}

// slice returns the list at a position, or nil for position 0.
func (s *listStore[T]) slice(pos uint32, n uint32) []T {
	if pos == 0 {
		return nil
	}
	off := pos & (listPageSize - 1)
	return (*s.dir.Load())[pos>>listPageShift][off : off+n : off+n]
}
