package api

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/projecttestutil"
	"gotest.tools/v3/assert"
)

func TestAdoptSourceFiles(t *testing.T) {
	t.Parallel()
	projectSession, _ := projecttestutil.Setup(map[string]any{})
	defer projectSession.Close()
	session := NewLSPSession(projectSession, nil)
	defer session.Close()
	ctx := context.Background()
	create := func(id uint64, name, text string) *ast.SourceFile {
		t.Helper()
		_, err := session.handleCreateSourceFile(ctx, &CreateSourceFileParams{
			SourceFileID: id, FileName: name, SourceTextBase64: base64.StdEncoding.EncodeToString([]byte(text)),
		})
		assert.NilError(t, err)
		return session.sourceFiles[id]
	}
	a := create(1, "/virtual/a.ts", `import { b } from "./b"; export const a: number = b;`)
	b := create(2, "/virtual/b.ts", `export const b = 1;`)
	params := &CreateProgramParams{
		RootFiles:            []DocumentIdentifier{{FileName: a.FileName()}},
		CreateProgramOptions: CreateProgramOptions{CompilerOptions: core.CompilerOptions{NoLib: core.TSTrue}},
		SourceFiles:          []uint64{1, 2},
	}
	check := func(response *CreateProgramResponse, expectedA *ast.SourceFile) {
		t.Helper()
		sd, err := session.getSnapshotData(response.Snapshot)
		assert.NilError(t, err)
		program, err := sd.getProgram(response.Project.Id)
		assert.NilError(t, err)
		assert.Equal(t, program.GetSourceFile(a.FileName()), expectedA)
		assert.Equal(t, program.GetSourceFile(b.FileName()), b)
		assert.Assert(t, expectedA.IsBound())
		diagnostics, err := session.handleGetSemanticDiagnostics(ctx, &GetDiagnosticsParams{
			Snapshot: response.Snapshot, Project: response.Project.Id,
		})
		assert.NilError(t, err)
		assert.Equal(t, len(diagnostics), 0)
	}
	first, err := session.handleCreateProgram(ctx, params)
	assert.NilError(t, err)
	check(first, a)
	second, err := session.handleCreateProgram(ctx, params)
	assert.NilError(t, err)
	check(second, a)
	_, err = session.handleCreateSourceFile(ctx, &CreateSourceFileParams{
		SourceFileID: 1, FileName: a.FileName(), SourceTextBase64: "",
	})
	assert.ErrorContains(t, err, "duplicate source file handle")
	params.OldProgram = &CreateProgramOldProgramParams{Snapshot: first.Snapshot, Project: first.Project.Id}
	params.SourceFiles = nil
	params.FileChanges = &APIFileChanges{InvalidateAll: true}
	rebuilt, err := session.handleCreateProgram(ctx, params)
	assert.NilError(t, err)
	assert.NilError(t, session.releaseSnapshot(first.Snapshot))
	check(rebuilt, a)

	replacement := create(3, a.FileName(), `import { b } from "./b"; export const a: number = b + 1;`)
	params.SourceFiles = []uint64{3}
	params.OldProgram = &CreateProgramOldProgramParams{Snapshot: rebuilt.Snapshot, Project: rebuilt.Project.Id}
	updated, err := session.handleCreateProgram(ctx, params)
	assert.NilError(t, err)
	check(updated, replacement)
	check(rebuilt, a)

	_, err = session.handleCreateProgram(ctx, &CreateProgramParams{
		RootFiles:   params.RootFiles,
		SourceFiles: params.SourceFiles,
		OldProgram:  params.OldProgram,
		CreateProgramOptions: CreateProgramOptions{CompilerOptions: core.CompilerOptions{
			NoLib:           core.TSTrue,
			ModuleDetection: core.ModuleDetectionKindForce,
		}},
	})
	assert.ErrorContains(t, err, "incompatible parse options")
	params.SourceFiles = []uint64{999}
	_, err = session.handleCreateProgram(ctx, params)
	assert.ErrorContains(t, err, "unknown source file")
	params.SourceFiles = []uint64{1, 3}
	_, err = session.handleCreateProgram(ctx, params)
	assert.ErrorContains(t, err, "duplicate source file")

	session.Close()
	assert.Equal(t, len(session.sourceFiles), 0)
	_, err = session.getSourceFilesForProgram([]uint64{1})
	assert.ErrorContains(t, err, "unknown source file")
}
