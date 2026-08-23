package checker

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
)

func NewIntersectionTypeForTest(c *Checker, types []*Type) *Type {
	return c.newIntersectionType(ObjectFlagsNone, types)
}

func IteratePropertyNamesForTest(c *Checker, t *Type, limit int) []string {
	var names []string
	for prop := range c.iteratePropertiesOfUnionOrIntersectionType(t, false /*skipYield*/) {
		names = append(names, prop.Name)
		if len(names) == limit {
			break
		}
	}
	return names
}

func PropertyResolutionStateForTest(t *Type) (resolved bool, partialCount int) {
	d := t.AsUnionOrIntersectionType()
	return d.resolvedProperties != nil, len(d.partiallyResolvedProperties)
}

func GetReducedTypeForTest(c *Checker, t *Type) *Type {
	return c.getReducedType(t)
}

func GetUnionOrIntersectionPropertiesForTest(c *Checker, t *Type) []*ast.Symbol {
	return c.getPropertiesOfUnionOrIntersectionType(t)
}

func PropertyNamesForTest(properties []*ast.Symbol) []string {
	return slices.Collect(func(yield func(string) bool) {
		for _, prop := range properties {
			if !yield(prop.Name) {
				return
			}
		}
	})
}
