package checker_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
	"gotest.tools/v3/assert"
)

func makeDisjointLiteralNarrowingSource(smallCount int, largeCount int) string {
	var source strings.Builder
	writeUnion := func(name string, prefix string, count int) {
		fmt.Fprintf(&source, "type %s = ", name)
		for i := range count {
			if i != 0 {
				source.WriteString(" | ")
			}
			fmt.Fprintf(&source, "%q", fmt.Sprintf("%s%d", prefix, i))
		}
		source.WriteString(";\n")
	}
	writeUnion("Small", "small", smallCount)
	writeUnion("Large", "large", largeCount)
	source.WriteString(`
type Value = Small | Large;
declare function isLarge(value: string): value is Large;
function narrow(value: Value) {
    if (isLarge(value)) {
        const large: Large = value;
        return large;
    }
    const small: Small = value;
    return small;
}
`)
	return source.String()
}

func TestGetSymbolAtLocation(t *testing.T) {
	t.Parallel()

	content := `interface Foo {
  bar: string;
}
declare const foo: Foo;
foo.bar;`
	fs := vfstest.FromMap(map[string]string{
		"/foo.ts": content,
		"/tsconfig.json": `
				{
					"compilerOptions": {},
					"files": ["foo.ts"]
				}
			`,
	}, false /*useCaseSensitiveFileNames*/)
	fs = bundled.WrapFS(fs)

	cd := "/"
	host := compiler.NewCompilerHost(cd, fs, bundled.LibPath(), nil, nil, nil)

	parsed, errors := tsoptions.GetParsedCommandLineOfConfigFile("/tsconfig.json", &core.CompilerOptions{}, nil, host, nil)
	assert.Equal(t, len(errors), 0, "Expected no errors in parsed command line")

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: parsed,
		Host:   host,
	})
	p.BindSourceFiles()
	c, done := p.GetTypeChecker(t.Context())
	defer done()
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

func BenchmarkDisjointLiteralUnionNarrowing(b *testing.B) {
	const fileName = "/index.ts"
	fs := vfstest.FromMap(map[string]string{
		fileName: makeDisjointLiteralNarrowingSource(334, 11_346),
	}, true /*useCaseSensitiveFileNames*/)
	options := &core.CompilerOptions{
		NoLib:            core.TSTrue,
		StrictNullChecks: core.TSTrue,
	}
	config := &tsoptions.ParsedCommandLine{
		ParsedConfig: &tsoptions.ParsedOptions{
			FileNames:       []string{fileName},
			CompilerOptions: options,
		},
	}
	host := compiler.NewCompilerHost("/", fs, "", nil, nil, nil)

	b.ReportAllocs()
	for b.Loop() {
		program := compiler.NewProgram(compiler.ProgramOptions{
			Config:         config,
			Host:           host,
			SingleThreaded: core.TSTrue,
		})
		diagnostics := program.GetSemanticDiagnostics(b.Context(), program.GetSourceFile(fileName))
		if len(diagnostics) != 0 {
			b.Fatalf("unexpected diagnostics: %v", diagnostics)
		}
	}
}
