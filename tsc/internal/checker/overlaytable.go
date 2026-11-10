package checker

import "math/bits"

// overlayTable holds the entries of an overlay: in a slice in the order
// they were first written, which publishing walks, with an open-addressed
// index from their keys to their positions. Both are reused from one
// request to the next, and the index is cleared in constant time, since a
// checker publishes after every statement it checks.
type overlayTable struct {
	entries []overlayEntry // An entry without ops is absent
	slots   []overlaySlot
	gen     uint32
	count   int // Of slots used
	shift   uint8
	shadows bool                            // Whether an entry was put over a result the store has
	filter  [1 << ownFilterBits / 64]uint64 // Hashes of the keys, which rule out most lookups
}

// ownFilterBits sizes the filter of the keys in the overlay.
const ownFilterBits = 14

type overlaySlot struct {
	key   overlayKey
	index int32
	gen   uint32
}

func (t *overlayTable) len() int {
	return len(t.entries)
}

func (t *overlayTable) slot(h uint64) int {
	return int(h >> t.shift)
}

func (t *overlayTable) mayHave(h uint64) bool {
	i := h >> (64 - ownFilterBits)
	return t.filter[i/64]&(1<<(i%64)) != 0
}

// find returns the position of the entry for k, whose hash is h, or -1.
func (t *overlayTable) find(k overlayKey, h uint64) int {
	if t.count == 0 || !t.mayHave(h) {
		return -1
	}
	mask := len(t.slots) - 1
	for i := t.slot(h); ; i = (i + 1) & mask {
		s := &t.slots[i]
		if s.gen != t.gen {
			return -1
		}
		if s.key == k {
			return int(s.index)
		}
	}
}

// get returns the entry for k, whose hash is h, if present.
func (t *overlayTable) get(k overlayKey, h uint64) (*overlayEntry, bool) {
	if i := t.find(k, h); i >= 0 && t.entries[i].ops != nil {
		return &t.entries[i], true
	}
	return nil, false
}

// put sets the entry for k, whose hash is e.hash.
func (t *overlayTable) put(k overlayKey, e overlayEntry) {
	h := e.hash
	if i := t.find(k, h); i >= 0 {
		t.entries[i] = e
		return
	}
	if (t.count+1)*4 > len(t.slots)*3 {
		t.grow()
	}
	i := h >> (64 - ownFilterBits)
	t.filter[i/64] |= 1 << (i % 64)
	mask := len(t.slots) - 1
	for i := t.slot(h); ; i = (i + 1) & mask {
		s := &t.slots[i]
		if s.gen != t.gen {
			*s = overlaySlot{key: k, index: int32(len(t.entries)), gen: t.gen}
			t.count++
			t.entries = append(t.entries, e)
			return
		}
	}
}

// remove makes the entry for k, whose hash is h, absent.
func (t *overlayTable) remove(k overlayKey, h uint64) {
	if i := t.find(k, h); i >= 0 {
		t.entries[i] = overlayEntry{}
	}
}

func (t *overlayTable) grow() {
	old := t.slots
	oldGen := t.gen
	size := max(len(old)*2, 256)
	t.slots = make([]overlaySlot, size)
	t.shift = uint8(64 - bits.TrailingZeros(uint(size)))
	t.gen = 1
	t.count = 0
	mask := size - 1
	for _, s := range old {
		if s.gen != oldGen {
			continue
		}
		h := t.entries[s.index].hash
		if t.entries[s.index].ops == nil {
			h = s.key.hash()
		}
		for i := t.slot(h); ; i = (i + 1) & mask {
			if n := &t.slots[i]; n.gen != t.gen {
				*n = overlaySlot{key: s.key, index: s.index, gen: t.gen}
				t.count++
				break
			}
		}
	}
}

// reset empties the table, keeping its storage unless it grew large.
func (t *overlayTable) reset() {
	if len(t.entries) == 0 {
		return
	}
	if len(t.entries) > 1<<16 {
		*t = overlayTable{}
		return
	}
	clear(t.entries)
	t.entries = t.entries[:0]
	t.shadows = false
	t.filter = [1 << ownFilterBits / 64]uint64{}
	t.count = 0
	t.gen++
	if t.gen == 0 {
		clear(t.slots)
		t.gen = 1
	}
}
