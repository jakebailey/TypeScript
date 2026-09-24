package project

import (
	"bytes"
	"context"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/ls/lsconv"
	"github.com/microsoft/TypeScript/tsc/internal/lsp/lsproto"
	"github.com/microsoft/TypeScript/tsc/internal/project/logging"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
	"gotest.tools/v3/assert"
)

type nativeWatchTestFS struct {
	vfs.FS
	comparer fswatch.PathComparer
	err      error
	calls    atomic.Int64
}

func (fs *nativeWatchTestFS) WatchPathComparer(string) (fswatch.PathComparer, error) {
	fs.calls.Add(1)
	return fs.comparer, fs.err
}

func TestResolutionLookupGlobsPreserveOriginalNames(t *testing.T) {
	t.Parallel()
	const workspace, current, lib = "/Workspace/K", "/Current/K", "/Library/K"
	for _, tc := range []struct {
		name, file, glob, outside string
	}{
		{name: "workspace", file: workspace + "/main.ts", glob: workspace + "/**/*"},
		{name: "current", file: current + "/main.ts", glob: current + "/**/*"},
		{name: "library", file: lib + "/lib.d.ts", glob: lib + "/**/*"},
		{name: "node modules", file: "/External/K/NODE_MODULES/Pkg/main.ts", glob: "/External/K/NODE_MODULES/**/*"},
		{name: "external", file: "/External/K/Project/main.ts", outside: "/External/K/Project"},
		{name: "NFD", file: "/External/e\u0301/Project/main.ts", outside: "/External/e\u0301/Project"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			source := newSourceFS(true, nil, func(name string) tspath.Path {
				return tspath.ToPath(name, current, false)
			})
			source.Track(tc.file)
			result := createResolutionLookupGlobMapper(workspace, lib, current, false)(source.seenFiles)
			if tc.glob != "" {
				assert.DeepEqual(t, result.patternsInsideWorkspace, []string{tc.glob})
				assert.Equal(t, len(result.directoriesOutsideWorkspace), 0)
			} else {
				assert.DeepEqual(t, result.directoriesOutsideWorkspace, []string{tc.outside})
				assert.Equal(t, len(result.patternsInsideWorkspace), 0)
			}
		})
	}
}

func TestNativeWatchIndexIsLazy(t *testing.T) {
	t.Parallel()
	for _, enabled := range []bool{false, true} {
		t.Run(strconv.FormatBool(enabled), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			const main = `import { value } from "./value";`
			fs := &nativeWatchTestFS{FS: vfstest.FromMap(map[string]string{
				"/src/tsconfig.json": `{"compilerOptions":{"noLib":true,"types":[]},"files":["main.ts"]}`,
				"/src/main.ts":       main,
				"/src/value.ts":      "export const value = 1;",
			}, true)}
			session := NewSession(&SessionInit{
				BackgroundCtx: ctx, FS: fs, Client: &noopClient{},
				Options: &SessionOptions{CurrentDirectory: "/src", WatchEnabled: enabled},
			})
			defer session.Close()
			session.DidOpenFile(ctx, "file:///src/main.ts", 1, main, lsproto.LanguageKindTypeScript)
			session.DidChangeFile(ctx, "file:///src/main.ts", 2, []lsproto.TextDocumentContentChangePartialOrWholeDocument{{
				WholeDocument: &lsproto.TextDocumentContentChangeWholeDocument{Text: main + "\n"},
			}})
			_, err := session.GetLanguageService(ctx, "file:///src/main.ts")
			assert.NilError(t, err)
			session.WaitForBackgroundTasks()
			assert.Equal(t, fs.calls.Load(), int64(0), "editor-only activity must not build a native index")
			old := session.Snapshot()
			old.ref()
			defer old.Deref()
			assert.NilError(t, fs.WriteFile("/src/value.ts", "export const value = 2;"))
			events := []*lsproto.FileEvent{{Uri: "file:///src/value.ts", Type: lsproto.FileChangeTypeChanged}}
			session.DidChangeWatchedFiles(ctx, events)
			calls := fs.calls.Load()
			assert.Equal(t, calls > 0, enabled)
			session.DidChangeWatchedFiles(ctx, events)
			assert.Equal(t, fs.calls.Load(), calls, "reuse the index within a snapshot")
			service, err := session.GetLanguageService(ctx, "file:///src/main.ts")
			assert.NilError(t, err)
			assert.Equal(t, service.GetProgram().GetSourceFile("/src/value.ts").Text(), "export const value = 2;")
			assert.Equal(t, old.GetDefaultProject("file:///src/main.ts").Program.GetSourceFile("/src/value.ts").Text(), "export const value = 1;")
			assert.Equal(t, fs.calls.Load(), calls, "cloning must not eagerly build the next index")
		})
	}
}

func TestNativeWatchComparerFailureInvalidatesState(t *testing.T) {
	t.Parallel()
	for _, failure := range []error{syscall.EIO, syscall.EACCES} {
		t.Run(failure.Error(), func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			const main = `import { value } from "./value";`
			fs := &nativeWatchTestFS{err: failure, FS: vfstest.FromMap(map[string]string{
				"/src/tsconfig.json": `{"extends":"./middle.json","files":["main.ts"]}`,
				"/src/middle.json":   `{"extends":"./base.json"}`,
				"/src/base.json":     `{"compilerOptions":{"noLib":true,"types":[],"strict":false}}`,
				"/src/main.ts":       main,
				"/src/value.ts":      "export const value = 1;",
			}, true)}
			var logs bytes.Buffer
			session := NewSession(&SessionInit{
				BackgroundCtx: ctx, FS: fs, Client: &noopClient{}, Logger: logging.NewLogger(&logs),
				Options: &SessionOptions{CurrentDirectory: "/src", WatchEnabled: true},
			})
			defer session.Close()
			session.DidOpenFile(ctx, "file:///src/main.ts", 1, main, lsproto.LanguageKindTypeScript)
			session.WaitForBackgroundTasks()
			assert.Equal(t, fs.calls.Load(), int64(0))
			assert.NilError(t, fs.WriteFile("/src/value.ts", "export const value = 2;"))
			assert.NilError(t, fs.WriteFile("/src/main.ts", main+"\n"))
			assert.NilError(t, fs.WriteFile("/src/base.json", `{"compilerOptions":{"noLib":true,"types":[],"strict":true}}`))
			session.DidChangeWatchedFiles(ctx, []*lsproto.FileEvent{{
				Uri: "file:///unrecognized/alias.data", Type: lsproto.FileChangeTypeChanged,
			}})
			session.WaitForBackgroundTasks()
			service, err := session.GetLanguageService(ctx, "file:///src/main.ts")
			assert.NilError(t, err)
			assert.Equal(t, service.GetProgram().GetSourceFile("/src/value.ts").Text(), "export const value = 2;")
			assert.Equal(t, service.GetProgram().Options().Strict, core.TSTrue)
			overlay := session.Snapshot().overlays()[session.toPath("/src/main.ts")]
			assert.Equal(t, overlay.Content(), main)
			assert.Assert(t, !overlay.MatchesDiskText())
			session.WaitForBackgroundTasks()
			assert.Assert(t, strings.Contains(logs.String(), failure.Error()))
			assert.Assert(t, strings.Contains(logs.String(), "invalidating"))
			calls := fs.calls.Load()
			fs.err = nil
			session.DidChangeWatchedFiles(ctx, []*lsproto.FileEvent{{
				Uri: "file:///src/value.ts", Type: lsproto.FileChangeTypeChanged,
			}})
			_, err = session.GetLanguageService(ctx, "file:///src/main.ts")
			assert.NilError(t, err)
			assert.Assert(t, fs.calls.Load() > calls, "later snapshots must retry the comparer")
		})
	}
}

func TestNativeWatchCoalescingAndOverlays(t *testing.T) {
	t.Parallel()
	comparer, err := fswatch.PathComparerForPath(t.TempDir())
	assert.NilError(t, err)
	if comparer.Key("/src/é") != comparer.Key("/src/e\u0301") {
		t.Skip("native Unicode comparison unavailable")
	}
	ctx := context.Background()
	const main = `import { value } from "./é/value";`
	const value = "export const value = 1;"
	const valueName = "/src/e\u0301/value.ts"
	fs := &nativeWatchTestFS{comparer: comparer, FS: vfstest.FromMap(map[string]string{
		"/src/tsconfig.json": `{"compilerOptions":{"noLib":true,"types":[]},"files":["main.ts"]}`,
		"/src/main.ts":       main,
		valueName:            value,
	}, true)}
	session := NewSession(&SessionInit{
		BackgroundCtx: ctx, FS: fs, Client: &noopClient{},
		Options: &SessionOptions{CurrentDirectory: "/src", WatchEnabled: true},
	})
	defer session.Close()
	session.DidOpenFile(ctx, "file:///src/main.ts", 1, main, lsproto.LanguageKindTypeScript)
	session.WaitForBackgroundTasks()
	session.DidChangeWatchedFiles(ctx, []*lsproto.FileEvent{
		{Uri: lsconv.FileNameToDocumentURI("/src/é"), Type: lsproto.FileChangeTypeDeleted},
		{Uri: lsconv.FileNameToDocumentURI("/src/e\u0301"), Type: lsproto.FileChangeTypeCreated},
	})
	service, err := session.GetLanguageService(ctx, "file:///src/main.ts")
	assert.NilError(t, err)
	source := service.GetProgram().GetSourceFile(valueName)
	assert.Assert(t, source != nil, "directory recreation must not delete cached descendants")
	assert.Equal(t, source.Text(), value)
	valueURI := lsconv.FileNameToDocumentURI(valueName)
	session.DidOpenFile(ctx, valueURI, 1, value, lsproto.LanguageKindTypeScript)
	assert.NilError(t, fs.WriteFile(valueName, "export const value = 2;"))
	session.DidChangeWatchedFiles(ctx, []*lsproto.FileEvent{{
		Uri: lsconv.FileNameToDocumentURI("/src/é/value.ts"), Type: lsproto.FileChangeTypeChanged,
	}})
	service, err = session.GetLanguageService(ctx, valueURI)
	assert.NilError(t, err)
	assert.Equal(t, service.GetProgram().GetSourceFile(valueName).Text(), value)
	overlay := session.Snapshot().overlays()[session.toPath(valueName)]
	assert.Equal(t, overlay.Content(), value)
	assert.Equal(t, overlay.Version(), int32(1))
	assert.Assert(t, !overlay.MatchesDiskText(), "native events must update overlay disk state")
}

func TestWatchMockFilesystemDoesNotAcquireNativeSemantics(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	const main = `import "./s"; import "./ſ";`
	disk := vfstest.FromMap(map[string]string{
		"/src/tsconfig.json": `{"compilerOptions":{"noLib":true,"types":[]},"files":["main.ts"]}`,
		"/src/main.ts":       main,
		"/src/s.ts":          "export const value = 1;",
		"/src/ſ.ts":          "export const value = 2;",
	}, true)
	session := NewSession(&SessionInit{
		BackgroundCtx: ctx, FS: disk, Client: &noopClient{},
		Options: &SessionOptions{CurrentDirectory: "/src", WatchEnabled: true},
	})
	defer session.Close()
	session.DidOpenFile(ctx, "file:///src/main.ts", 1, main, lsproto.LanguageKindTypeScript)
	assert.NilError(t, disk.WriteFile("/src/s.ts", "export const value = 3;"))
	session.DidChangeWatchedFiles(ctx, []*lsproto.FileEvent{{Uri: "file:///src/s.ts", Type: lsproto.FileChangeTypeChanged}})
	service, err := session.GetLanguageService(ctx, "file:///src/main.ts")
	assert.NilError(t, err)
	assert.Equal(t, service.GetProgram().GetSourceFile("/src/s.ts").Text(), "export const value = 3;")
	assert.Equal(t, service.GetProgram().GetSourceFile("/src/ſ.ts").Text(), "export const value = 2;")
	index, err := session.Snapshot().getNativeWatchIndex()
	assert.NilError(t, err)
	assert.Assert(t, index == nil)
}
