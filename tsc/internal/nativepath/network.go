package nativepath

// IsNetworkPath reports whether Windows may resolve path over the network.
// This includes UNC paths (`//server/share`), Win32 device paths (`//?/...`,
// `//./...`), and NT object paths (`/??/...`), with either separator. Device
// and object paths are included even when they name local volumes, because
// their `UNC` and `GLOBALROOT` forms reach the network redirector.
//
// Accessing such a path can make Windows authenticate to an arbitrary host,
// leaking credentials, so these paths are never accessed on any platform.
func IsNetworkPath(path string) bool {
	if len(path) < 2 || !isAnyDirectorySeparator(path[0]) {
		return false
	}
	if isAnyDirectorySeparator(path[1]) {
		return true
	}
	return len(path) >= 3 && path[1] == '?' && path[2] == '?' && (len(path) == 3 || isAnyDirectorySeparator(path[3]))
}

func isAnyDirectorySeparator(char byte) bool {
	return char == '/' || char == '\\'
}

// PathKind is the kind of entry a path names, as found by [CheckLocalPath].
type PathKind uint8

const (
	// PathUnknown means the path was not examined; access it normally.
	PathUnknown PathKind = iota
	PathMissing
	PathFile
	PathDirectory
)

// CheckLocalPath reports whether resolving path stays on local volumes, and
// on Windows, the kind of entry it names. Paths for which ok is false must
// not be accessed.
//
// Network paths (see [IsNetworkPath]) are refused on every platform. On
// Windows, links are also resolved one at a time, and a path is refused if
// it passes through a link whose target could reach the network.
func CheckLocalPath(path string) (kind PathKind, ok bool) {
	if IsNetworkPath(path) {
		return PathUnknown, false
	}
	return checkLocalPath(path)
}

// ClearLocalPathCache makes later [CheckLocalPath] calls check every
// directory again. Call it when a build starts, so that no build relies on
// checks made against an earlier state of the file system.
func ClearLocalPathCache() {
	clearLocalPathCache()
}
