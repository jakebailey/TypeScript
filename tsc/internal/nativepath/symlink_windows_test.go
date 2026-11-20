package nativepath

import (
	"encoding/binary"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/sys/windows"
	"gotest.tools/v3/assert"
)

func TestIsSymlinkOrReparsePoint(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	t.Run("regular file", func(t *testing.T) {
		t.Parallel()
		file := filepath.Join(tmp, "regular.txt")
		assert.NilError(t, os.WriteFile(file, []byte("hello"), 0o666))
		assert.Equal(t, IsSymlinkOrReparsePoint(file), false)
	})

	t.Run("regular directory", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(tmp, "regular-dir")
		assert.NilError(t, os.MkdirAll(dir, 0o777))
		assert.Equal(t, IsSymlinkOrReparsePoint(dir), false)
	})

	t.Run("junction point", func(t *testing.T) {
		t.Parallel()
		target := filepath.Join(tmp, "junction-target")
		link := filepath.Join(tmp, "junction-link")
		assert.NilError(t, os.MkdirAll(target, 0o777))
		mklink(t, target, link, true)
		assert.Equal(t, IsSymlinkOrReparsePoint(link), true)
	})

	t.Run("file symlink", func(t *testing.T) {
		t.Parallel()
		target := filepath.Join(tmp, "symlink-target.txt")
		link := filepath.Join(tmp, "symlink-link.txt")
		assert.NilError(t, os.WriteFile(target, []byte("hello"), 0o666))
		mklink(t, target, link, false)
		assert.Equal(t, IsSymlinkOrReparsePoint(link), true)
	})

	t.Run("directory symlink", func(t *testing.T) {
		t.Parallel()
		target := filepath.Join(tmp, "dir-symlink-target")
		link := filepath.Join(tmp, "dir-symlink-link")
		assert.NilError(t, os.MkdirAll(target, 0o777))
		mklink(t, target, link, false)
		assert.Equal(t, IsSymlinkOrReparsePoint(link), true)
	})

	t.Run("nonexistent path", func(t *testing.T) {
		t.Parallel()
		nonexistent := filepath.Join(tmp, "does-not-exist")
		assert.Equal(t, IsSymlinkOrReparsePoint(nonexistent), false)
	})

	t.Run("empty path", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, IsSymlinkOrReparsePoint(""), false)
	})

	t.Run("invalid path with null byte", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, IsSymlinkOrReparsePoint("invalid\x00path"), false)
	})
}

func TestIsSymlinkOrReparsePointLongPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Create a deeply nested path that exceeds 248 characters
	longPathBase := tmp
	pathComponent := "very_long_directory_name_to_exceed_max_path_limit_abcdefghijklmnopqrstuvwxyz"

	for len(longPathBase) < 250 {
		longPathBase = filepath.Join(longPathBase, pathComponent)
	}

	target := filepath.Join(longPathBase, "target")
	link := filepath.Join(longPathBase, "link")

	// Use \\?\ prefix to enable long path support for mklink
	longTarget := `\\?\` + target
	longLink := `\\?\` + link

	assert.NilError(t, os.MkdirAll(longTarget, 0o777))
	assert.NilError(t, exec.Command("cmd", "/c", "mklink", "/J", longLink, longTarget).Run())

	// With long path support enabled, this should work even for paths >= 248 chars
	assert.Equal(t, IsSymlinkOrReparsePoint(link), true)
}

func TestIsSymlinkOrReparsePointNestedInSymlink(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()

	// Create a structure: target/inner-target, link -> target, then check link/inner-link
	target := filepath.Join(tmp, "target")
	innerTarget := filepath.Join(target, "inner-target")
	assert.NilError(t, os.MkdirAll(innerTarget, 0o777))

	link := filepath.Join(tmp, "link")
	mklink(t, target, link, true)

	// Create a junction inside the target
	innerLink := filepath.Join(target, "inner-link")
	mklink(t, innerTarget, innerLink, true)

	// Check the junction through the symlink path
	nestedPath := filepath.Join(link, "inner-link")
	assert.Equal(t, IsSymlinkOrReparsePoint(nestedPath), true)
}

func TestIsSymlinkOrReparsePointRelativePath(t *testing.T) { //nolint:paralleltest // Cannot use t.Parallel() with t.Chdir()
	tmp := t.TempDir()
	t.Chdir(tmp)

	target := "target-rel"
	link := "link-rel"
	assert.NilError(t, os.MkdirAll(target, 0o777))
	mklink(t, target, link, true)

	assert.Equal(t, IsSymlinkOrReparsePoint(link), true)
	assert.Equal(t, IsSymlinkOrReparsePoint(target), false)
}

func TestCheckLocalPath(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	target := filepath.Join(tmp, "target")
	assert.NilError(t, os.MkdirAll(filepath.Join(target, "dir"), 0o777))
	assert.NilError(t, os.WriteFile(filepath.Join(target, "file.ts"), nil, 0o666))
	mklink(t, target, filepath.Join(tmp, "junction"), true)
	mklink(t, filepath.VolumeName(tmp)+`\`, filepath.Join(tmp, "volume"), true)
	dirSymlink(t, "target", filepath.Join(tmp, "relative"))
	// Never followed, so no connection is attempted.
	dirSymlink(t, `\\127.0.0.1\share`, filepath.Join(tmp, "unc"))
	dirSymlink(t, `..\..\unc`, filepath.Join(target, "dir", "chained"))
	// Link targets skip DOS normalization, so `target.` names an entry other
	// than `target`.
	dirSymlink(t, "target.", filepath.Join(tmp, "dotted"))

	// Windows normalizes `unc.` to `unc` when opening this path, including
	// when it is long enough to be opened through `\\?\`, so the probe must
	// normalize it too.
	long := filepath.Join(tmp, strings.Repeat("a", 200), strings.Repeat("b", 50))
	assert.NilError(t, os.MkdirAll(long, 0o777))
	dirSymlink(t, `\\127.0.0.1\share`, filepath.Join(long, "unc"))

	tmpVolume, err := volumeGUIDPath(target)
	assert.NilError(t, err)
	mklink(t, target, filepath.Join(tmp, "guid"), true)
	setJunctionTarget(t, filepath.Join(tmp, "guid"), strings.TrimPrefix(tmpVolume, `\\?\`))

	for _, path := range []string{
		target,
		filepath.Join(target, "dir"),
		filepath.Join(target, "missing", "file.ts"),
		filepath.Join(tmp, "junction"),
		filepath.Join(tmp, "junction", "dir", "file.ts"),
		filepath.Join(tmp, "relative", "dir"),
		filepath.Join(tmp, "volume", tmp[len(filepath.VolumeName(tmp)):], "target"),
		filepath.Join(tmp, "guid", "dir"),
	} {
		assert.Assert(t, isLocal(path), path)
	}
	for _, path := range []string{
		filepath.Join(tmp, "unc"),
		filepath.Join(tmp, "unc", "file.ts"),
		filepath.Join(target, "dir", "chained", "file.ts"),
		filepath.Join(tmp, "junction", "dir", "chained", "file.ts"),
		filepath.Join(tmp, "relative", "dir", "chained"),
		filepath.Join(tmp, "dotted", "file.ts"),
		filepath.Join(long, "unc.", "file.ts"),
		filepath.Join(target, "file.ts::$DATA"),
	} {
		assert.Assert(t, !isLocal(path), path)
	}
}

func isLocal(path string) bool {
	_, ok := CheckLocalPath(path)
	return ok
}

func assertKind(t *testing.T, path string, want PathKind) {
	t.Helper()
	kind, ok := CheckLocalPath(path)
	assert.Assert(t, ok, path)
	assert.Equal(t, kind, want, path)
}

func TestCheckLocalPathRetargetedJunction(t *testing.T) {
	t.Parallel()

	tmp := t.TempDir()
	withFile := filepath.Join(tmp, "with-file")
	withoutFile := filepath.Join(tmp, "without-file")
	assert.NilError(t, os.Mkdir(withFile, 0o777))
	assert.NilError(t, os.Mkdir(withoutFile, 0o777))
	assert.NilError(t, os.WriteFile(filepath.Join(withFile, "file.ts"), nil, 0o666))
	link := filepath.Join(tmp, "link")
	mklink(t, withFile, link, true)

	file := filepath.Join(link, "file.ts")
	assertKind(t, file, PathFile)
	setJunctionTarget(t, link, withoutFile)
	assertKind(t, file, PathMissing)
	assert.NilError(t, os.Remove(link))
	assertKind(t, link, PathMissing)
}

// setJunctionTarget points an existing junction at target in place.
func setJunctionTarget(t *testing.T, link string, target string) {
	t.Helper()
	substitute := utf16.Encode([]rune(`\??\` + target))
	pathBytes := 2 * (len(substitute) + 2)
	buffer := make([]byte, 16+pathBytes)
	binary.LittleEndian.PutUint32(buffer[0:], windows.IO_REPARSE_TAG_MOUNT_POINT)
	binary.LittleEndian.PutUint16(buffer[4:], uint16(8+pathBytes))
	binary.LittleEndian.PutUint16(buffer[10:], uint16(2*len(substitute)))
	binary.LittleEndian.PutUint16(buffer[12:], uint16(2*len(substitute)+2))
	for i, char := range substitute {
		binary.LittleEndian.PutUint16(buffer[16+2*i:], char)
	}

	name, err := windows.UTF16PtrFromString(link)
	assert.NilError(t, err)
	handle, err := windows.CreateFile(name, windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_OPEN_REPARSE_POINT|windows.FILE_FLAG_BACKUP_SEMANTICS, 0)
	assert.NilError(t, err)
	defer windows.CloseHandle(handle) //nolint:errcheck
	var n uint32
	assert.NilError(t, windows.DeviceIoControl(handle, windows.FSCTL_SET_REPARSE_POINT, &buffer[0], uint32(len(buffer)), nil, 0, &n, nil))
}

func TestCheckLocalPathSubst(t *testing.T) { //nolint:paralleltest // Drive letters are global.
	tmp := t.TempDir()
	dirSymlink(t, `\\127.0.0.1\share`, filepath.Join(tmp, "unc"))
	drive := ""
	for letter := 'Z'; letter > 'D'; letter-- {
		if _, err := os.Stat(string(letter) + `:\`); err != nil {
			drive = string(letter) + ":"
			break
		}
	}
	if drive == "" {
		t.Skip("no free drive letter")
	}
	assert.NilError(t, exec.Command("subst", drive, tmp).Run())
	t.Cleanup(func() { _ = exec.Command("subst", drive, "/D").Run() })

	assertKind(t, drive+`\`, PathDirectory)
	assertKind(t, drive+`\missing`, PathMissing)
	assert.Assert(t, !isLocal(drive+`\unc\file.ts`))
}

func volumeGUIDPath(path string) (string, error) {
	var volumePath [windows.MAX_PATH]uint16
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	if err := windows.GetVolumePathName(name, &volumePath[0], uint32(len(volumePath))); err != nil {
		return "", err
	}
	var guid [windows.MAX_PATH]uint16
	if err := windows.GetVolumeNameForVolumeMountPoint(&volumePath[0], &guid[0], uint32(len(guid))); err != nil {
		return "", err
	}
	mount := windows.UTF16ToString(volumePath[:])
	return windows.UTF16ToString(guid[:]) + path[len(mount):], nil
}

func BenchmarkIsSymlinkOrJunction(b *testing.B) {
	tmp := b.TempDir()

	regularFile := filepath.Join(tmp, "regular.txt")
	assert.NilError(b, os.WriteFile(regularFile, []byte("hello"), 0o666))

	target := filepath.Join(tmp, "target")
	link := filepath.Join(tmp, "link")
	assert.NilError(b, os.MkdirAll(target, 0o777))
	assert.NilError(b, exec.Command("cmd", "/c", "mklink", "/J", link, target).Run())

	b.Run("regular file", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			IsSymlinkOrReparsePoint(regularFile)
		}
	})

	b.Run("junction", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			IsSymlinkOrReparsePoint(link)
		}
	})

	b.Run("nonexistent", func(b *testing.B) {
		b.ReportAllocs()
		nonexistent := filepath.Join(tmp, "does-not-exist")
		for b.Loop() {
			IsSymlinkOrReparsePoint(nonexistent)
		}
	})
}

// dirSymlink creates a directory symlink without resolving its target.
func dirSymlink(tb testing.TB, target, link string) {
	tb.Helper()
	linkPtr, err := windows.UTF16PtrFromString(link)
	assert.NilError(tb, err)
	targetPtr, err := windows.UTF16PtrFromString(target)
	assert.NilError(tb, err)
	const allowUnprivilegedCreate = 0x2
	err = windows.CreateSymbolicLink(linkPtr, targetPtr, windows.SYMBOLIC_LINK_FLAG_DIRECTORY|allowUnprivilegedCreate)
	if errors.Is(err, windows.ERROR_PRIVILEGE_NOT_HELD) {
		tb.Skip("symlink support is not enabled without elevation or developer mode")
	}
	assert.NilError(tb, err)
}

func mklink(tb testing.TB, target, link string, isDir bool) {
	tb.Helper()

	if isDir {
		assert.NilError(tb, exec.Command("cmd", "/c", "mklink", "/J", link, target).Run())
	} else {
		err := os.Symlink(target, link)
		if err != nil && strings.Contains(err.Error(), "A required privilege is not held by the client") {
			tb.Log(err)
			tb.Skip("file symlink support is not enabled without elevation or developer mode")
		}
		assert.NilError(tb, err)
	}
}
