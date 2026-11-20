package osvfs_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/nativepath"
	"github.com/microsoft/TypeScript/tsc/internal/repo"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"gotest.tools/v3/assert"
)

func TestGetGlobalTypingsCacheLocation(t *testing.T) {
	t.Run("user cache", func(t *testing.T) {
		cacheRoot := t.TempDir()
		t.Setenv("HOME", cacheRoot)
		t.Setenv("home", cacheRoot)
		t.Setenv("LocalAppData", cacheRoot)
		t.Setenv("XDG_CACHE_HOME", cacheRoot)
		cacheDir, err := os.UserCacheDir()
		assert.NilError(t, err)

		subdir := "typescript"
		if runtime.GOOS == "windows" {
			subdir = "Microsoft/TypeScript"
		}
		assert.Equal(t, osvfs.GetGlobalTypingsCacheLocation(), tspath.RootedDirectoryPathFromAbsolute(tspath.CombinePaths(cacheDir, subdir, core.VersionMajorMinor())))
	})

	t.Run("missing user cache", func(t *testing.T) {
		for _, name := range []string{"HOME", "home", "LocalAppData", "XDG_CACHE_HOME"} {
			t.Setenv(name, "")
		}
		assert.Equal(t, osvfs.GetGlobalTypingsCacheLocation(), tspath.RootedDirectoryPath(""))
	})

	t.Run("relative XDG cache", func(t *testing.T) {
		switch runtime.GOOS {
		case "windows", "darwin", "ios", "plan9":
			t.Skip("XDG_CACHE_HOME is not used on this platform")
		}
		t.Setenv("XDG_CACHE_HOME", "relative")
		assert.Equal(t, osvfs.GetGlobalTypingsCacheLocation(), tspath.RootedDirectoryPath(""))
	})

	t.Run("relative user cache", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", "")
		for _, name := range []string{"HOME", "home", "LocalAppData"} {
			t.Setenv(name, "relative-home")
		}
		assert.Equal(t, osvfs.GetGlobalTypingsCacheLocation(), tspath.RootedDirectoryPath(""))
	})
}

func TestOS(t *testing.T) {
	t.Parallel()

	fs := osvfs.FS()

	t.Run("ReadFile", func(t *testing.T) {
		t.Parallel()

		goMod := filepath.Join(repo.RootPath(), "go.mod")
		goModPath := tspath.RootedFilePathFromAbsolute(goMod)

		expectedRaw, err := os.ReadFile(goMod)
		assert.NilError(t, err)
		expected := string(expectedRaw)

		contents, ok := fs.ReadFile(goModPath)
		assert.Assert(t, ok)
		assert.Equal(t, contents, expected)
	})

	t.Run("Realpath", func(t *testing.T) {
		t.Parallel()

		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip(err)
		}
		homePath := tspath.RootedDirectoryPathFromAbsolute(home)

		expected := homePath.AsString()
		if runtime.GOOS == "windows" {
			// Windows drive letters can be lowercase, but realpath will always return uppercase.
			expected = strings.ToUpper(expected[:1]) + expected[1:]
		}
		realpath := fs.Realpath(homePath.AsPath())
		assert.Equal(t, realpath.AsString(), expected)
	})

	t.Run("NetworkPaths", func(t *testing.T) {
		t.Parallel()

		if runtime.GOOS == "windows" {
			t.Skip("requires POSIX paths where a leading // resolves locally")
		}

		// On POSIX systems, //dir resolves to /dir, so these paths name real
		// files; the file system must refuse to touch them anyway.
		dir := t.TempDir()
		assert.NilError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o666))

		networkDir := tspath.RootedDirectoryPathFromAbsolute("/" + dir)
		networkFile := tspath.RootedFilePathFromAbsolute("/" + dir + "/a.txt")
		newNetworkFile := tspath.RootedFilePathFromAbsolute("/" + dir + "/b.txt")
		assert.Assert(t, nativepath.IsNetworkPath(networkFile.AsString()))

		assert.Assert(t, !fs.DirectoryExists(networkDir))
		assert.Assert(t, !fs.FileExists(networkFile))
		_, ok := fs.ReadFile(networkFile)
		assert.Assert(t, !ok)
		assert.Assert(t, fs.Stat(networkFile.AsPath()) == nil)
		entries := fs.GetAccessibleEntries(networkDir)
		assert.Equal(t, len(entries.Files), 0)
		assert.Equal(t, len(entries.Directories), 0)
		assert.Equal(t, fs.Realpath(networkFile.AsPath()), networkFile.AsPath())

		assert.Assert(t, fs.WriteFile(newNetworkFile, "b") != nil)
		assert.Assert(t, fs.AppendFile(newNetworkFile, "b") != nil)
		assert.Assert(t, fs.Chtimes(networkFile.AsPath(), time.Now(), time.Now()) != nil)
		assert.Assert(t, fs.Remove(networkFile.AsPath()) != nil)

		_, err := os.Stat(filepath.Join(dir, "a.txt"))
		assert.NilError(t, err)
		_, err = os.Stat(filepath.Join(dir, "b.txt"))
		assert.Assert(t, os.IsNotExist(err))
	})

	t.Run("CaseSensitivity", func(t *testing.T) {
		t.Parallel()

		// Just check that it works.
		fs.CaseSensitivity()

		switch runtime.GOOS {
		case "windows":
			assert.Assert(t, fs.CaseSensitivity().IsCaseInsensitive())
		case "linux":
			assert.Assert(t, fs.CaseSensitivity().IsCaseSensitive())
		}
	})
}

func TestRootedLinkTargets(t *testing.T) {
	t.Parallel()
	if runtime.GOOS != "windows" {
		t.Skip("links to rooted targets are only refused on Windows")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	assert.NilError(t, os.Mkdir(target, 0o777))
	assert.NilError(t, os.WriteFile(filepath.Join(target, "a.ts"), []byte("a"), 0o666))

	// A root-relative target (`\...`) names a directory on this drive, so
	// unguarded accesses would succeed.
	err := os.Symlink(target[len(filepath.VolumeName(target)):], filepath.Join(dir, "rooted"))
	if err != nil && strings.Contains(err.Error(), "A required privilege is not held by the client") {
		t.Skip("symlink support is not enabled without elevation or developer mode")
	}
	assert.NilError(t, err)
	assert.NilError(t, os.Symlink("rooted", filepath.Join(dir, "chained")))
	assert.NilError(t, os.Symlink("target", filepath.Join(dir, "relative")))
	assert.NilError(t, exec.Command("cmd", "/c", "mklink", "/J", filepath.Join(dir, "junction"), target).Run())

	fs := osvfs.FS()
	root := tspath.RootedDirectoryPathFromAbsolute(dir)
	entries := fs.GetAccessibleEntries(root)
	assert.DeepEqual(t, entries.Directories, []string{"junction", "relative", "target"})

	for _, name := range []string{"junction", "relative", "target"} {
		file := root.ResolveDirectory(name).ResolveFile("a.ts")
		assert.Assert(t, fs.FileExists(file), name)
		_, ok := fs.ReadFile(file)
		assert.Assert(t, ok, name)
	}
	for _, name := range []string{"rooted", "chained"} {
		dir := root.ResolveDirectory(name)
		file := dir.ResolveFile("a.ts")
		assert.Assert(t, !fs.DirectoryExists(dir), name)
		assert.Assert(t, !fs.FileExists(file), name)
		_, ok := fs.ReadFile(file)
		assert.Assert(t, !ok, name)
		assert.Assert(t, fs.Stat(file.AsPath()) == nil, name)
		assert.Equal(t, fs.Realpath(file.AsPath()), file.AsPath(), name)
		assert.Assert(t, fs.WriteFile(dir.ResolveFile("b.ts"), "b") != nil, name)
	}
	_, err = os.Stat(filepath.Join(target, "b.ts"))
	assert.Assert(t, os.IsNotExist(err))
}
