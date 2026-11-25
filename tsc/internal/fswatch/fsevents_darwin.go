//go:build darwin && (amd64 || arm64)

package fswatch

import (
	"errors"
	"fmt"
	"os"
	"runtime/cgo"
	"slices"
	"sort"
	"strings"
	"structs"
	"sync"
	"sync/atomic"
	"syscall"
	"unicode/utf8"
	"unsafe"

	"golang.org/x/sys/unix"
)

// ---------------------------------------------------------------------------
// fsevents_darwin.go: macOS FSEvents backend (event processing)
//
// Uses Apple's FSEvents API to receive file-level notifications for watched
// directory trees. FSEvents is a high-level, path-based API that watches
// recursively without requiring an fd per file (unlike kqueue). Events are
// coalesced by the kernel and delivered in batches.
//
// This file contains the event classification and stream lifecycle logic.
// The CoreFoundation, CoreServices, and libdispatch bindings live in
// fsevents_darwin_bindings.go (using Go's cgo bindings without a C compiler).
//
// Goroutines and threading:
//   - FSEvents delivers callbacks on a per-stream serial GCD dispatch queue
//     thread directly into the exported Go callback (fswatchFSEventsCallback).
//   - subscribe/closeWatch rebuild the stream chunks on the caller's goroutine
//     under watcherBase.mu. Old streams are swapped out before teardown so a
//     callback cannot deadlock against stream teardown while routing events.
//
// Callback delivery:
//   dirWatch.notify() posts to the shared process-wide debouncer. After a
//   coalescing window (50 ms min / 500 ms max), the debouncer invokes all
//   registered WatchCallbacks on its own dedicated goroutine; never on
//   the GCD thread or the caller's goroutine.
//   On all backends, events matching a WithIgnore function are filtered
//   per-subscriber before delivery.
//
// WatchDirectory flow (caller goroutine):
//   subscribe/closeWatch snapshots active dirWatches and creates one or more
//   FSEventStreams with kFSEventStreamEventIdSinceNow. Each stream receives a
//   chunk of physical watch roots. No directory walk or tree is needed;
//   FSEvents watches recursively via the kernel, and event classification uses
//   only the flags.
//
// Event classification (fsEventsProcessBatch, on GCD callback thread):
//   Each batch delivers arrays of paths, flags, and event IDs. The flags
//   bitmask may combine multiple states (created + modified + renamed).
//   Pure removes emit EventDelete with no syscalls. Renames and
//   remove+create combos do one Lstat to check existence (the kernel
//   reports some deletions as renames). Everything else emits EventUpdate
//   with no syscalls.
//
// Overflow:
//   flagMustScanSubDirs → ErrOverflow with detail (user/kernel/too-many).
//
// Root deletion:
//   Detected in the callback; the logical watch is marked terminated and
//   receives ErrWatchTerminated. The shared stream remains active for other
//   watches until the owner closes or reconciles the terminated watch.
// ---------------------------------------------------------------------------

// ----- FSEvents flag bits (from FSEvents.h) ------------------------------

const (
	flagMustScanSubDirs = 0x00000001
	flagUserDropped     = 0x00000002
	flagKernelDropped   = 0x00000004
	flagHistoryDone     = 0x00000010

	flagItemCreated        = 0x00000100
	flagItemRemoved        = 0x00000200
	flagItemInodeMetaMod   = 0x00000400
	flagItemRenamed        = 0x00000800
	flagItemModified       = 0x00001000
	flagItemFinderInfoMod  = 0x00002000
	flagItemChangeOwner    = 0x00004000
	flagItemXattrMod       = 0x00008000
	flagItemIsFile         = 0x00010000
	flagItemIsDir          = 0x00020000
	flagItemIsSymlink      = 0x00040000
	flagItemIsHardlink     = 0x00100000
	flagItemIsLastHardlink = 0x00200000
	flagItemCloned         = 0x00400000

	flagCreateUseCFTypes = 0x01
	flagCreateFileEvents = 0x10

	cfStringEncodingUTF8 = 0x08000100

	// kFSEventStreamEventIdSinceNow == ((FSEventStreamEventId)0xFFFFFFFFFFFFFFFFULL)
	eventIDSinceNow = uint64(0xFFFFFFFFFFFFFFFF)
)

const ignoredFlags = flagItemIsHardlink | flagItemIsLastHardlink |
	flagItemIsSymlink | flagItemIsDir | flagItemIsFile | flagItemCloned

const (
	cfStringNormalizationFormC = 2 // kCFStringNormalizationFormC
	nativePathFolding          = true
)

// isASCII reports whether every byte in s is below 0x80. Pure-ASCII paths
// are identical in every Unicode normalization form, so we can skip the
// CoreFoundation round-trip entirely, which is the overwhelming common case.
func isASCII(s string) bool {
	for i := range len(s) {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

// cfStringToNFC returns the CFString at src as a NFC-normalized Go string.
// If normalization fails, it falls back to the unnormalized UTF-8 contents.
// Returns "" only if both the normalized and unnormalized conversions fail
// (e.g. src is not a CFString, or allocation fails).
func cfStringToNFC(src uintptr) string {
	if src == 0 {
		return ""
	}
	if s := cfStringNormalizedToGo(src); s != "" {
		return s
	}
	return cfStringToGo(src)
}

// cfStringNormalizedToGo returns the CFString at src as a NFC-normalized Go
// string, or "" on any failure.
func cfStringNormalizedToGo(src uintptr) string {
	mut := cfStringCreateMutableCopy(0, 0, src)
	if mut == 0 {
		return ""
	}
	defer cfRelease(mut)

	cfStringNormalize(mut, cfStringNormalizationFormC)
	return cfStringToGo(mut)
}

// cfStringToGo extracts the UTF-8 contents of the CFString at src as a Go
// string, or "" on failure.
func cfStringToGo(src uintptr) string {
	length := cfStringGetLength(src)
	bufSize := cfStringGetMaximumSizeForEncoding(length, cfStringEncodingUTF8) + 1
	buf := make([]byte, bufSize)
	if cfStringGetCString(src, &buf[0], bufSize, cfStringEncodingUTF8) == 0 {
		return ""
	}
	// CFStringGetCString writes a NUL terminator; trim it.
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n])
}

// normalizeNFC returns s in Unicode NFC (canonical composed) form. ASCII
// inputs are returned unchanged. Non-ASCII inputs go through CoreFoundation;
// if any step fails (e.g. invalid UTF-8 from a corrupt path), the original
// string is returned so the caller still sees *something* rather than nothing.
func normalizeNFC(s string) string {
	if isASCII(s) {
		return s
	}

	cstr := append([]byte(s), 0)
	src := cfStringCreate(0, &cstr[0], cfStringEncodingUTF8)
	if src == 0 {
		return s
	}
	defer cfRelease(src)

	normalized := cfStringToNFC(src)
	if normalized == "" {
		return s
	}
	return normalized
}

// foldNativePath is a comparison form, never a displayed or opened path.
// Case folding expands sharp s and ligatures without making diacritics,
// dotless i, circled letters, or character widths interchangeable.
func foldNativePath(s string) string {
	if isASCII(s) {
		return strings.ToLower(s)
	}
	if !utf8.ValidString(s) || strings.IndexByte(s, 0) >= 0 {
		return ""
	}
	cstr := append([]byte(s), 0)
	src := cfStringCreate(0, &cstr[0], cfStringEncodingUTF8)
	if src == 0 {
		panic("fswatch: cannot create CFString for path folding")
	}
	defer cfRelease(src)
	mut := cfStringCreateMutableCopy(0, 0, src)
	if mut == 0 {
		panic("fswatch: cannot copy CFString for path folding")
	}
	defer cfRelease(mut)
	// Normalize before folding as well: a decomposed capital I with dot
	// must have the same comparison form as precomposed dotted capital I.
	cfStringNormalize(mut, cfStringNormalizationFormC)
	const cfCompareCaseInsensitive = 1
	cfStringFold(mut, cfCompareCaseInsensitive, 0)
	cfStringNormalize(mut, cfStringNormalizationFormC)
	folded := cfStringToGo(mut)
	if folded == "" {
		panic("fswatch: cannot extract folded CFString")
	}
	return folded
}

// fsEventStreamContext mirrors the C struct of the same name.
//
//	typedef struct {
//	  CFIndex version;          // signed long, 8 bytes on 64-bit
//	  void   *info;             // pointer
//	  void   *retain;           // pointer
//	  void   *release;          // pointer
//	  void   *copyDescription;  // pointer
//	} FSEventStreamContext;
type fsEventStreamContext struct {
	_               structs.HostLayout
	version         int64
	info            uintptr
	retain          uintptr
	release         uintptr
	copyDescription uintptr
}

type fseventsState struct {
	terminated atomic.Bool
}

type fseventsStream struct {
	stream  atomic.Uintptr
	queue   uintptr
	handle  cgo.Handle
	watches []fseventsWatchSnapshot
}

// ----- the watcherImpl -------------------------------------------------------

// fsEventsBackend.
type fsEventsBackend struct {
	watcherBase

	mu      sync.Mutex
	watches map[*dirWatch]*fseventsState
	streams []*fseventsStream
}

func init() {
	fseventsWatcher.factory = func() watcherImpl { return newFSEventsBackend() }
	fseventsWatcher.sequence = fsEventsGetCurrentEventID
}

func newFSEventsBackend() *fsEventsBackend {
	b := &fsEventsBackend{
		watches: make(map[*dirWatch]*fseventsState),
	}
	b.watcherBase.init(b)
	return b
}

func (b *fsEventsBackend) start() error {
	b.notifyStarted()
	return nil
}

// checkWatcher mirrors the helper of the same name.
func checkWatcher(w *dirWatch) error {
	info, err := os.Stat(w.physicalDir)
	if err != nil {
		return &dirWatchError{err: err, dirWatch: w}
	}
	if !info.IsDir() {
		return &dirWatchError{err: syscall.ENOTDIR, dirWatch: w}
	}
	return nil
}

var (
	errCFStringCreateNull = errors.New("CFStringCreate returned NULL")
	errCFArrayCreateNull  = errors.New("CFArrayCreate returned NULL")
	errStreamCreateNull   = errors.New("FSEventStreamCreate returned NULL")
	errStreamStartFailed  = errors.New("error starting FSEvents stream")
)

var (
	errFSEventsUserDropped   = fmt.Errorf("events were dropped by the FSEvents client: %w", ErrOverflow)
	errFSEventsKernelDropped = fmt.Errorf("events were dropped by the kernel: %w", ErrOverflow)
	errFSEventsTooMany       = fmt.Errorf("too many events: %w", ErrOverflow)
)

const fseventsPathsPerStream = 512

type fseventsWatchSnapshot struct {
	w     *dirWatch
	state *fseventsState
}

func (b *fsEventsBackend) activeWatchesLocked() []fseventsWatchSnapshot {
	watches := make([]fseventsWatchSnapshot, 0, len(b.watches))
	for w, state := range b.watches {
		if state.terminated.Load() {
			continue
		}
		watches = append(watches, fseventsWatchSnapshot{w: w, state: state})
	}
	return watches
}

func (b *fsEventsBackend) startStreams(watches []fseventsWatchSnapshot) ([]*fseventsStream, error) {
	return startFSEventsStreams(watches, b.startStream)
}

func startFSEventsStreams(watches []fseventsWatchSnapshot, startStream func([]string, []fseventsWatchSnapshot) (*fseventsStream, error)) ([]*fseventsStream, error) {
	if len(watches) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(watches))
	paths := make([]string, 0, len(watches))
	for _, watch := range watches {
		path := watch.w.physicalDir
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		paths = append(paths, path)
	}
	sort.Strings(paths)

	stream, err := startStream(paths, watches)
	if err == nil {
		return []*fseventsStream{stream}, nil
	}

	streams := make([]*fseventsStream, 0, (len(paths)+fseventsPathsPerStream-1)/fseventsPathsPerStream)
	remainingPaths := paths
	for len(remainingPaths) > 0 {
		chunkLen := min(len(remainingPaths), fseventsPathsPerStream)
		chunkPaths := remainingPaths[:chunkLen]
		stream, err := startStream(chunkPaths, watchesForFSEventsPaths(watches, chunkPaths))
		if err != nil {
			stopFSEventsStreams(streams)
			return nil, err
		}
		streams = append(streams, stream)
		remainingPaths = remainingPaths[chunkLen:]
	}
	return streams, nil
}

func watchesForFSEventsPaths(watches []fseventsWatchSnapshot, paths []string) []fseventsWatchSnapshot {
	if len(paths) == 0 {
		return nil
	}
	filtered := make([]fseventsWatchSnapshot, 0, len(watches))
	for _, watch := range watches {
		if _, ok := slices.BinarySearch(paths, watch.w.physicalDir); ok {
			filtered = append(filtered, watch)
		}
	}
	return filtered
}

// startStream creates and starts one FSEventStream watching all supplied paths.
func (b *fsEventsBackend) startStream(paths []string, watches []fseventsWatchSnapshot) (*fseventsStream, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	cfStrings := make([]uintptr, 0, len(paths))
	for _, path := range paths {
		dirCStr := append([]byte(path), 0)
		cfDir := cfStringCreate(0, &dirCStr[0], cfStringEncodingUTF8)
		if cfDir == 0 {
			for _, cfString := range cfStrings {
				cfRelease(cfString)
			}
			return nil, errCFStringCreateNull
		}
		cfStrings = append(cfStrings, cfDir)
	}
	defer func() {
		for _, cfString := range cfStrings {
			cfRelease(cfString)
		}
	}()

	pathsToWatch := cfArrayCreate(0, &cfStrings[0], int64(len(cfStrings)), 0)
	if pathsToWatch == 0 {
		return nil, errCFArrayCreateNull
	}
	defer cfRelease(pathsToWatch)

	label := []byte("typescript.fswatch.fsevents.stream\x00")
	queue := dispatchQueueCreate(&label[0], 0)
	if queue == 0 {
		return nil, errStreamCreateNull
	}

	state := &fseventsStream{
		queue:   queue,
		watches: slices.Clone(watches),
	}
	state.handle = cgo.NewHandle(state)

	ctx := fsEventStreamContext{info: uintptr(state.handle)}
	stream := fsEventStreamCreate(
		0,
		fsEventsCallbackPtr(),
		&ctx,
		pathsToWatch,
		eventIDSinceNow,
		0.001,
		flagCreateUseCFTypes|flagCreateFileEvents,
	)
	if stream == 0 {
		dispatchRelease(queue)
		state.handle.Delete()
		return nil, errStreamCreateNull
	}

	fsEventStreamSetDispatchQueue(stream, queue)
	if fsEventStreamStart(stream) == 0 {
		fsEventStreamInvalidate(stream)
		fsEventStreamRelease(stream)
		dispatchRelease(queue)
		state.handle.Delete()
		return nil, errStreamStartFailed
	}
	fsEventStreamFlushSync(stream)
	state.stream.Store(stream)
	return state, nil
}

// The atomic Swap gates teardown so concurrent or repeated calls are safe:
// only the goroutine that observes a non-zero stream performs the cleanup.
func stopFSEventsStreams(streams []*fseventsStream) {
	for _, stream := range streams {
		stopFSEventsStream(stream)
	}
}

func stopFSEventsStream(state *fseventsStream) {
	if state == nil {
		return
	}
	stream := state.stream.Swap(0)
	if stream == 0 {
		return
	}
	fsEventStreamStop(stream)
	fsEventStreamInvalidate(stream)
	if state.queue != 0 {
		dispatchSync(state.queue, 0, dispatchNoopPtr())
		dispatchRelease(state.queue)
		state.queue = 0
	}
	fsEventStreamRelease(stream)
	if state.handle != 0 {
		state.handle.Delete()
		state.handle = 0
	}
}

// subscribe mirrors `fsEventsBackend::subscribe`.
func (b *fsEventsBackend) subscribe(w *dirWatch) error {
	return b.subscribeMany([]*dirWatch{w})
}

func (b *fsEventsBackend) subscribeMany(watchesToAdd []*dirWatch) error {
	if len(watchesToAdd) == 0 {
		return nil
	}
	states := make(map[*dirWatch]*fseventsState, len(watchesToAdd))
	for _, w := range watchesToAdd {
		if err := checkWatcher(w); err != nil {
			return err
		}
		states[w] = &fseventsState{}
	}

	b.mu.Lock()
	for w, state := range states {
		w.state = state
		b.watches[w] = state
	}
	watches := b.activeWatchesLocked()
	b.mu.Unlock()

	streams, err := b.startStreams(watches)
	if err != nil {
		b.mu.Lock()
		for w, state := range states {
			if b.watches[w] == state {
				delete(b.watches, w)
				w.state = nil
			}
		}
		b.mu.Unlock()
		return &dirWatchError{err: err, dirWatch: watchesToAdd[0]}
	}

	b.mu.Lock()
	oldStreams := b.streams
	b.streams = streams
	b.mu.Unlock()
	stopFSEventsStreams(oldStreams)
	return nil
}

// closeWatch mirrors `fsEventsBackend::closeWatch`.
func (b *fsEventsBackend) closeWatch(w *dirWatch) error {
	state, _ := w.state.(*fseventsState)
	w.state = nil
	if state == nil {
		return nil
	}
	state.terminated.Store(true)

	b.mu.Lock()
	delete(b.watches, w)
	watches := b.activeWatchesLocked()
	b.mu.Unlock()

	streams, err := b.startStreams(watches)
	if err != nil {
		return err
	}

	b.mu.Lock()
	oldStreams := b.streams
	b.streams = streams
	b.mu.Unlock()
	stopFSEventsStreams(oldStreams)
	return nil
}

// fsEventsProcessBatch processes a batch of FSEvents.
func fsEventsProcessBatch(watches []fseventsWatchSnapshot, numEvents uint64, paths uintptr, eventFlags *uint32, eventIDs *uint64) {
	if paths == 0 || eventFlags == nil || eventIDs == nil || numEvents == 0 {
		return
	}

	flags := unsafe.Slice(eventFlags, numEvents)
	ids := unsafe.Slice(eventIDs, numEvents)
	touched := map[*dirWatch]struct{}{}

	for i := range numEvents {
		flag := flags[i]
		eventID := ids[i]
		pathRef := cfArrayGetValueAtIndex(paths, int64(i))
		path := cfStringToNFC(pathRef)
		if path == "" {
			continue
		}
		comparison := comparisonPath{path: path}

		isRemoved := flag&flagItemRemoved != 0
		isRenamed := flag&flagItemRenamed != 0
		isCreated := flag&flagItemCreated != 0
		isDone := flag&flagHistoryDone != 0

		if flag&flagMustScanSubDirs != 0 {
			var overflow error
			switch {
			case flag&flagUserDropped != 0:
				overflow = errFSEventsUserDropped
			case flag&flagKernelDropped != 0:
				overflow = errFSEventsKernelDropped
			default:
				overflow = errFSEventsTooMany
			}
			for _, watch := range watches {
				if watch.state.terminated.Load() {
					continue
				}
				if fseventsOverflowMatchesPrepared(watch.w, &comparison) {
					watch.w.events.setError(overflow)
					touched[watch.w] = struct{}{}
				}
			}
		}

		if isDone {
			break
		}

		if flag&^uint32(ignoredFlags) == 0 {
			continue
		}

		rawPath := path
		pathExists := false
		pathExistsKnown := false

		for _, watch := range watches {
			if watch.state.terminated.Load() {
				continue
			}
			w := watch.w
			displayPath, ok := fseventsDisplayPathPrepared(w, &comparison)
			if !ok {
				continue
			}

			// Skip events for the watched directory itself unless it's been
			// removed. fseventsd reports a change on the watched dir when a
			// child is added or removed; subscribers observe changes *within*
			// the directory, not the dir's own metadata churn.
			// (A removal of the dir is still propagated because Watcher
			// relies on it to tear down the stream.)
			if displayPath == w.dir && !isRemoved && !isRenamed {
				continue
			}

			switch {
			case isRemoved && !isCreated:
				if displayPath == w.dir {
					w.events.removeWatchRootAt(displayPath, eventID)
				} else {
					w.events.removeAt(displayPath, eventID)
				}
				if w.terminateCallbacksForDeletedRoot(displayPath, eventID, fmt.Errorf("%w: watched directory removed", ErrWatchTerminated)) {
					touched[w] = struct{}{}
				}
				if displayPath == w.dir {
					watch.state.terminated.Store(true)
					w.events.setError(fmt.Errorf("%w: watched directory removed", ErrWatchTerminated))
				}
			case isRenamed || (isRemoved && isCreated):
				if !pathExistsKnown {
					var st unix.Stat_t
					pathExists = unix.Lstat(rawPath, &st) == nil
					pathExistsKnown = true
				}
				if pathExists {
					if displayPath == w.dir {
						w.events.updateWatchRootAt(displayPath, eventID)
					} else {
						w.events.updateAt(displayPath, eventID)
					}
				} else {
					if displayPath == w.dir {
						w.events.removeWatchRootAt(displayPath, eventID)
					} else {
						w.events.removeAt(displayPath, eventID)
					}
					if w.terminateCallbacksForDeletedRoot(displayPath, eventID, fmt.Errorf("%w: watched directory removed", ErrWatchTerminated)) {
						touched[w] = struct{}{}
					}
					if displayPath == w.dir {
						watch.state.terminated.Store(true)
						w.events.setError(fmt.Errorf("%w: watched directory removed", ErrWatchTerminated))
					}
				}
			default:
				if displayPath == w.dir {
					w.events.updateWatchRootAt(displayPath, eventID)
				} else {
					w.events.updateAt(displayPath, eventID)
				}
			}
			touched[w] = struct{}{}
		}
	}

	for w := range touched {
		w.notify()
	}
}

func fseventsDisplayPath(w *dirWatch, rawPath string) (string, bool) {
	path := comparisonPath{path: rawPath}
	return fseventsDisplayPathPrepared(w, &path)
}

func fseventsDisplayPathPrepared(w *dirWatch, rawPath *comparisonPath) (string, bool) {
	physical := comparisonPath{path: w.physicalDir, folded: w.physicalDirFold, ready: w.physicalDirFold != ""}
	if path, ok := w.comparer.rebasePrepared(rawPath, physical, w.dir); ok {
		return path, true
	}
	if w.physicalDir != w.dir {
		logical := comparisonPath{path: w.dir, folded: w.dirFold, ready: w.dirFold != ""}
		return w.comparer.rebasePrepared(rawPath, logical, w.dir)
	}
	return "", false
}

func fseventsOverflowMatches(w *dirWatch, rawPath string) bool {
	path := comparisonPath{path: rawPath}
	return fseventsOverflowMatchesPrepared(w, &path)
}

func fseventsOverflowMatchesPrepared(w *dirWatch, rawPath *comparisonPath) bool {
	physical := comparisonPath{path: w.physicalDir, folded: w.physicalDirFold, ready: w.physicalDirFold != ""}
	if _, ok := w.comparer.suffixPrepared(physical, rawPath); ok {
		return true
	}
	if _, ok := w.comparer.suffixPrepared(*rawPath, &physical); ok {
		return true
	}
	if w.physicalDir != w.dir {
		logical := comparisonPath{path: w.dir, folded: w.dirFold, ready: w.dirFold != ""}
		if _, ok := w.comparer.suffixPrepared(logical, rawPath); ok {
			return true
		}
		_, ok := w.comparer.suffixPrepared(*rawPath, &logical)
		return ok
	}
	return false
}
