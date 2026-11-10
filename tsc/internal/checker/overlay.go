package checker

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

// Overlay
//
// Checkers sharing a store concurrently (see concurrentStores) do not write
// the results they compute to the store while a request is in progress.
// Objects are published by the cells caching them before they are finished,
// and results computed in a cycle may be discarded, so another checker must
// not see them until nothing is in progress. A checker instead writes them to
// its overlay, and reads its overlay where the store has no result yet: a
// result the store has is final. When the outermost request ends, the checker
// publishes its overlay to the store, unless another checker has published a
// result for one of its entries since: its results then refer to its own
// copies of objects the store has others of, so it publishes none of them.
// What it reported is the same either way, as results do not depend on which
// checker computes them.

// overlayEntry is the result a checker caches for one memo cell, flag memo
// or memo map entry while a request is in progress.
type overlayEntry struct {
	value     any
	publish   func()      // Writes value to the store
	published func() bool // Reports whether the store has a result
}

// overlayKey identifies a memo cell or flag memo in an overlay by where its
// result is stored: the cell, or the word and bits holding a flag memo.
type overlayKey struct {
	p    unsafe.Pointer
	mask uint32
}

// mapOverlayKey identifies an entry of a memo map in an overlay.
type mapOverlayKey struct {
	source memoStateSource
	key    any
}

// overlaying reports whether results are written to the overlay.
func (c *Checker) overlaying() bool {
	return concurrentStores && c.overlayDepth != 0
}

// beginOverlay starts a request, from which results are written to the
// overlay until the outermost request ends.
func (c *Checker) beginOverlay() {
	if concurrentStores {
		c.overlayDepth++
	}
}

// endOverlay ends a request, publishing the overlay once the outermost
// request ends.
func (c *Checker) endOverlay() {
	if !concurrentStores {
		return
	}
	c.overlayDepth--
	if c.overlayDepth == 0 {
		c.publishOverlay()
	}
}

func (c *Checker) publishOverlay() {
	if len(c.overlay) == 0 && len(c.mapOverlay) == 0 && len(c.memoStates) == 0 && len(c.mapStates) == 0 {
		return
	}
	// Publishing must not interleave with another checker's, which could
	// publish a result for one of the entries after it was checked.
	c.publishMu.Lock()
	defer c.publishMu.Unlock()
	conflict := false
	for _, e := range c.overlay {
		if e.published() {
			conflict = true
			break
		}
	}
	if !conflict {
		for _, e := range c.mapOverlay {
			if e.published() {
				conflict = true
				break
			}
		}
	}
	if !conflict {
		c.publishStates()
	}
	c.dropStates()
	if conflict {
		c.overlayDrops++
	} else {
		c.overlayPublishes++
		for _, e := range c.overlay {
			e.publish()
		}
		for _, e := range c.mapOverlay {
			e.publish()
		}
	}
	clear(c.overlay)
	clear(c.mapOverlay)
}

// sharedMemoStates holds the memo states that checkers sharing a store
// concurrently published with their overlays. A checker holds the states of
// the results in its overlay itself, so that no other checker sees them
// before the results.
type sharedMemoStates struct {
	mu      sync.RWMutex
	states  map[memoKey]*memoState
	filter  atomic.Pointer[[1 << sharedStateFilterBits / 64]atomic.Uint64]
	pending atomic.Int64 // Published states with unreplayed effects
}

const sharedStateFilterBits = 20

func sharedStateFilterIndex(key memoKey) uint64 {
	return memoKeyBits(key) >> (64 - sharedStateFilterBits)
}

func (s *sharedMemoStates) get(key memoKey) *memoState {
	f := s.filter.Load()
	if f == nil {
		return nil
	}
	if i := sharedStateFilterIndex(key); f[i/64].Load()&(1<<(i%64)) == 0 {
		return nil
	}
	s.mu.RLock()
	state := s.states[key]
	s.mu.RUnlock()
	return state
}

// statePublisher is a memo map, which publishes the memo states of its
// entries.
type statePublisher interface {
	publishState(k any, state *memoState)
}

func (m *memoMap[K, V]) publishState(k any, state *memoState) {
	key := k.(K)
	s := m.sideTable()
	m.mu.Lock()
	defer m.mu.Unlock()
	if s.memoStates == nil {
		s.memoStates = make(map[K]*memoState)
	}
	s.memoStates[key] = state
	if s.keyBits != nil {
		s.stateKeys.add(s.keyBits(key), s.memoStates, s.keyBits)
	}
	if state.errors != nil {
		s.hasErrors.Store(true)
	}
}

// publishStates publishes the memo states c holds with the overlay it
// publishes.
func (c *Checker) publishStates() {
	if len(c.memoStates) != 0 {
		s := &c.sharedStates
		f := s.filter.Load()
		if f == nil {
			f = new([1 << sharedStateFilterBits / 64]atomic.Uint64)
			s.filter.Store(f)
		}
		s.mu.Lock()
		if s.states == nil {
			s.states = make(map[memoKey]*memoState)
		}
		for key, state := range c.memoStates {
			if state.pending != nil {
				s.pending.Add(1)
			}
			s.states[key] = state
			i := sharedStateFilterIndex(key)
			f[i/64].Or(1 << (i % 64))
		}
		s.mu.Unlock()
	}
	for key, state := range c.mapStates {
		if state.pending != nil {
			c.sharedStates.pending.Add(1)
		}
		key.source.(statePublisher).publishState(key.key, state)
	}
}

// dropStates forgets the memo states c holds, once published or dropped
// with its overlay.
func (c *Checker) dropStates() {
	clear(c.memoStates)
	c.stateFilter = stateKeyFilter{}
	c.memoPendingCount = 0
	clear(c.mapStates)
	c.syncObserving()
}

// overlayCell returns the overlay entry stored under k.
func (c *Checker) overlayCell(k overlayKey) (*overlayEntry, bool) {
	e, ok := c.overlay[k]
	return e, ok
}

// putOverlayCell caches e for the memo cell or flag memo stored under key,
// identified by memoKey, in the overlay. If discardable and results may be
// discarded, it logs how to restore the previous entry, as set does for the
// store.
func (c *Checker) putOverlayCell(key overlayKey, e *overlayEntry, memo *memoHeader, memoKey memoKey, discardable bool) {
	if c.overlay == nil {
		c.overlay = make(map[overlayKey]*overlayEntry)
	}
	if discardable && c.logResults() {
		old, had := c.overlay[key]
		c.logMemoDiscard(memo, memoKey, func() {
			if had {
				c.overlay[key] = old
			} else {
				delete(c.overlay, key)
			}
		})
	}
	c.overlay[key] = e
}

// putOverlayEntry caches e under key in the overlay, without logging how to
// discard it; the caller does.
func (c *Checker) putOverlayEntry(key overlayKey, e *overlayEntry) {
	if c.overlay == nil {
		c.overlay = make(map[overlayKey]*overlayEntry)
	}
	c.overlay[key] = e
}

// putOverlayMapEntry is putOverlayCell for an entry of a memo map, logging
// the restore as own if it is the result of the current computation.
func (c *Checker) putOverlayMapEntry(k mapOverlayKey, e *overlayEntry, own bool, discardable bool) {
	if c.mapOverlay == nil {
		c.mapOverlay = make(map[mapOverlayKey]*overlayEntry)
	}
	if discardable && c.logResults() {
		old, had := c.mapOverlay[k]
		c.logResultDiscard(own, nil, func() {
			if had {
				c.mapOverlay[k] = old
			} else {
				delete(c.mapOverlay, k)
			}
		})
	}
	c.mapOverlay[k] = e
}

// isZeroValue reports whether *v is the zero value of T.
func isZeroValue[T any](v *T) bool {
	// SAFETY: Reads the bytes of a value of T, which are all zero exactly for
	// its zero value for the types of memoized results.
	b := unsafe.Slice((*byte)(unsafe.Pointer(v)), unsafe.Sizeof(*v))
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}
