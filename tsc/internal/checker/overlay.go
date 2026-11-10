package checker

import (
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
	if len(c.overlay) == 0 && len(c.mapOverlay) == 0 {
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
