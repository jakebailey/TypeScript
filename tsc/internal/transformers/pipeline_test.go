package transformers_test

import (
	"slices"
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
	want := []string{"enter B", "enter A", "A one", "B one", "A two", "B two", "exit A", "exit B"}
	if !slices.Equal(events, want) {
		t.Fatalf("events = %v, want %v", events, want)
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
