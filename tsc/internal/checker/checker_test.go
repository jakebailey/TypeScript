package checker_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/checker"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
	"gotest.tools/v3/assert"
)

func setupChecker(t testing.TB, content string) (*compiler.Program, *checker.Checker) {
	t.Helper()
	fs := vfstest.FromMap(map[string]string{
		"/foo.ts": content,
		"/tsconfig.json": `{
			"compilerOptions": { "strict": true },
			"files": ["foo.ts"]
		}`,
	}, false /*useCaseSensitiveFileNames*/)
	fs = bundled.WrapFS(fs)
	host := compiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil, nil)
	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, host, nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")
	p := compiler.NewProgram(compiler.ProgramOptions{Config: parsed, Host: host})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	t.Cleanup(done)
	return p, c
}

func TestGetSymbolAtLocation(t *testing.T) {
	t.Parallel()

	content := `interface Foo {
  bar: string;
}
declare const foo: Foo;
foo.bar;`
	p, c := setupChecker(t, content)
	file := p.GetSourceFile("/foo.ts")
	interfaceId := file.Statements.Nodes[0].Name()
	varId := file.Statements.Nodes[1].AsVariableStatement().DeclarationList.AsVariableDeclarationList().Declarations.Nodes[0].Name()
	propAccess := file.Statements.Nodes[2].Expression()
	nodes := []*ast.Node{interfaceId, varId, propAccess}
	for _, node := range nodes {
		symbol := c.GetSymbolAtLocation(node)
		if symbol == nil {
			t.Fatalf("Expected symbol to be non-nil")
		}
	}
}

func TestLazyPropertiesOfUnionOrIntersection(t *testing.T) {
	t.Parallel()

	const content = `
	type I = {
	    first: "a";
	    shared: number;
	    onlyA: string;
	} & {
	    first: "b";
	    shared: number;
	    onlyB: boolean;
	};
	type U = {
	    kind: "a";
	    shared: number;
	    onlyA: string;
	} | {
	    kind: "b";
	    shared: number;
	    onlyB: boolean;
	};
	type IndexedU = {
	    [key: string]: number;
	    shared: number;
	} | {
	    shared: number;
	    fromIndex: number;
	};
	type EmptyU = {
	    onlyA: string;
	} | {
	    onlyB: boolean;
	};`
	p, c := setupChecker(t, content)
	file := p.GetSourceFile("/foo.ts")
	intersection := c.GetTypeFromTypeNode(file.Statements.Nodes[0].Type())
	union := c.GetTypeFromTypeNode(file.Statements.Nodes[1].Type())
	indexedUnion := c.GetTypeFromTypeNode(file.Statements.Nodes[2].Type())
	emptyUnion := c.GetTypeFromTypeNode(file.Statements.Nodes[3].Type())

	freshIntersection := checker.NewIntersectionTypeForTest(c, intersection.Types())
	assert.DeepEqual(t, checker.IteratePropertyNamesForTest(c, freshIntersection, 1), []string{"first"})
	resolved, partialCount := checker.PropertyResolutionStateForTest(freshIntersection)
	assert.Assert(t, !resolved)
	assert.Equal(t, partialCount, 1)

	assert.DeepEqual(t, checker.IteratePropertyNamesForTest(c, freshIntersection, 2), []string{"first", "shared"})
	resolved, partialCount = checker.PropertyResolutionStateForTest(freshIntersection)
	assert.Assert(t, !resolved)
	assert.Equal(t, partialCount, 2)

	fullAfterPartial := checker.PropertyNamesForTest(checker.GetUnionOrIntersectionPropertiesForTest(c, freshIntersection))
	assert.DeepEqual(t, fullAfterPartial, []string{"first", "shared", "onlyA", "onlyB"})
	resolved, partialCount = checker.PropertyResolutionStateForTest(freshIntersection)
	assert.Assert(t, resolved)
	assert.Equal(t, partialCount, 0)
	assert.DeepEqual(t, checker.IteratePropertyNamesForTest(c, freshIntersection, len(fullAfterPartial)), fullAfterPartial)

	freshFull := checker.NewIntersectionTypeForTest(c, intersection.Types())
	assert.DeepEqual(t, checker.PropertyNamesForTest(checker.GetUnionOrIntersectionPropertiesForTest(c, freshFull)), fullAfterPartial)

	freshReduced := checker.NewIntersectionTypeForTest(c, intersection.Types())
	assert.Assert(t, checker.GetReducedTypeForTest(c, freshReduced).Flags()&checker.TypeFlagsNever != 0)
	resolved, partialCount = checker.PropertyResolutionStateForTest(freshReduced)
	assert.Assert(t, !resolved)
	assert.Equal(t, partialCount, 1)

	assert.DeepEqual(t, checker.IteratePropertyNamesForTest(c, union, 1), []string{"kind"})
	resolved, partialCount = checker.PropertyResolutionStateForTest(union)
	assert.Assert(t, !resolved)
	assert.Equal(t, partialCount, 1)
	assert.DeepEqual(t, checker.PropertyNamesForTest(c.GetPropertiesOfType(union)), []string{"kind", "shared"})

	assert.DeepEqual(t, checker.PropertyNamesForTest(c.GetPropertiesOfType(indexedUnion)), []string{"shared", "fromIndex"})

	assert.Equal(t, len(c.GetPropertiesOfType(emptyUnion)), 0)
	resolved, partialCount = checker.PropertyResolutionStateForTest(emptyUnion)
	assert.Assert(t, resolved)
	assert.Equal(t, partialCount, 0)
	assert.Equal(t, len(c.GetPropertiesOfType(emptyUnion)), 0)
}

func BenchmarkLazyPropertiesOfIntersection(b *testing.B) {
	var source strings.Builder
	source.WriteString("type I = {\n    first: \"a\";\n")
	for i := range 500 {
		fmt.Fprintf(&source, "    p%d: number;\n", i)
	}
	source.WriteString("} & {\n    first: \"b\";\n")
	for i := range 500 {
		fmt.Fprintf(&source, "    p%d: number;\n", i)
	}
	source.WriteString("};\n")

	p, c := setupChecker(b, source.String())
	intersection := c.GetTypeFromTypeNode(p.GetSourceFile("/foo.ts").Statements.Nodes[0].Type())
	types := intersection.Types()

	b.Run("EarlyExit", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			t := checker.NewIntersectionTypeForTest(c, types)
			checker.GetReducedTypeForTest(c, t)
		}
	})
	b.Run("FullEnumeration", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			t := checker.NewIntersectionTypeForTest(c, types)
			checker.GetUnionOrIntersectionPropertiesForTest(c, t)
		}
	})
}
