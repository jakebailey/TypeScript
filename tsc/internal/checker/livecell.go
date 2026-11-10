package checker

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Checkers sharing a store concurrently publish results into memos that
// other checkers read without a lock; see overlay.go. A result one word wide
// is read and written atomically. A wider one, such as a slice, is written
// between two increments of publishSeq, and read again if publishSeq changed
// meanwhile, so that no checker reads half of it.

var publishSeq atomic.Uint64

type liveKind uint8

const (
	liveWide liveKind = iota
	livePointer
	liveWord64
	liveWord32
)

func liveKindOf[T any]() liveKind {
	switch t := reflect.TypeFor[T](); t.Kind() {
	case reflect.Pointer, reflect.UnsafePointer, reflect.Map, reflect.Chan, reflect.Func:
		return livePointer
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Uintptr:
		return liveWord64
	case reflect.Int32, reflect.Uint32:
		return liveWord32
	}
	return liveWide
}

// loadLive reads the result in *p, which another checker may publish.
func loadLive[T any](p *T) T {
	if !concurrentStores {
		return *p
	}
	// SAFETY: Each case reads a value of the kind and size of T, at the
	// alignment of T, and reinterprets it as T.
	switch liveKindOf[T]() {
	case livePointer:
		v := atomic.LoadPointer((*unsafe.Pointer)(unsafe.Pointer(p)))
		return *(*T)(unsafe.Pointer(&v))
	case liveWord64:
		v := atomic.LoadUint64((*uint64)(unsafe.Pointer(p)))
		return *(*T)(unsafe.Pointer(&v))
	case liveWord32:
		v := atomic.LoadUint32((*uint32)(unsafe.Pointer(p)))
		return *(*T)(unsafe.Pointer(&v))
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
	if !concurrentStores {
		*p = v
		return
	}
	// SAFETY: As for loadLive.
	switch liveKindOf[T]() {
	case livePointer:
		atomic.StorePointer((*unsafe.Pointer)(unsafe.Pointer(p)), *(*unsafe.Pointer)(unsafe.Pointer(&v)))
	case liveWord64:
		atomic.StoreUint64((*uint64)(unsafe.Pointer(p)), *(*uint64)(unsafe.Pointer(&v)))
	case liveWord32:
		atomic.StoreUint32((*uint32)(unsafe.Pointer(p)), *(*uint32)(unsafe.Pointer(&v)))
	default:
		publishSeq.Add(1)
		*p = v
		publishSeq.Add(1)
	}
}
