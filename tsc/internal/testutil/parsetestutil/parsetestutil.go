package parsetestutil

import (
	"strings"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/diagnosticwriter"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
	"github.com/microsoft/TypeScript/tsc/internal/tspath"
)

// Simplifies parsing an input string into a SourceFile for testing purposes.
func ParseTypeScript(text string, jsx bool) *ast.SourceFile {
	fileName := tspath.RootedFilePathFromNormalized(core.IfElse(jsx, "/main.tsx", "/main.ts"))
	file := parser.ParseSourceFile(ast.SourceFileParseOptions{
		FileName: fileName,
		PathKey:  tspath.CaseSensitive.PathKey(tspath.RootedPath(fileName)),
	}, text, core.GetScriptKindFromFileName(fileName))
	return file
}

// Asserts that the given file has no parse diagnostics.
func CheckDiagnostics(t *testing.T, file *ast.SourceFile) {
	t.Helper()
	if len(file.Diagnostics()) > 0 {
		var b strings.Builder
		diagnosticwriter.WriteFormatDiagnostics(&b, diagnosticwriter.FromASTDiagnostics(file.Diagnostics()), &diagnosticwriter.FormattingOptions{
			NewLine: "\n",
		})
		t.Error(b.String())
	}
}

// Asserts that the given file has no parse diagnostics and asserts the given message.
func CheckDiagnosticsMessage(t *testing.T, file *ast.SourceFile, message string) {
	t.Helper()
	if len(file.Diagnostics()) > 0 {
		var b strings.Builder
		diagnosticwriter.WriteFormatDiagnostics(&b, diagnosticwriter.FromASTDiagnostics(file.Diagnostics()), &diagnosticwriter.FormattingOptions{
			NewLine: "\n",
		})
		t.Error(message + b.String())
	}
}

func newSyntheticRecursiveVisitor(store *ast.NodeStore) *ast.NodeVisitor {
	var v *ast.NodeVisitor
	v = ast.NewNodeVisitor(
		func(node ast.Node) ast.Node {
			return v.VisitEachChild(node)
		},
		ast.NewNodeFactory(store, ast.NodeFactoryHooks{}),
		ast.NodeVisitorHooks{
			VisitNode: func(node ast.Node, v *ast.NodeVisitor) ast.Node {
				if !node.IsNil() {
					node.SetLoc(core.UndefinedTextRange())
				}
				return v.VisitNode(node)
			},
			VisitToken: func(node ast.Node, v *ast.NodeVisitor) ast.Node {
				if !node.IsNil() {
					node.SetLoc(core.UndefinedTextRange())
				}
				return v.VisitNode(node)
			},
			VisitNodes: func(nodes ast.NodeList, v *ast.NodeVisitor) ast.NodeList {
				if !nodes.IsNil() {
					nodes.SetLoc(core.UndefinedTextRange())
				}
				return v.VisitNodes(nodes)
			},
			VisitModifiers: func(nodes ast.ModifierList, v *ast.NodeVisitor) ast.ModifierList {
				if !nodes.IsNil() {
					nodes.SetLoc(core.UndefinedTextRange())
				}
				return v.VisitModifiers(nodes)
			},
		},
	)
	return v
}

// Sets the Loc of the given node and every Node in its subtree to an undefined TextRange (-1,-1).
func MarkSyntheticRecursive(node ast.Node) {
	// Visiting makes updated nodes that are unused.
	var store ast.NodeStore
	defer store.Release()
	newSyntheticRecursiveVisitor(&store).VisitNode(node)
}

// NewNodeStore makes a node store that is released when the test finishes.
func NewNodeStore(tb testing.TB) *ast.NodeStore {
	store := ast.NewNodeStore()
	tb.Cleanup(store.Release)
	return store
}

// NewNodeFactory makes a node factory whose nodes stay valid until the test
// finishes.
func NewNodeFactory(tb testing.TB) *ast.NodeFactory {
	return ast.NewNodeFactory(NewNodeStore(tb), ast.NodeFactoryHooks{})
}
