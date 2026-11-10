package checker

import (
	"sync/atomic"
)

// mapOwners holds what checkers sharing a store concurrently hold for the
// entries of one memo map or intern map while their requests are in
// progress, their own entries and memo states, each indexed by the checker's
// index among the checkers sharing the store and kept in maps typed like
// the map's own, so that looking an entry up costs a map lookup; and when
// each entry of the map was published, so that a checker reads the map as
// it was when its request started (see "Snapshots" in overlay.go). The
// epochs are guarded by the lock of the map.
type mapOwners[K comparable, V any] struct {
	target  mapStore[K, V]
	entries []map[K]V
	states  []map[K]*memoState
	epochs  map[K]uint64
}

// mapStore is a memo map or intern map, which a checker publishes its own
// entries to.
type mapStore[K comparable, V any] interface {
	storeAt(k K, v V, epoch uint64)
	storeGet(k K) (V, bool)
}

// ownedMap is a map whose owners a checker publishes or drops its overlay
// to; its methods are called while publishing, under publishMu.
type ownedMap interface {
	conflicts(c *Checker) bool
	publish(c *Checker)
	drop(c *Checker)
}

func ownersOf[K comparable, V any](p *atomic.Pointer[mapOwners[K, V]], c *Checker, target mapStore[K, V]) *mapOwners[K, V] {
	if o := p.Load(); o != nil {
		return o
	}
	n := len(c.sharers)
	o := &mapOwners[K, V]{
		target:  target,
		entries: make([]map[K]V, n),
		states:  make([]map[K]*memoState, n),
	}
	if p.CompareAndSwap(nil, o) {
		return o
	}
	return p.Load()
}

// own returns the entry c holds for k.
func (o *mapOwners[K, V]) own(c *Checker, k K) (V, bool) {
	if om := o.entries[c.sharerIndex]; om != nil {
		v, ok := om[k]
		return v, ok
	}
	var zero V
	return zero, false
}

// visible reports whether c sees an entry the store has, published at
// epoch.
func visible(c *Checker, epoch uint64) bool {
	return epoch <= c.requestEpoch
}

// put caches v for k in the overlay of c. If discardable and results may be
// discarded, it logs how to restore the previous entry, as own if it is the
// result of the computation in progress.
func (o *mapOwners[K, V]) put(c *Checker, k K, v V, own bool, discardable bool) {
	om := o.entries[c.sharerIndex]
	if om == nil {
		om = make(map[K]V)
		o.entries[c.sharerIndex] = om
		if o.states[c.sharerIndex] == nil {
			c.ownMaps = append(c.ownMaps, o)
		}
	}
	if discardable && c.logResults() {
		old, had := om[k]
		c.logResultDiscard(own, nil, func() {
			if had {
				om[k] = old
			} else {
				delete(om, k)
			}
		})
	}
	om[k] = v
}

// ownState returns the memo state c holds for the entry for k, if any.
func (o *mapOwners[K, V]) ownState(c *Checker, k K) *memoState {
	if s := o.states[c.sharerIndex]; s != nil {
		return s[k]
	}
	return nil
}

// putState holds state as the memo state of the entry for k in the overlay
// of c.
func (o *mapOwners[K, V]) putState(c *Checker, k K, state *memoState) {
	s := o.states[c.sharerIndex]
	if s == nil {
		s = make(map[K]*memoState)
		o.states[c.sharerIndex] = s
		if o.entries[c.sharerIndex] == nil {
			c.ownMaps = append(c.ownMaps, o)
		}
	}
	s[k] = state
}

func (o *mapOwners[K, V]) conflicts(c *Checker) bool {
	for k, v := range o.entries[c.sharerIndex] {
		if live, ok := o.target.storeGet(k); ok && !sameBits(&live, &v) {
			return true
		}
	}
	return false
}

func (o *mapOwners[K, V]) publish(c *Checker) {
	for k, v := range o.entries[c.sharerIndex] {
		// An entry another checker published the same keeps its epoch.
		if _, ok := o.target.storeGet(k); !ok {
			o.target.storeAt(k, v, c.publishEpoch)
		}
	}
	if states := o.states[c.sharerIndex]; len(states) != 0 {
		pub := o.target.(statePublisher)
		for k, state := range states {
			if state.pending != nil {
				c.sharedStates.pending.Add(1)
				dbgPendingPublished.Add(1)
			}
			pub.publishState(k, state)
		}
	}
}

func (o *mapOwners[K, V]) drop(c *Checker) {
	o.entries[c.sharerIndex] = nil
	o.states[c.sharerIndex] = nil
}
