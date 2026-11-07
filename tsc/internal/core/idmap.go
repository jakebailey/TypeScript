package core

import "iter"

// IdKey is the key of an IdMap: a number that is never zero.
type IdKey interface {
	~uint32 | ~uint64
}

// idMapMultiplier is 2^64 divided by the golden ratio. Multiplying by it and
// keeping the high bits spreads consecutive ids evenly over the table.
const idMapMultiplier = 0x9e3779b97f4a7c15

// An IdMap is a hash map from nonzero numeric keys (ids, or ids packed
// into a word) to values, which it holds inline. It is open-addressed and
// probed linearly, and a zero key marks an empty slot, so an entry is just
// its key and value and a lookup touches no memory besides the slots it
// probes. The zero IdMap is empty and ready to use.
//
// A pointer returned by Ptr is valid until the next insertion.
type IdMap[K IdKey, V any] struct {
	entries []idMapEntry[K, V]
	// shift turns a hash into a slot: 64 minus log2(len(entries)).
	shift uint32
	count int
}

type idMapEntry[K IdKey, V any] struct {
	key   K
	value V
}

func (m *IdMap[K, V]) slot(key K) int {
	return int(uint64(key) * idMapMultiplier >> m.shift)
}

// Get returns the value of key, or the zero value if key is absent.
func (m *IdMap[K, V]) Get(key K) V {
	if m.count != 0 {
		mask := len(m.entries) - 1
		for i := m.slot(key); ; i = (i + 1) & mask {
			e := &m.entries[i]
			if e.key == key {
				return e.value
			}
			if e.key == 0 {
				break
			}
		}
	}
	var zero V
	return zero
}

// TryGet returns the value of key and whether key is present.
func (m *IdMap[K, V]) TryGet(key K) (V, bool) {
	if m.count != 0 {
		mask := len(m.entries) - 1
		for i := m.slot(key); ; i = (i + 1) & mask {
			e := &m.entries[i]
			if e.key == key {
				return e.value, true
			}
			if e.key == 0 {
				break
			}
		}
	}
	var zero V
	return zero, false
}

// Has reports whether key is present.
func (m *IdMap[K, V]) Has(key K) bool {
	_, ok := m.TryGet(key)
	return ok
}

// Len returns the number of entries.
func (m *IdMap[K, V]) Len() int {
	return m.count
}

// Set sets the value of key.
func (m *IdMap[K, V]) Set(key K, value V) {
	*m.Ptr(key) = value
}

// Ptr returns the address of key's value, inserting a zero value if key is
// absent. The address is valid until the next insertion.
func (m *IdMap[K, V]) Ptr(key K) *V {
	if m.count != 0 {
		mask := len(m.entries) - 1
		for i := m.slot(key); ; i = (i + 1) & mask {
			e := &m.entries[i]
			if e.key == key {
				return &e.value
			}
			if e.key == 0 {
				break
			}
		}
	}
	return m.insert(key)
}

// insert adds key, which is absent, with a zero value.
func (m *IdMap[K, V]) insert(key K) *V {
	if key == 0 {
		panic("IdMap: zero key")
	}
	// Keep the table at most 3/4 full, so probes stay short.
	if (m.count+1)*4 > len(m.entries)*3 {
		m.grow()
	}
	mask := len(m.entries) - 1
	i := m.slot(key)
	for m.entries[i].key != 0 {
		i = (i + 1) & mask
	}
	m.count++
	e := &m.entries[i]
	e.key = key
	return &e.value
}

func (m *IdMap[K, V]) grow() {
	old := m.entries
	size := max(len(old)*2, 8)
	m.entries = make([]idMapEntry[K, V], size)
	m.shift = uint32(64 - log2(size))
	mask := size - 1
	for _, e := range old {
		if e.key != 0 {
			i := m.slot(e.key)
			for m.entries[i].key != 0 {
				i = (i + 1) & mask
			}
			m.entries[i] = e
		}
	}
}

func log2(n int) int {
	b := 0
	for n > 1 {
		n >>= 1
		b++
	}
	return b
}

// Delete removes key. The entries that probed past its slot are shifted
// back, so no probe sequence is broken and no tombstone is needed.
func (m *IdMap[K, V]) Delete(key K) {
	if m.count == 0 {
		return
	}
	mask := len(m.entries) - 1
	i := m.slot(key)
	for m.entries[i].key != key {
		if m.entries[i].key == 0 {
			return
		}
		i = (i + 1) & mask
	}
	j := i
	for {
		j = (j + 1) & mask
		e := m.entries[j]
		if e.key == 0 {
			break
		}
		// Move e into the hole at i unless its home lies cyclically in
		// (i, j].
		home := m.slot(e.key)
		if (j > i && (home <= i || home > j)) || (j < i && home <= i && home > j) {
			m.entries[i] = e
			i = j
		}
	}
	m.entries[i] = idMapEntry[K, V]{}
	m.count--
}

// Clear removes all entries, keeping the table's storage.
func (m *IdMap[K, V]) Clear() {
	if m.count != 0 {
		clear(m.entries)
		m.count = 0
	}
}

// All iterates over the entries in no particular order. The map must not be
// changed during iteration.
func (m *IdMap[K, V]) All() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i := range m.entries {
			e := &m.entries[i]
			if e.key != 0 && !yield(e.key, e.value) {
				return
			}
		}
	}
}

// Keys iterates over the keys in no particular order. The map must not be
// changed during iteration.
func (m *IdMap[K, V]) Keys() iter.Seq[K] {
	return func(yield func(K) bool) {
		for i := range m.entries {
			if k := m.entries[i].key; k != 0 && !yield(k) {
				return
			}
		}
	}
}
