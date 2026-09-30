package checker

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/parser"
)

func compareNodesReference(c *Checker, n1, n2 *ast.Node) int {
	if n1 == n2 {
		return 0
	}
	if n1 == nil {
		return 1
	}
	if n2 == nil {
		return -1
	}
	s1 := ast.GetSourceFileOfNode(n1)
	s2 := ast.GetSourceFileOfNode(n2)
	if s1 != s2 {
		return c.fileIndexMap[s1] - c.fileIndexMap[s2]
	}
	return n1.Pos() - n2.Pos()
}

func TestCompareNodes(t *testing.T) {
	t.Parallel()

	c := &Checker{fileIndexMap: make(map[*ast.SourceFile]int)}
	nodes := []*ast.Node{nil}
	var collect ast.Visitor
	collect = func(node *ast.Node) bool {
		nodes = append(nodes, node)
		node.ForEachChild(collect)
		return false
	}
	for i, fileName := range []string{"/a.ts", "/b.ts"} {
		file := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: fileName}, `
namespace N {
    interface A { x: { y: string }; z: number }
    type B = A | { x: string };
    function f() { if (true) { return { x: 1 }; } return {}; }
}
type C = { x: string } | { y: number };
`, core.ScriptKindTS)
		// File order deliberately differs from collection order.
		c.fileIndexMap[file] = 2 - i
		collect(file.AsNode())
	}
	orphan := &ast.Node{Kind: ast.KindIdentifier, Loc: core.NewTextRange(20, 30)}
	nodes = append(nodes, orphan,
		&ast.Node{Kind: ast.KindIdentifier, Loc: core.NewTextRange(10, 15), Parent: orphan},
		&ast.Node{Kind: ast.KindIdentifier, Loc: core.NewTextRange(10, 15)},
	)
	for i, n1 := range nodes {
		for j, n2 := range nodes {
			if got, want := c.compareNodes(n1, n2), compareNodesReference(c, n1, n2); got != want {
				t.Fatalf("nodes %d and %d: got %d, want %d", i, j, got, want)
			}
		}
	}
}

func compareNodesSiblings(c *Checker, n1, n2 *ast.Node) int {
	if n1 == n2 {
		return 0
	}
	if n1 == nil {
		return 1
	}
	if n2 == nil {
		return -1
	}
	if n1.Parent != nil && n1.Parent == n2.Parent {
		return n1.Pos() - n2.Pos()
	}
	s1 := ast.GetSourceFileOfNode(n1)
	s2 := ast.GetSourceFileOfNode(n2)
	if s1 != s2 {
		return c.fileIndexMap[s1] - c.fileIndexMap[s2]
	}
	return n1.Pos() - n2.Pos()
}

func BenchmarkCompareNodes(b *testing.B) {
	file1 := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/a.ts"}, "", core.ScriptKindTS)
	file2 := parser.ParseSourceFile(ast.SourceFileParseOptions{FileName: "/b.ts"}, "", core.ScriptKindTS)
	c := &Checker{fileIndexMap: map[*ast.SourceFile]int{file1: 0, file2: 1}}
	chain := func(parent *ast.Node, depth, pos int) *ast.Node {
		for range depth {
			parent = &ast.Node{
				Kind:   ast.KindIdentifier,
				Loc:    core.NewTextRange(pos, pos+1),
				Parent: parent,
			}
		}
		return parent
	}
	shared := chain(file1.AsNode(), 30, 0)
	deep1 := chain(shared, 2, 10)
	deep2 := chain(shared, 2, 20)
	for _, test := range []struct {
		name   string
		n1, n2 *ast.Node
	}{
		{"identical", deep1, deep1},
		{"nil", nil, deep1},
		{"siblings", chain(shared, 1, 10), chain(shared, 1, 20)},
		{"grandparent", deep1, deep2},
		{"distantAncestor", chain(shared, 8, 10), chain(shared, 8, 20)},
		{"unequalDepth", deep1, chain(shared, 5, 20)},
		{"ancestor", shared, deep1},
		{"differentFilesShallow", chain(file1.AsNode(), 1, 10), chain(file2.AsNode(), 1, 20)},
		{"differentFilesDeep", deep1, chain(file2.AsNode(), 32, 20)},
		{"differentFilesUnequalDepth", deep1, chain(file2.AsNode(), 2, 20)},
		{"sourceFiles", file1.AsNode(), file2.AsNode()},
		{"orphans", chain(nil, 2, 10), chain(nil, 5, 20)},
	} {
		b.Run(test.name, func(b *testing.B) {
			for _, variant := range []struct {
				name    string
				compare func(*Checker, *ast.Node, *ast.Node) int
			}{
				{"reference", compareNodesReference},
				{"siblings", compareNodesSiblings},
				{"sharedAncestor", (*Checker).compareNodes},
			} {
				b.Run(variant.name, func(b *testing.B) {
					if got, want := variant.compare(c, test.n1, test.n2), compareNodesReference(c, test.n1, test.n2); got != want {
						b.Fatalf("got %d, want %d", got, want)
					}
					b.ReportAllocs()
					for b.Loop() {
						variant.compare(c, test.n1, test.n2)
					}
				})
			}
		})
	}
}
