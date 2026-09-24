package project

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/fswatch"
	"github.com/microsoft/TypeScript/tsc/internal/ls/lsconv"
	"github.com/microsoft/TypeScript/tsc/internal/lsp/lsproto"
	"github.com/microsoft/TypeScript/tsc/internal/vfs"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/osvfs"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
	"github.com/microsoft/TypeScript/tsc/internal/watchalias"
	"gotest.tools/v3/assert"
)

type nativeWatchBenchmarkFS struct {
	vfs.FS
	native    bool
	queries   atomic.Int64
	realpaths atomic.Int64
}

func (f *nativeWatchBenchmarkFS) WatchPathComparisonEnabled() bool {
	return f.native
}

func (f *nativeWatchBenchmarkFS) WatchPathComparer(directory string) (fswatch.PathComparer, error) {
	f.queries.Add(1)
	return osvfs.FS().(watchalias.ComparerProvider).WatchPathComparer(directory)
}

func (f *nativeWatchBenchmarkFS) Realpath(name string) string {
	f.realpaths.Add(1)
	return f.FS.Realpath(name)
}

// Contents are virtual, but comparer queries use real repository ancestors.
// Run comparison binaries from the same directory so pathname costs agree.
func BenchmarkSessionNativeWatch(b *testing.B) {
	cwd, err := os.Getwd()
	assert.NilError(b, err)
	root := filepath.ToSlash(cwd) + "/watch-bench"
	ctx := context.Background()
	for _, size := range []int{1000, 10000} {
		for _, native := range []bool{false, true} {
			for _, disk := range []bool{false, true} {
				b.Run(fmt.Sprintf("files=%d/native=%v/disk=%v", size, native, disk), func(b *testing.B) {
					if native && !fswatch.NativePathComparisonAvailable {
						b.Skip("native path comparison unavailable")
					}
					files := map[string]string{
						root + "/tsconfig.json": `{"compilerOptions":{"noLib":true,"types":[]}}`,
						root + "/main.ts":       "export const value = 1;",
					}
					for i := range size {
						files[fmt.Sprintf("%s/group%d/file%d.ts", root, i/100, i)] = "export const value = 1;"
					}
					fs := &nativeWatchBenchmarkFS{FS: vfstest.FromMap(files, true), native: native}
					session := NewSession(&SessionInit{
						BackgroundCtx: ctx, FS: fs, Client: &noopClient{},
						Options: &SessionOptions{CurrentDirectory: root, WatchEnabled: true},
					})
					defer session.Close()
					uri := lsconv.FileNameToDocumentURI(root + "/main.ts")
					session.DidOpenFile(ctx, uri, 1, files[uri.FileName()], lsproto.LanguageKindTypeScript)
					session.WaitForBackgroundTasks()
					name := uri.FileName()
					if disk {
						name = root + "/group0/file0.ts"
					}
					events := []*lsproto.FileEvent{{Uri: lsconv.FileNameToDocumentURI(name), Type: lsproto.FileChangeTypeChanged}}
					version := int32(1)
					fs.queries.Store(0)
					fs.realpaths.Store(0)
					b.ReportAllocs()
					b.ResetTimer()
					for b.Loop() {
						version++
						text := fmt.Sprintf("export const value = %d;", version)
						if disk {
							b.StopTimer()
							assert.NilError(b, fs.WriteFile(name, text))
							b.StartTimer()
							session.DidChangeWatchedFiles(ctx, events)
						} else {
							session.DidChangeFile(ctx, uri, version, []lsproto.TextDocumentContentChangePartialOrWholeDocument{{
								WholeDocument: &lsproto.TextDocumentContentChangeWholeDocument{Text: text},
							}})
						}
						service, serviceErr := session.GetLanguageService(ctx, uri)
						assert.NilError(b, serviceErr)
						assert.Equal(b, len(service.GetProgram().GetSourceFiles()), size+1)
						assert.Equal(b, service.GetProgram().GetSourceFile(name).Text(), text)
					}
					b.StopTimer()
					session.WaitForBackgroundTasks()
					b.ReportMetric(float64(fs.queries.Load())/float64(b.N), "comparer-queries/op")
					b.ReportMetric(float64(fs.realpaths.Load())/float64(b.N), "realpath/op")
				})
			}
		}
	}
}
