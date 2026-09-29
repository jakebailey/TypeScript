package compiler_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/bundled"
	"github.com/microsoft/TypeScript/tsc/internal/compiler"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/tsoptions"
	"github.com/microsoft/TypeScript/tsc/internal/vfs/vfstest"
)

// generateLongLineTS generates TypeScript source code that produces a single very long line.
// This simulates generated code (e.g., from code generators) that has no line breaks,
// which triggers O(n²) behavior in source map generation due to
// GetECMALineAndUTF16CharacterOfPosition scanning from line start for each position.
func generateLongLineTS(numProperties int) string {
	// Build a large object literal all on one line, with no line breaks.
	var b strings.Builder
	b.WriteString("export const data: Record<string, number> = {")
	for i := range numProperties {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "prop_%d: %d", i, i)
	}
	b.WriteString("};")
	return b.String()
}

func BenchmarkEmitPipeline(b *testing.B) {
	if !bundled.Embedded {
		b.Skip("bundled files are not embedded")
	}
	for _, test := range []struct {
		name   string
		source string
		module core.ModuleKind
		target core.ScriptTarget
	}{
		{"Plain", `export function f%d(x: number) { return x + 1; }`, core.ModuleKindESNext, core.ScriptTargetESNext},
		{"Syntax", `export function f%d(x: any) { x.value ??= x?.other ** 2; return x.value; }`, core.ModuleKindESNext, core.ScriptTargetES2015},
		{"Classes", `export class C%d { #x = 1; async run(x: any) { const { a, ...rest } = x; return this.#x + (await x?.run?.(rest) ?? a); } }`, core.ModuleKindESNext, core.ScriptTargetES2015},
		{"CommonJS", `export let x%d = 0; export async function f%[1]d(x: any) { x%[1]d ||= await x?.run(); return { ...x, x%[1]d }; }`, core.ModuleKindCommonJS, core.ScriptTargetES2015},
		{"JSX", `export const C%d = (props: any) => <div {...props}>{props.value ?? "default"}</div>;`, core.ModuleKindCommonJS, core.ScriptTargetES2015},
		{"Using", `using resource%d = acquire(); export const x%[1]d = resource%[1]d?.value ?? 0;`, core.ModuleKindCommonJS, core.ScriptTargetES2015},
	} {
		b.Run(test.name, func(b *testing.B) {
			var source strings.Builder
			source.WriteString("declare function acquire(): any;\n")
			for i := range 200 {
				fmt.Fprintf(&source, test.source+"\n", i)
			}
			fs := bundled.WrapFS(vfstest.FromMap(map[string]string{"/src/index.tsx": source.String()}, true))
			opts := core.CompilerOptions{
				Target: test.target, Module: test.module, Jsx: core.JsxEmitReactJSX,
				SourceMap: core.TSTrue, OutDir: "/out",
			}
			p := compiler.NewProgram(compiler.ProgramOptions{
				Config: &tsoptions.ParsedCommandLine{ParsedConfig: &tsoptions.ParsedOptions{
					FileNames: []string{"/src/index.tsx"}, CompilerOptions: &opts,
				}},
				Host: compiler.NewCompilerHost("/src", fs, bundled.LibPath(), nil, nil, nil),
			})
			emit := func() {
				p.Emit(context.Background(), compiler.EmitOptions{
					WriteFile: func(string, string, *compiler.WriteFileData) error { return nil },
				})
			}
			emit()
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				emit()
			}
		})
	}
}

func BenchmarkEmitLongLines(b *testing.B) {
	if !bundled.Embedded {
		b.Skip("bundled files are not embedded")
	}

	for _, numProps := range []int{1000, 5000, 10000} {
		b.Run(fmt.Sprintf("props_%d", numProps), func(b *testing.B) {
			source := generateLongLineTS(numProps)

			fs := vfstest.FromMap(map[string]string{
				"/dev/src/index.ts": source,
			}, true /*useCaseSensitiveFileNames*/)
			fs = bundled.WrapFS(fs)

			opts := core.CompilerOptions{
				Target:    core.ScriptTargetES2015,
				SourceMap: core.TSTrue,
				OutDir:    "/dev/out",
			}

			host := compiler.NewCompilerHost("/dev/src", fs, bundled.LibPath(), nil, nil, nil)

			p := compiler.NewProgram(compiler.ProgramOptions{
				Config: &tsoptions.ParsedCommandLine{
					ParsedConfig: &tsoptions.ParsedOptions{
						FileNames:       []string{"/dev/src/index.ts"},
						CompilerOptions: &opts,
					},
				},
				Host: host,
			})

			// Discard written files — we only care about emit performance.
			nopWriteFile := func(fileName string, text string, data *compiler.WriteFileData) error {
				return nil
			}

			b.ResetTimer()
			b.ReportAllocs()

			for b.Loop() {
				p.Emit(context.Background(), compiler.EmitOptions{
					WriteFile: nopWriteFile,
				})
			}
		})
	}
}

func BenchmarkEmitManyFiles(b *testing.B) {
	if !bundled.Embedded {
		b.Skip("bundled files are not embedded")
	}

	// Simulate many files with moderately long single-line content.
	numFiles := 200
	numPropsPerFile := 500

	files := make(map[string]string, numFiles)
	fileNames := make([]string, 0, numFiles)
	for i := range numFiles {
		name := fmt.Sprintf("/dev/src/file_%d.ts", i)
		files[name] = generateLongLineTS(numPropsPerFile)
		fileNames = append(fileNames, name)
	}

	fs := vfstest.FromMap(files, true)
	fs = bundled.WrapFS(fs)

	opts := core.CompilerOptions{
		Target:    core.ScriptTargetES2015,
		SourceMap: core.TSTrue,
		OutDir:    "/dev/out",
	}

	host := compiler.NewCompilerHost("/dev/src", fs, bundled.LibPath(), nil, nil, nil)

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &tsoptions.ParsedOptions{
				FileNames:       fileNames,
				CompilerOptions: &opts,
			},
		},
		Host: host,
	})

	nopWriteFile := func(fileName string, text string, data *compiler.WriteFileData) error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		p.Emit(context.Background(), compiler.EmitOptions{
			WriteFile: nopWriteFile,
		})
	}
}

// BenchmarkEmitLongLinesWithLineBreaks is a control benchmark that emits the same amount
// of code but WITH line breaks, showing that the issue is specific to long lines.
func BenchmarkEmitLongLinesWithLineBreaks(b *testing.B) {
	if !bundled.Embedded {
		b.Skip("bundled files are not embedded")
	}

	numProperties := 10000

	// Same content but with newlines between each property.
	var sb strings.Builder
	sb.WriteString("export const data: Record<string, number> = {\n")
	for i := range numProperties {
		if i > 0 {
			sb.WriteString(",\n")
		}
		fmt.Fprintf(&sb, "  prop_%d: %d", i, i)
	}
	sb.WriteString("\n};\n")
	source := sb.String()

	fs := vfstest.FromMap(map[string]string{
		"/dev/src/index.ts": source,
	}, true)
	fs = bundled.WrapFS(fs)

	opts := core.CompilerOptions{
		Target:    core.ScriptTargetES2015,
		SourceMap: core.TSTrue,
		OutDir:    "/dev/out",
	}

	host := compiler.NewCompilerHost("/dev/src", fs, bundled.LibPath(), nil, nil, nil)

	p := compiler.NewProgram(compiler.ProgramOptions{
		Config: &tsoptions.ParsedCommandLine{
			ParsedConfig: &tsoptions.ParsedOptions{
				FileNames:       []string{"/dev/src/index.ts"},
				CompilerOptions: &opts,
			},
		},
		Host: host,
	})

	nopWriteFile := func(fileName string, text string, data *compiler.WriteFileData) error {
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		p.Emit(context.Background(), compiler.EmitOptions{
			WriteFile: nopWriteFile,
		})
	}
}
