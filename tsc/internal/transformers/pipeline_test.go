package transformers_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/printer"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/emittestutil"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/parsetestutil"
	"github.com/microsoft/TypeScript/tsc/internal/transformers"
)

func TestPipelineLocality(t *testing.T) {
	t.Parallel()
	file := parsetestutil.ParseTypeScript("one; two;", false)
	context := printer.NewEmitContext()
	var events []string
	makeTransform := func(name string) *transformers.Transformer {
		tx := &transformers.Transformer{}
		return tx.NewTransformer(func(node *ast.Node) *ast.Node {
			if ast.IsSourceFile(node) {
				events = append(events, "enter "+name)
				result := tx.Visitor().VisitEachChild(node)
				events = append(events, "exit "+name)
				return result
			}
			if ast.IsIdentifier(node) {
				events = append(events, name+" "+node.Text())
			}
			return tx.Visitor().VisitEachChild(node)
		}, context)
	}
	transformers.Pipeline(file, []*transformers.Transformer{makeTransform("A"), makeTransform("B")})
	want := []string{"enter B", "enter A", "A one", "A two", "B one", "B two", "exit A", "exit B"}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
	}
}

func TestPipelineBoundedWorkingSet(t *testing.T) {
	t.Parallel()
	file := parsetestutil.ParseTypeScript(strings.Repeat("element;\n", 100), false)
	context := printer.NewEmitContext()
	seen := [2]int{}
	makeTransform := func(phase int) *transformers.Transformer {
		tx := &transformers.Transformer{}
		return tx.NewTransformer(func(node *ast.Node) *ast.Node {
			if ast.IsExpressionStatement(node) {
				seen[phase]++
				if seen[0]-seen[1] > 16 {
					t.Fatal("an earlier phase ran ahead of the bounded working set")
				}
			}
			return tx.Visitor().VisitEachChild(node)
		}, context)
	}
	transformers.Pipeline(file, []*transformers.Transformer{makeTransform(0), makeTransform(1)})
	if seen != [2]int{100, 100} {
		t.Fatalf("visits = %v, want [100 100]", seen)
	}
}

func TestPipelineCaptureOwnership(t *testing.T) {
	t.Parallel()
	for _, nested := range []bool{false, true} {
		t.Run(fmt.Sprintf("nested=%v", nested), func(t *testing.T) {
			t.Parallel()
			var source strings.Builder
			for i := range 10 {
				fmt.Fprintf(&source, "keep%d; drop%d; expand%d; capture%d;\n", i, i, i, i)
			}
			file := parsetestutil.ParseTypeScript(source.String(), false)
			context := printer.NewEmitContext()
			first := &transformers.Transformer{}
			var captured []*ast.Node
			callbacks := 0
			first.AfterSourceElement = func(root *ast.Node, output []*ast.Node) []*ast.Node {
				callbacks++
				name := root.Expression().Text()
				var want []string
				switch {
				case strings.HasPrefix(name, "expand"):
					want = []string{name + "_a_done", name + "_b_done"}
				case !strings.HasPrefix(name, "drop"):
					want = []string{name + "_done"}
				}
				var actual []string
				for _, node := range output {
					actual = append(actual, node.Expression().Text())
				}
				if !slices.Equal(actual, want) {
					t.Fatalf("%s output = %v, want %v", name, actual, want)
				}
				if strings.HasPrefix(name, "keep") {
					return output
				}
				captured = append(captured, output...)
				return nil
			}
			first.NewTransformer(func(node *ast.Node) *ast.Node {
				if ast.IsSourceFile(node) {
					visited := first.Visitor().VisitEachChild(node).AsSourceFile()
					statements := append(slices.Clone(visited.Statements.Nodes),
						first.Factory().NewBlock(first.Factory().NewNodeList(captured), true))
					return first.Factory().UpdateSourceFile(visited, first.Factory().NewNodeList(statements), visited.EndOfFileToken)
				}
				return node
			}, context)
			second := &transformers.Transformer{}
			if nested {
				second.AfterSourceElement = func(_ *ast.Node, output []*ast.Node) []*ast.Node {
					return output
				}
			}
			second.NewTransformer(func(node *ast.Node) *ast.Node {
				if ast.IsExpressionStatement(node) {
					name := node.Expression().Text()
					if strings.HasPrefix(name, "drop") {
						return nil
					}
					if strings.HasPrefix(name, "expand") {
						return second.Factory().NewSyntaxList([]*ast.Node{
							second.Factory().NewExpressionStatement(second.Factory().NewIdentifier(name + "_a")),
							second.Factory().NewExpressionStatement(second.Factory().NewIdentifier(name + "_b")),
						})
					}
				}
				return second.Visitor().VisitEachChild(node)
			}, context)
			last := &transformers.Transformer{}
			visits := 0
			last.NewTransformer(func(node *ast.Node) *ast.Node {
				if ast.IsIdentifier(node) {
					visits++
					return last.Factory().NewIdentifier(node.Text() + "_done")
				}
				return last.Visitor().VisitEachChild(node)
			}, context)
			result := transformers.Pipeline(file, []*transformers.Transformer{first, second, last})
			var want strings.Builder
			for i := range 10 {
				fmt.Fprintf(&want, "keep%d_done;\n", i)
			}
			want.WriteString("{\n")
			for i := range 10 {
				fmt.Fprintf(&want, "    expand%d_a_done;\n    expand%d_b_done;\n    capture%d_done;\n", i, i, i)
			}
			want.WriteString("}")
			emittestutil.CheckEmit(t, context, result, want.String())
			if callbacks != 40 || visits != 40 {
				t.Fatalf("callbacks = %d, visits = %d, want 40 each", callbacks, visits)
			}
		})
	}
}

func TestPipelineEnvironments(t *testing.T) {
	t.Parallel()
	file := parsetestutil.ParseTypeScript("one; two;", false)
	context := printer.NewEmitContext()
	makeTransform := func(name string) *transformers.Transformer {
		tx := &transformers.Transformer{}
		helper := &printer.EmitHelper{Name: name}
		return tx.NewTransformer(func(node *ast.Node) *ast.Node {
			if ast.IsSourceFile(node) {
				result := tx.Visitor().VisitEachChild(node)
				helpers := context.ReadEmitHelpers()
				if !slices.Equal(helpers, []*printer.EmitHelper{helper}) {
					t.Fatalf("%s consumed another transform's helpers: %v", name, helpers)
				}
				context.AddEmitHelper(result, helpers...)
				return result
			}
			if ast.IsExpressionStatement(node) {
				context.AddVariableDeclaration(tx.Factory().NewIdentifier(name + node.Expression().Text()))
				context.RequestEmitHelper(helper)
			}
			return node
		}, context)
	}
	result := transformers.Pipeline(file, []*transformers.Transformer{makeTransform("a"), makeTransform("b")})
	p := printer.NewPrinter(printer.PrinterOptions{NoEmitHelpers: true, NewLine: core.NewLineKindLF}, printer.PrintHandlers{}, context)
	want := "var bone, btwo;\nvar aone, atwo;\none;\ntwo;\n"
	if actual := p.EmitSourceFile(result); actual != want {
		t.Fatalf("emit = %q, want %q", actual, want)
	}
	helpers := context.GetEmitHelpers(result.AsNode())
	if len(helpers) != 2 || helpers[0].Name != "a" || helpers[1].Name != "b" {
		t.Fatalf("file helpers = %v", helpers)
	}
}

func TestPipelineEpilogue(t *testing.T) {
	t.Parallel()
	file := parsetestutil.ParseTypeScript("original;", false)
	context := printer.NewEmitContext()
	first := &transformers.Transformer{}
	first.NewTransformer(func(node *ast.Node) *ast.Node {
		if ast.IsSourceFile(node) {
			visited := first.Visitor().VisitEachChild(node).AsSourceFile()
			nodes := append(slices.Clone(visited.Statements.Nodes), first.Factory().NewExpressionStatement(first.Factory().NewIdentifier("generated")))
			return first.Factory().UpdateSourceFile(visited, first.Factory().NewNodeList(nodes), visited.EndOfFileToken)
		}
		return node
	}, context)
	second := &transformers.Transformer{}
	var seen []string
	second.NewTransformer(func(node *ast.Node) *ast.Node {
		if ast.IsIdentifier(node) {
			seen = append(seen, node.Text())
			return second.Factory().NewIdentifier(node.Text() + "_visited")
		}
		return second.Visitor().VisitEachChild(node)
	}, context)
	result := transformers.Pipeline(file, []*transformers.Transformer{first, second})
	emittestutil.CheckEmit(t, context, result, "original_visited;\ngenerated_visited;")
	if !slices.Equal(seen, []string{"original", "generated"}) {
		t.Fatalf("visits = %v", seen)
	}
}

func TestPipelineWrappedEpilogue(t *testing.T) {
	t.Parallel()
	file := parsetestutil.ParseTypeScript("one; two;", false)
	context := printer.NewEmitContext()
	first := &transformers.Transformer{}
	var body []*ast.Node
	first.AfterSourceElement = func(_ *ast.Node, output []*ast.Node) []*ast.Node {
		body = append(body, output...)
		return nil
	}
	first.NewTransformer(func(node *ast.Node) *ast.Node {
		if ast.IsSourceFile(node) {
			visited := first.Visitor().VisitEachChild(node).AsSourceFile()
			nodes := []*ast.Node{first.Factory().NewBlock(first.Factory().NewNodeList(append(body,
				first.Factory().NewExpressionStatement(first.Factory().NewIdentifier("generated")))), true)}
			return first.Factory().UpdateSourceFile(visited, first.Factory().NewNodeList(nodes), visited.EndOfFileToken)
		}
		return node
	}, context)
	second := &transformers.Transformer{}
	var seen []string
	second.NewTransformer(func(node *ast.Node) *ast.Node {
		if ast.IsIdentifier(node) {
			seen = append(seen, node.Text())
			return second.Factory().NewIdentifier(node.Text() + "_visited")
		}
		return second.Visitor().VisitEachChild(node)
	}, context)
	result := transformers.Pipeline(file, []*transformers.Transformer{first, second})
	emittestutil.CheckEmit(t, context, result, "{\n    one_visited;\n    two_visited;\n    generated_visited;\n}")
	if !slices.Equal(seen, []string{"one", "two", "generated"}) {
		t.Fatalf("visits = %v", seen)
	}
}
