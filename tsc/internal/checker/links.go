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

// symbolNodeLinkStore holds the symbol a node resolves to, if any, in a
// map per store keyed by node index.
type symbolNodeLinkStore struct {
	stores map[*ast.NodeStore]map[uint32]*ast.Symbol
	// last caches the most recently used store's map, since nodes are
	// mostly resolved file by file.
	lastStore *ast.NodeStore
	last      map[uint32]*ast.Symbol
}

func (s *symbolNodeLinkStore) storeLinks(node ast.Node) map[uint32]*ast.Symbol {
	store := node.Store()
	if store != s.lastStore || s.last == nil {
		s.lastStore = store
		s.last = s.stores[store]
	}
	return s.last
}

func (s *symbolNodeLinkStore) get(node ast.Node) *ast.Symbol {
	return s.storeLinks(node)[node.Index()]
}

func (s *symbolNodeLinkStore) set(node ast.Node, symbol *ast.Symbol) {
	m := s.storeLinks(node)
	if m == nil {
		if s.stores == nil {
			s.stores = make(map[*ast.NodeStore]map[uint32]*ast.Symbol)
		}
		m = make(map[uint32]*ast.Symbol)
		s.stores[node.Store()] = m
		s.last = m
	}
	m[node.Index()] = symbol
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
