package ast_test

import (
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/testutil/parsetestutil"
)

func TestGeneratedSyntaxSubtreeFacts(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		source string
		facts  ast.SubtreeFacts
	}{
		{"using resource = value;", 0},
		{"await using resource = value;", ast.SubtreeContainsAnyAwait},
		{"class C { constructor(value: number) {} }", 0},
		{"class C { constructor(public value: number) {} }", ast.SubtreeContainsClassFields},
	} {
		t.Run(test.source, func(t *testing.T) {
			t.Parallel()
			file := parsetestutil.ParseTypeScript(test.source, false)
			actual := file.SubtreeFacts() & (ast.SubtreeContainsAnyAwait | ast.SubtreeContainsClassFields)
			if actual != test.facts {
				t.Fatalf("facts = %v, want %v", actual, test.facts)
			}
		})
	}
}
