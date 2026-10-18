package main

import (
	"bytes"
	"os"
	"runtime/metrics"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

// startHugePageAdvisor backs the Go heap with transparent huge pages.
//
// The checker's working set is large and pointer-heavy, so 4 KiB pages make it
// TLB-bound. The Go runtime does not request huge pages for the heap, and most
// distributions only grant them on request (THP "madvise" mode). Each heap
// growth is a fresh mapping without the advice, so while the program is
// allocating this periodically advises the heap mappings and collapses ranges
// that were faulted in as small pages. Once allocation stops (for example, in
// watch mode), polling backs off.
func startHugePageAdvisor() {
	if strings.Contains(os.Getenv("GODEBUG"), "disablethp=1") {
		return
	}
	go func() {
		const minInterval = 5 * time.Millisecond
		const maxInterval = time.Second
		const collapseInterval = 250 * time.Millisecond
		sample := []metrics.Sample{{Name: "/gc/heap/allocs:bytes"}}
		interval := minInterval
		const activeAllocBytes = 1 << 20
		var buf []byte
		var lastAllocs uint64
		var lastCollapse time.Time
		for {
			metrics.Read(sample)
			if allocs := sample[0].Value.Uint64(); allocs-lastAllocs >= activeAllocBytes {
				lastAllocs = allocs
				interval = minInterval
				collapse := time.Since(lastCollapse) >= collapseInterval
				var ok bool
				buf, ok = adviseHeapMappings(buf, collapse)
				if !ok {
					return
				}
				if collapse {
					lastCollapse = time.Now()
				}
			} else {
				interval = min(interval*2, maxInterval)
			}
			time.Sleep(interval)
		}
	}()
}

func adviseHeapMappings(buf []byte, collapse bool) ([]byte, bool) {
	buf, ok := readProcSelfMaps(buf)
	if !ok {
		return buf, false
	}
	for line := range bytes.Lines(buf) {
		if !bytes.HasSuffix(bytes.TrimSpace(line), []byte("[anon: Go: heap]")) {
			continue
		}
		dash := bytes.IndexByte(line, '-')
		space := bytes.IndexByte(line, ' ')
		if dash < 0 || space < dash {
			continue
		}
		lo, ok1 := parseHex(line[:dash])
		hi, ok2 := parseHex(line[dash+1 : space])
		if !ok1 || !ok2 || hi <= lo {
			continue
		}
		if _, _, errno := unix.Syscall(unix.SYS_MADVISE, lo, hi-lo, unix.MADV_HUGEPAGE); errno != 0 {
			return buf, false
		}
		if collapse {
			_, _, _ = unix.Syscall(unix.SYS_MADVISE, lo, hi-lo, unix.MADV_COLLAPSE)
		}
	}
	return buf, true
}

// readProcSelfMaps reads /proc/self/maps into buf, reusing its capacity so that
// polling does not itself allocate.
func readProcSelfMaps(buf []byte) ([]byte, bool) {
	fd, err := unix.Open("/proc/self/maps", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return buf, false
	}
	defer unix.Close(fd)
	buf = buf[:0]
	for {
		if len(buf) == cap(buf) {
			buf = append(buf, 0)[:len(buf)]
		}
		n, err := unix.Read(fd, buf[len(buf):cap(buf)])
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return buf, false
		}
		if n == 0 {
			return buf, true
		}
		buf = buf[:len(buf)+n]
	}
}

func parseHex(b []byte) (uintptr, bool) {
	if len(b) == 0 {
		return 0, false
	}
	var v uintptr
	for _, c := range b {
		switch {
		case '0' <= c && c <= '9':
			v = v<<4 | uintptr(c-'0')
		case 'a' <= c && c <= 'f':
			v = v<<4 | uintptr(c-'a'+10)
		default:
			return 0, false
		}
	}
	return v, true
}
