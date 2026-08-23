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

func TestNestedRelationMaybeStack(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		depth        int
		incompatible bool
		diagnostics  int
	}{
		{name: "shallow compatible", depth: 4},
		{name: "shallow incompatible", depth: 4, incompatible: true, diagnostics: 1},
		{name: "threshold compatible", depth: 9},
		{name: "threshold incompatible", depth: 9, incompatible: true, diagnostics: 1},
		{name: "deep compatible", depth: 64},
		{name: "deep incompatible", depth: 64, incompatible: true, diagnostics: 1},
		{name: "overflow compatible", depth: 128},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			content := nestedRelationSource(test.depth, test.incompatible)
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
			assert.Equal(t, len(errors), 0)

			program := compiler.NewProgram(compiler.ProgramOptions{
				Config: parsed,
				Host:   host,
			})
			file := program.GetSourceFile("/foo.ts")
			diagnostics := program.GetSemanticDiagnostics(t.Context(), file)
			assert.Equal(t, len(diagnostics), test.diagnostics)
		})
	}
}

func nestedRelationSource(depth int, incompatible bool) string {
	var source strings.Builder
	for i := range depth {
		fmt.Fprintf(&source, "interface Source%d { value: string;", i)
		if i+1 < depth {
			fmt.Fprintf(&source, " next: Source%d; }\n", i+1)
		} else {
			source.WriteString(" }\n")
		}

		fmt.Fprintf(&source, "interface Target%d { value: string;", i)
		if i+1 < depth {
			fmt.Fprintf(&source, " next: Target%d; }\n", i+1)
		} else {
			if incompatible {
				source.WriteString(" incompatible: number;")
			}
			source.WriteString(" }\n")
		}
	}
	source.WriteString("declare const source: Source0;\n")
	source.WriteString("let target: Target0;\n")
	source.WriteString("target = source;\n")
	source.WriteString("interface CyclicSource { next: CyclicSource; }\n")
	source.WriteString("interface CyclicTarget { next: CyclicTarget; }\n")
	source.WriteString("declare const cyclicSource: CyclicSource;\n")
	source.WriteString("let cyclicTarget: CyclicTarget;\n")
	source.WriteString("cyclicTarget = cyclicSource;\n")
	return source.String()
}
