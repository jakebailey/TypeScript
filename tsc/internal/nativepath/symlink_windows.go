package nativepath

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

// maxLinks matches the number of reparse points Windows follows when
// opening a path.
const maxLinks = 63

// checkLocalPath checks each link's target before following it. Following
// a link to a UNC or device path can make Windows authenticate to an
// arbitrary host, leaking credentials.
//
// Relative targets and absolute targets on drives (`C:\...`) or volume GUID
// paths (`\\?\Volume{...}\...`) are followed. Any other target, and any
// path that cannot be resolved this way, is refused.
func checkLocalPath(path string) (PathKind, bool) {
	full, err := filepath.Abs(path)
	if err != nil {
		return PathUnknown, false
	}
	return probe(full, 0)
}

// checkedDirectories holds directories found to resolve without rooted
// links, so probes of paths beneath them skip checking them again.
var (
	checkedDirectories      sync.Map // map[string]struct{}
	checkedDirectoriesCount atomic.Int32
)

const maxCheckedDirectories = 10_000

func rememberCheckedDirectory(dir string) {
	if _, loaded := checkedDirectories.LoadOrStore(dir, struct{}{}); !loaded {
		if checkedDirectoriesCount.Add(1) > maxCheckedDirectories {
			clearLocalPathCache()
		}
	}
}

func clearLocalPathCache() {
	checkedDirectories.Clear()
	checkedDirectoriesCount.Store(0)
}

func probe(full string, links int) (PathKind, bool) {
	volume := filepath.VolumeName(full)
	// The compiler never opens alternate data streams (`file:stream`).
	if !isLocalVolume(volume) || strings.ContainsRune(full[len(volume):], ':') {
		return PathUnknown, false
	}
	if len(full) <= len(volume)+1 {
		return PathDirectory, true
	}
	if _, found := checkedDirectories.Load(filepath.Dir(full)); !found {
		if kind, ok := probe(filepath.Dir(full), links); kind != PathDirectory {
			return PathMissing, ok
		}
	}
	info, err := os.Lstat(full)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return PathMissing, true
		}
		return PathUnknown, false
	}
	attributes := info.Sys().(*syscall.Win32FileAttributeData).FileAttributes
	if attributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0 {
		tag, ok := reparseTag(full)
		if !ok {
			return PathUnknown, false
		}
		if tag&reparseTagNameSurrogate != 0 {
			if links == maxLinks {
				return PathUnknown, false
			}
			target, err := os.Readlink(full)
			if err != nil {
				return PathUnknown, false
			}
			resolved, ok := resolveTarget(filepath.Dir(full), target)
			if !ok {
				return PathUnknown, false
			}
			kind, ok := probe(resolved, links+1)
			if kind == PathDirectory {
				rememberCheckedDirectory(full)
			}
			return kind, ok
		}
		// Other reparse points, such as cloud placeholders and deduplicated
		// files, have no target.
	}
	if attributes&syscall.FILE_ATTRIBUTE_DIRECTORY == 0 {
		return PathFile, true
	}
	rememberCheckedDirectory(full)
	return PathDirectory, true
}

const reparseTagNameSurrogate = 0x20000000

// reparseTag returns the tag of the reparse point at path.
func reparseTag(path string) (uint32, bool) {
	name, err := syscall.UTF16PtrFromString(fixLongPath(path))
	if err != nil {
		return 0, false
	}
	var data syscall.Win32finddata
	handle, err := syscall.FindFirstFile(name, &data)
	if err != nil {
		return 0, false
	}
	_ = syscall.FindClose(handle)
	return data.Reserved0, true
}

// fixLongPath prefixes long paths with `\\?\` so [syscall.FindFirstFile]
// accepts them.
func fixLongPath(path string) string {
	if len(path) < 248 || strings.HasPrefix(path, `\\`) {
		return path
	}
	return `\\?\` + path
}

// resolveTarget resolves the target of a link in dir, reporting false for
// targets [checkLocalPath] does not follow.
func resolveTarget(dir string, target string) (string, bool) {
	volume := filepath.VolumeName(target)
	if volume == "" {
		if target == "" || target[0] == '\\' {
			return "", false
		}
	} else if !isLocalVolume(volume) || len(target) == len(volume) || target[len(volume)] != '\\' {
		// Drive-relative targets like `C:dir` depend on the working directory.
		return "", false
	}
	// Windows resolves link targets without DOS normalization, so a component
	// that [os.Lstat] would normalize could name a different entry.
	if components := strings.Trim(target[len(volume):], `\`); components != "" {
		for component := range strings.SplitSeq(components, `\`) {
			if component != ".." && !isPlainComponent(component) {
				return "", false
			}
		}
	}
	if volume == "" {
		return filepath.Join(dir, target), true
	}
	return target, true
}

// isLocalVolume reports whether volume is a drive (`C:`) or a volume GUID
// path (`\\?\Volume{...}`).
func isLocalVolume(volume string) bool {
	if len(volume) == 2 {
		return volume[1] == ':' && ('a' <= volume[0]|0x20 && volume[0]|0x20 <= 'z')
	}
	guid, found := strings.CutPrefix(volume, `\\?\Volume{`)
	return found && strings.HasSuffix(guid, "}")
}

// isPlainComponent reports whether component has no syntax that DOS paths
// treat specially.
func isPlainComponent(component string) bool {
	if component == "" || component == "." || component == ".." || strings.ContainsAny(component, ":/") {
		return false
	}
	last := component[len(component)-1]
	return last != '.' && last != ' '
}

func IsSymlinkOrReparsePoint(path string) bool {
	if len(path) >= 248 {
		path = `\\?\` + path
	}

	pathUTF16, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return false
	}

	var data syscall.Win32FileAttributeData
	// SAFETY: GetFileExInfoStandard writes a Win32FileAttributeData, which data is.
	err = syscall.GetFileAttributesEx(
		pathUTF16,
		syscall.GetFileExInfoStandard,
		(*byte)(unsafe.Pointer(&data)),
	)
	if err != nil {
		return false
	}

	return data.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
