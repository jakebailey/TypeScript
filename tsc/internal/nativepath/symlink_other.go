//go:build !windows

package nativepath

import "os"

// checkLocalPath accepts every path outside Windows, which alone has UNC and
// device link targets.
func checkLocalPath(path string) (PathKind, bool) {
	return PathUnknown, true
}

func IsSymlinkOrReparsePoint(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}
