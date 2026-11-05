package checker

import (
	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// When possible, symbolLinkStore stores Symbol links in an efficient and densely
// packed paged array store. When a Symbol has not yet been assigned an ID, the store provides one
// from a generator. The generator grabs chunks of 256 IDs from a central atomic counter, ensuring that each
// block of 256 IDs is consecutive and causing Symbol links to be densely packed within a single page.
// IDs from a generator are in a reserved range of the ID space.

const maxPageLinkCount = 0x100_0000 // 16M

// symbolNodeLinkStore holds the symbol a node resolves to, if any.
type symbolNodeLinkStore struct {
	m map[ast.Node]*ast.Symbol
}

func (s *symbolNodeLinkStore) get(node ast.Node) *ast.Symbol {
	return s.m[node]
}

func (s *symbolNodeLinkStore) set(node ast.Node, symbol *ast.Symbol) {
	if s.m == nil {
		s.m = make(map[ast.Node]*ast.Symbol)
	}
	s.m[node] = symbol
}

type symbolLinkStore[V any] struct {
	gen   ast.SymbolIdGenerator
	links core.LinkStore[uint64, V]
	pages core.PagedLinkStore[V]
}

func (s *symbolLinkStore[V]) Get(symbol *ast.Symbol) *V {
	id := uint64(s.gen.GetSymbolId(symbol))
	if id >= ast.BlockIdOffset && id < ast.BlockIdOffset+maxPageLinkCount {
		return s.pages.Get(id - ast.BlockIdOffset)
	}
	return s.links.Get(id)
}

func (s *symbolLinkStore[V]) Has(symbol *ast.Symbol) bool {
	return s.TryGet(symbol) != nil
}

func (s *symbolLinkStore[V]) TryGet(symbol *ast.Symbol) *V {
	id := uint64(s.gen.GetSymbolId(symbol))
	if id >= ast.BlockIdOffset && id < ast.BlockIdOffset+maxPageLinkCount {
		return s.pages.TryGet(id - ast.BlockIdOffset)
	}
	return s.links.TryGet(id)
}
