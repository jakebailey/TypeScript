package checker

import (
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Checkers sharing a store concurrently publish results into memos that
// other checkers read without a lock; see overlay.go. A result wider than a
// word, such as a slice, is written between two increments of publishSeq,
// and read again if publishSeq changed meanwhile, so that no checker reads
// half of it.

var publishSeq atomic.Uint64

// loadLive reads the result in *p, which another checker may publish. A
// result one word wide or less is read with one load, which hardware does
// not tear for aligned values; a wider one is read again if a publisher
// wrote meanwhile.
func loadLive[T any](p *T) T {
	if !concurrentStores || unsafe.Sizeof(*p) <= unsafe.Sizeof(uintptr(0)) {
		return *p
	}
	for {
		s := publishSeq.Load()
		if s&1 == 0 {
			v := *p
			if publishSeq.Load() == s {
				return v
			}
		}
		runtime.Gosched()
	}
}

// storeLive publishes v into *p, while publishing an overlay.
func storeLive[T any](p *T, v T) {
	if !concurrentStores || unsafe.Sizeof(*p) <= unsafe.Sizeof(uintptr(0)) {
		*p = v
		return
	}
	publishSeq.Add(1)
	*p = v
	publishSeq.Add(1)
}
