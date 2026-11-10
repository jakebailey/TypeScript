package checker

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
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
// or map entry while a request is in progress.
type overlayEntry struct {
	value any
	p     unsafe.Pointer // Where the store keeps the result: a cell or a flag word
	ops   overlayOps
	hash  uint64 // Of the key, for recent filters; see markRecent
	mask  uint32 // The bits of a flag memo
}

// overlayOps publishes the results of entries of one kind. Its
// implementations have no fields, so that an entry needs no allocation.
type overlayOps interface {
	publish(c *Checker, e *overlayEntry)
	// conflicts reports whether the store has a different result: then the
	// results of the overlay may refer to objects of which the store has
	// other copies. A store holding the same result, which another checker
	// computed meanwhile, as results do not depend on which checker computes
	// them, is no conflict.
	conflicts(c *Checker, e *overlayEntry) bool
}

// sameBits reports whether a and b hold the same value, or for a slice, the
// same one.
func sameBits[T any](a, b *T) bool {
	if reflect.TypeFor[T]().Comparable() {
		return any(*a) == any(*b)
	}
	// SAFETY: Compares the bytes of two values of T, which is a slice or
	// holds them, and has no padding.
	n := unsafe.Sizeof(*a)
	return unsafe.String((*byte)(unsafe.Pointer(a)), n) == unsafe.String((*byte)(unsafe.Pointer(b)), n)
}

// valueAs returns v as a T, or the zero T if v is nil.
func valueAs[T any](v any) T {
	t, _ := v.(T)
	return t
}

// cellOps publishes the result of a memo cell, or of a field holding one.
type cellOps[T any] struct{}

func (cellOps[T]) publish(_ *Checker, e *overlayEntry) { storeLive((*T)(e.p), valueAs[T](e.value)) }
func (cellOps[T]) conflicts(_ *Checker, e *overlayEntry) bool {
	live := loadLive((*T)(e.p))
	own := valueAs[T](e.value)
	return !isZeroValue(&live) && !sameBits(&live, &own)
}

// flagBitsOps publishes the bits of a flag memo, under the mask of the
// entry, in a word that other bits share.
type flagBitsOps[T ~uint32] struct{}

func (flagBitsOps[T]) publish(_ *Checker, e *overlayEntry) {
	p := (*uint32)(e.p)
	bits := uint32(valueAs[T](e.value)) & e.mask
	for {
		old := atomic.LoadUint32(p)
		if atomic.CompareAndSwapUint32(p, old, old&^e.mask|bits) {
			return
		}
	}
}

func (flagBitsOps[T]) conflicts(_ *Checker, e *overlayEntry) bool {
	live := atomic.LoadUint32((*uint32)(e.p)) & e.mask
	return live != 0 && live != uint32(valueAs[T](e.value))&e.mask
}

// noOps is the entry of a result that is no longer cached.
type noOps struct{}

func (noOps) publish(*Checker, *overlayEntry)        {}
func (noOps) conflicts(*Checker, *overlayEntry) bool { return false }

// overlayKey identifies a memo cell or flag memo in an overlay by where its
// result is stored: the cell, or the word and bits holding a flag memo.
type overlayKey struct {
	p    unsafe.Pointer
	mask uint32
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
		if c.overlayDepth == 1 && dbgSerial {
			dbgSerialMu.Lock()
		}
	}
}

var dbgSerial = os.Getenv("TSGO_SHARED_SERIAL") != ""
var dbgSerialMu sync.Mutex

// endOverlay ends a request, publishing the overlay once the outermost
// request ends.
func (c *Checker) endOverlay() {
	if !concurrentStores {
		return
	}
	c.overlayDepth--
	if c.overlayDepth == 0 {
		c.publishOverlay()
		if dbgSerial {
			dbgSerialMu.Unlock()
		}
	}
}

// publishCheckpoint publishes the overlay while a check is in progress, at
// a point where nothing it computes is: between the statements of a file,
// so that a conflict drops only what was computed since.
func (c *Checker) publishCheckpoint() {
	if c.overlayDepth == 1 && c.memoTop == 0 && len(c.resolutions) == 1 && len(c.discardLog) == 0 &&
		c.circularityStart == 0 && c.memoPendingCount == 0 && len(c.heldEffects) == 0 && c.currentNodeErrorCollectors == 0 {
		c.publishOverlay()
		if dbgSerial {
			dbgSerialMu.Unlock()
			runtime.Gosched()
			dbgSerialMu.Lock()
		}
	}
}

func (c *Checker) publishOverlay() {
	if len(c.overlay) == 0 && len(c.ownMaps) == 0 && len(c.memoStates) == 0 {
		return
	}
	// Publishing must not interleave with another checker's, which could
	// publish a result for one of the entries after it was checked.
	c.publishMu.Lock()
	conflict := false
	for _, e := range c.overlay {
		if e.ops.conflicts(c, &e) {
			conflict = true
			dbgConflictKinds.Add(fmt.Sprintf("%T", e.ops), 1)
			break
		}
	}
	if !conflict {
		for _, m := range c.ownMaps {
			if m.conflicts(c) {
				conflict = true
				dbgConflictKinds.Add(fmt.Sprintf("%T", m), 1)
				break
			}
		}
	}
	if !conflict {
		c.markRecent()
		c.publishStates()
	}
	if conflict {
		c.overlayDrops++
		dbgDrops.Add(1)
		dbgDropEntries.Add(int64(len(c.overlay)))
	} else {
		c.overlayPublishes++
		dbgPubs.Add(1)
		dbgPubEntries.Add(int64(len(c.overlay)))
		c.publishEpoch++
		for _, e := range c.overlay {
			e.ops.publish(c, &e)
		}
		for _, m := range c.ownMaps {
			m.publish(c)
		}
	}
	// The next request starts here, while no other checker is publishing.
	c.requestEpoch = c.publishEpoch
	var recentKeys map[uint64]struct{}
	r := c.recent
	if r != nil {
		r.mu.Lock()
		if len(r.keys) != 0 {
			for i := range r.bits {
				r.bits[i].Store(0)
			}
			recentKeys = r.keys
			r.keys = nil
		}
		r.mu.Unlock()
	}
	c.publishMu.Unlock()
	c.dropStates()
	// Clearing a map costs as much as its largest size, so a map that grew
	// for a large request is dropped instead.
	if len(c.overlay) != 0 {
		c.ownFilter = ownFilter{}
	}
	if len(c.overlay) > 4096 {
		c.overlay = nil
	} else {
		clear(c.overlay)
	}
	for _, m := range c.ownMaps {
		m.drop(c)
	}
	clear(c.ownMaps)
	c.ownMaps = c.ownMaps[:0]
	if recentKeys != nil && len(recentKeys) <= 4096 {
		clear(recentKeys)
		r.mu.Lock()
		r.spare = recentKeys
		r.mu.Unlock()
	}
}

// Snapshots
//
// A checker sees the store as it was when its request started, with its own
// results on top. Otherwise, a result another checker publishes meanwhile
// could refer to that checker's copy of an object of which the checker has
// its own, and the checker would compute from both. Each checker keeps the
// keys published since its request started, which publishers add before
// writing their results, and reads a result whose key is among them as if
// the store had none. A filter of the keys rules out most reads without a
// lock. The keys must be exact: a result set when its owner was created is
// never published, and is never computed again if read as missing.

const recentFilterBits = 16

type recentFilter struct {
	bits  [1 << recentFilterBits / 64]atomic.Uint64
	mu    sync.Mutex
	keys  map[uint64]struct{}
	spare map[uint64]struct{} // A cleared map for keys to reuse
}

func (k overlayKey) hash() uint64 {
	return mixHash(uint64(uintptr(k.p)) ^ uint64(k.mask)*0xD6E8FEB86659FD93)
}

func mixHash(h uint64) uint64 {
	h *= 0x9E3779B97F4A7C15
	return h ^ h>>29
}

// markRecent marks the keys of the overlay c publishes in the recent filters
// of the other checkers sharing the store.
func (c *Checker) markRecent() {
	for _, other := range c.sharers {
		if other == c || other.recent == nil {
			continue
		}
		r := other.recent
		r.mu.Lock()
		if r.keys == nil {
			if r.keys = r.spare; r.keys == nil {
				r.keys = make(map[uint64]struct{})
			}
			r.spare = nil
		}
		mark := func(h uint64) {
			r.keys[h] = struct{}{}
			i := mixHash(h) >> (64 - recentFilterBits)
			r.bits[i/64].Or(1 << (i % 64))
		}
		for _, e := range c.overlay {
			mark(e.hash)
		}
		r.mu.Unlock()

	}
}

// recentlyPublished reports whether the result with key hash h may have been
// published since the request of c started, in which case c reads it as if
// the store had none.
func (c *Checker) recentlyPublished(h uint64) bool {
	r := c.recent
	if r == nil {
		return false
	}
	if i := mixHash(h) >> (64 - recentFilterBits); r.bits[i/64].Load()&(1<<(i%64)) == 0 {
		return false
	}
	r.mu.Lock()
	_, ok := r.keys[h]
	r.mu.Unlock()
	return ok
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
				dbgPendingPublished.Add(1)
			}
			s.states[key] = state
			i := sharedStateFilterIndex(key)
			f[i/64].Or(1 << (i % 64))
		}
		s.mu.Unlock()
	}
}

// dropStates forgets the memo states c holds, once published or dropped
// with its overlay.
func (c *Checker) dropStates() {
	c.memoStates = nil
	c.stateFilter = stateKeyFilter{}
	c.memoPendingCount = 0
	c.syncObserving()
}

// overlayCell returns the result c holds under k.
func (c *Checker) overlayCell(k overlayKey) (any, bool) {
	if i := k.hash() >> (64 - ownFilterBits); c.ownFilter[i/64]&(1<<(i%64)) == 0 {
		return nil, false
	}
	e, ok := c.overlay[k]
	return e.value, ok
}

// ownFilterBits sizes the filter of the keys in the overlay, which rules out
// most lookups of results the checker does not hold.
const ownFilterBits = 14

type ownFilter [1 << ownFilterBits / 64]uint64

func (c *Checker) addOwn(k overlayKey) {
	i := k.hash() >> (64 - ownFilterBits)
	c.ownFilter[i/64] |= 1 << (i % 64)
}

// putOverlayCell caches e for the memo cell or flag memo stored under key,
// identified by memoKey, in the overlay. If discardable and results may be
// discarded, it logs how to restore the previous entry, as set does for the
// store.
func (c *Checker) putOverlayCell(key overlayKey, e overlayEntry, memo *memoHeader, memoKey memoKey, discardable bool) {
	e.hash = key.hash()
	c.addOwn(key)
	if c.overlay == nil {
		c.overlay = make(map[overlayKey]overlayEntry)
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
func (c *Checker) putOverlayEntry(key overlayKey, e overlayEntry) {
	e.hash = key.hash()
	c.addOwn(key)
	if c.overlay == nil {
		c.overlay = make(map[overlayKey]overlayEntry)
	}
	c.overlay[key] = e
}

// isZeroValue reports whether *v is the zero value of T.
func isZeroValue[T any](v *T) bool {
	switch reflect.TypeFor[T]().Kind() {
	case reflect.Struct, reflect.Array:
		// Copies of a struct need not copy its padding, so its bytes do
		// not tell.
		return reflect.ValueOf(v).Elem().IsZero()
	}
	// SAFETY: Reads the bytes of a value of T, which has no padding, so
	// they are all zero exactly for its zero value.
	b := unsafe.Slice((*byte)(unsafe.Pointer(v)), unsafe.Sizeof(*v))
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}

var dbgPubs, dbgDrops, dbgPubEntries, dbgDropEntries, dbgPendingPublished atomic.Int64

type dbgCounter struct {
	mu sync.Mutex
	m  map[string]int
}

func (d *dbgCounter) Add(k string, n int) {
	d.mu.Lock()
	if d.m == nil {
		d.m = map[string]int{}
	}
	d.m[k] += n
	d.mu.Unlock()
}

var dbgConflictKinds dbgCounter

func DebugOverlayStats() string {
	dbgConflictKinds.mu.Lock()
	for k, v := range dbgConflictKinds.m {
		fmt.Fprintf(os.Stderr, "conflict %d %s\n", v, k)
	}
	dbgConflictKinds.mu.Unlock()
	return fmt.Sprintf("publishes=%d (%d entries) drops=%d (%d entries) pendingPublished=%d", dbgPubs.Load(), dbgPubEntries.Load(), dbgDrops.Load(), dbgDropEntries.Load(), dbgPendingPublished.Load())
}
