//go:build darwin && (amd64 || arm64)

package fswatch

// #cgo nocallback CFRelease
// #cgo noescape CFStringCreateWithCString
// #cgo nocallback CFStringCreateWithCString
// #cgo noescape CFArrayCreate
// #cgo nocallback CFArrayCreate
// #cgo nocallback CFArrayGetValueAtIndex
// #cgo nocallback CFStringCreateMutableCopy
// #cgo nocallback CFStringNormalize
// #cgo nocallback CFStringFold
// #cgo nocallback CFStringGetLength
// #cgo nocallback CFStringGetMaximumSizeForEncoding
// #cgo noescape CFStringGetCString
// #cgo nocallback CFStringGetCString
// #cgo noescape dispatch_queue_create
// #cgo nocallback dispatch_queue_create
// #cgo nocallback dispatch_release
// #cgo noescape FSEventStreamCreate
// #cgo nocallback FSEventStreamCreate
// #cgo nocallback FSEventStreamSetDispatchQueue
// #cgo nocallback FSEventStreamStart
// #cgo nocallback FSEventsGetCurrentEventId
// #cgo nocallback FSEventStreamStop
// #cgo nocallback FSEventStreamInvalidate
// #cgo nocallback FSEventStreamRelease
import "C"

import (
	"runtime/cgo"
	"unsafe"
)

// CoreFoundation bindings.
//
//cgo:func cfRelease CFRelease "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringCreate CFStringCreateWithCString "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfArrayCreate CFArrayCreate "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfArrayGetValueAtIndex CFArrayGetValueAtIndex "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringCreateMutableCopy CFStringCreateMutableCopy "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringNormalize CFStringNormalize "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringFold CFStringFold "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringGetLength CFStringGetLength "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringGetMaximumSizeForEncoding CFStringGetMaximumSizeForEncoding "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"
//cgo:func cfStringGetCString CFStringGetCString "/System/Library/Frameworks/CoreFoundation.framework/Versions/A/CoreFoundation"

func cfRelease(ref uintptr)
func cfStringCreate(allocator uintptr, cstr *byte, encoding uint32) uintptr
func cfArrayCreate(allocator uintptr, values *uintptr, count int64, callbacks uintptr) uintptr
func cfArrayGetValueAtIndex(array uintptr, index int64) uintptr
func cfStringCreateMutableCopy(allocator uintptr, maxLength int64, str uintptr) uintptr
func cfStringNormalize(mutStr uintptr, form int64)
func cfStringFold(mutStr uintptr, flags uint64, locale uintptr)
func cfStringGetLength(str uintptr) int64
func cfStringGetMaximumSizeForEncoding(length int64, encoding uint32) int64
func cfStringGetCString(str uintptr, buf *byte, bufSize int64, encoding uint32) uint8

// libdispatch bindings (in libSystem.B.dylib / libc).
//
//cgo:func dispatchQueueCreate dispatch_queue_create libc
//cgo:func dispatchRelease dispatch_release libc
//cgo:func dispatchSync dispatch_sync_f libc

func dispatchQueueCreate(label *byte, attr uintptr) uintptr
func dispatchRelease(obj uintptr)
func dispatchSync(queue, context uintptr, work unsafe.Pointer)

// CoreServices / FSEvents bindings.
//
//cgo:type fsEventStreamContext FSEventStreamContext
//cgo:func fsEventStreamCreate FSEventStreamCreate "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventStreamSetDispatchQueue FSEventStreamSetDispatchQueue "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventStreamStart FSEventStreamStart "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventStreamFlushSync FSEventStreamFlushSync "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventsGetCurrentEventID FSEventsGetCurrentEventId "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventStreamStop FSEventStreamStop "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventStreamInvalidate FSEventStreamInvalidate "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"
//cgo:func fsEventStreamRelease FSEventStreamRelease "/System/Library/Frameworks/CoreServices.framework/Versions/A/CoreServices"

func fsEventStreamCreate(allocator uintptr, callback unsafe.Pointer, ctx *fsEventStreamContext, paths uintptr, since uint64, latency float64, flags uint32) uintptr
func fsEventStreamSetDispatchQueue(stream, queue uintptr)
func fsEventStreamStart(stream uintptr) uint8
func fsEventStreamFlushSync(stream uintptr)
func fsEventsGetCurrentEventID() uint64
func fsEventStreamStop(stream uintptr)
func fsEventStreamInvalidate(stream uintptr)
func fsEventStreamRelease(stream uintptr)

func fsEventsCallbackPtr() unsafe.Pointer { return unsafe.Pointer(C.fswatchFSEventsCallback) }
func dispatchNoopPtr() unsafe.Pointer     { return unsafe.Pointer(C.fswatchDispatchNoop) }

//export fswatchFSEventsCallback
func fswatchFSEventsCallback(streamRef, info uintptr, numEvents uint64, eventPaths uintptr, eventFlags *uint32, eventIDs *uint64) {
	state := cgo.Handle(info).Value().(*fseventsStream)
	fsEventsProcessBatch(state.watches, numEvents, eventPaths, eventFlags, eventIDs)
}

//export fswatchDispatchNoop
func fswatchDispatchNoop(context uintptr) {}
