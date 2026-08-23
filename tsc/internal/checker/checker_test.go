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

func BenchmarkTemplateLiteralUnionReduction(b *testing.B) {
	const count = 500
	var source strings.Builder
	source.WriteString("type T =\n")
	for i := range count {
		fmt.Fprintf(&source, "    | \"common-prefix-%03d-value-common-suffix-%03d\"\n", i, i)
	}
	for i := range count {
		fmt.Fprintf(&source, "    | `common-prefix-%03d-${string}-common-suffix-%03d`\n", i, i)
	}

	fs := vfstest.FromMap(map[string]string{"/index.ts": source.String()}, true)
	options := core.CompilerOptions{NoLib: core.TSTrue}
	programOptions := compiler.ProgramOptions{
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &tsoptions.ParsedOptions{
				FileNames:       []string{"/index.ts"},
				CompilerOptions: &options,
			},
		},
		Host: compiler.NewCompilerHost("/", fs, bundled.LibPath(), nil, nil, nil),
	}

	b.ResetTimer()
	for b.Loop() {
		program := compiler.NewProgram(programOptions)
		program.BindSourceFiles()
		checker, done := program.GetTypeChecker(b.Context())
		checker.GetDiagnostics(b.Context(), program.GetSourceFile("/index.ts"))
		done()
	}
}
