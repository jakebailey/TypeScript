package checker

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// When possible, symbolLinkStore stores Symbol links in an efficient and densely
// packed paged array store. When a Symbol has not yet been assigned an ID, the store provides one
// from a generator. The generator grabs chunks of 256 IDs from a central atomic counter, ensuring that each
// block of 256 IDs is consecutive and causing Symbol links to be densely packed within a single page.
// IDs from a generator are in a reserved range of the ID space.

const maxPageLinkCount = 0x100_0000 // 16M

// initNodeLinks makes the checker's node link stores number files with refs.
func (c *Checker) initNodeLinks(refs ast.RefCache) {
	c.refs = refs
	c.nodeLinks.refs = &c.refs
	c.resolvedSignatures.refs = &c.refs
	c.effectsSignatures.refs = &c.refs
	c.decoratorSignatures.refs = &c.refs
	c.symbolNodeLinks.refs = &c.refs
	c.resolvedTypes.refs = &c.refs
	c.outerTypeParameters.refs = &c.refs
	c.sourceFileLinks.refs = &c.refs
	c.enumMemberLinks.refs = &c.refs
	c.assertionLinks.refs = &c.refs
	c.arrayLiteralLinks.refs = &c.refs
	c.switchStatementLinks.refs = &c.refs
	c.jsxElementLinks.refs = &c.refs
	c.computedNameLinks.refs = &c.refs
}

// nodeLinkStore holds links of nodes, keyed by NodeRef, in records from an
// arena that the map points to. Callers keep a record across calls that
// insert more links, so a record can't live in the map, whose entries move
// when it grows.
type nodeLinkStore[V any] struct {
	refs  *ast.RefCache
	links core.IdMap[uint64, *V]
	arena core.Arena[V]
}

func (s *nodeLinkStore[V]) Get(node ast.Node) *V {
	key := s.refs.Ref(node).Key()
	if value := s.links.Get(key); value != nil {
		return value
	}
	value := s.arena.New()
	s.links.Set(key, value)
	return value
}

func (s *nodeLinkStore[V]) Has(node ast.Node) bool {
	return s.links.Has(s.refs.Ref(node).Key())
}

func (s *nodeLinkStore[V]) TryGet(node ast.Node) *V {
	return s.links.Get(s.refs.Ref(node).Key())
}

// nodeValueMap holds one small value per node, keyed by NodeRef, in the
// map itself.
type nodeValueMap[V any] struct {
	refs   *ast.RefCache
	values core.IdMap[uint64, V]
}

func (s *nodeValueMap[V]) get(node ast.Node) V {
	return s.values.Get(s.refs.Ref(node).Key())
}

func (s *nodeValueMap[V]) set(node ast.Node, value V) {
	s.values.Set(s.refs.Ref(node).Key(), value)
}

func (s *nodeValueMap[V]) delete(node ast.Node) {
	s.values.Delete(s.refs.Ref(node).Key())
}

// sourceFileLinkStore holds the links of source files, indexed by the
// checker's number for the file.
type sourceFileLinkStore struct {
	refs  *ast.RefCache
	files []*SourceFileLinks
	arena core.Arena[SourceFileLinks]
}

func (s *sourceFileLinkStore) Get(file *ast.SourceFile) *SourceFileLinks {
	id := s.refs.Ref(file.AsNode()).File()
	if int(id) >= len(s.files) {
		s.files = slices.Grow(s.files, int(id)+1-len(s.files))[:id+1]
	}
	links := s.files[id]
	if links == nil {
		links = s.arena.New()
		s.files[id] = links
	}
	return links
}

// symbolValueMap holds one small value per symbol, keyed by SymbolId, for
// links that few symbols have.
type symbolValueMap[V any] struct {
	entries map[ast.SymbolId]V
}

func (s *symbolValueMap[V]) get(symbol ast.SymbolId) V {
	return s.entries[symbol]
}

func (s *symbolValueMap[V]) set(symbol ast.SymbolId, value V) {
	if s.entries == nil {
		s.entries = make(map[ast.SymbolId]V)
	}
	s.entries[symbol] = value
}

// symbolNodeLinkStore holds the symbol a node resolves to, if any, in a
// map per file keyed by node index.
type symbolNodeLinkStore struct {
	refs  *ast.RefCache
	files []core.IdMap[uint32, ast.SymbolId]
}

func (s *symbolNodeLinkStore) get(node ast.Node) ast.SymbolId {
	r := s.refs.Ref(node)
	if int(r.File()) < len(s.files) {
		return s.files[r.File()].Get(r.Index())
	}
	return 0
}

func (s *symbolNodeLinkStore) set(node ast.Node, symbol ast.SymbolId) {
	r := s.refs.Ref(node)
	if int(r.File()) >= len(s.files) {
		s.files = slices.Grow(s.files, int(r.File())+1-len(s.files))[:r.File()+1]
	}
	s.files[r.File()].Set(r.Index(), symbol)
}

// symbolIds numbers symbols for a checker, caching the base id of the store
// it numbered last.
type symbolIds struct {
	store *ast.SymbolStore
	last  *ast.SymbolStore
	base  uint32
}

func (s *symbolIds) id(symbol ast.Symbol) ast.SymbolId {
	st := symbol.Store()
	if st == s.last {
		return ast.SymbolId(s.base + symbol.Index())
	}
	if st == nil {
		return 0
	}
	return s.idSlow(symbol)
}

func (s *symbolIds) idSlow(symbol ast.Symbol) ast.SymbolId {
	st := symbol.Store()
	if base, ok := s.store.BaseOf(st); ok {
		s.last = st
		s.base = base
		return ast.SymbolId(base + symbol.Index())
	}
	return s.store.IdOf(symbol)
}

func (s *symbolIds) symbol(id ast.SymbolId) ast.Symbol {
	return s.store.SymbolOf(id)
}

// symbolOf returns the symbol id names among the checker's symbols.
func (c *Checker) symbolOf(id ast.SymbolId) ast.Symbol {
	return c.symbolIds.symbol(id)
}

// symbolId returns the id of symbol among the checker's symbols.
func (c *Checker) symbolId(symbol ast.Symbol) ast.SymbolId {
	return c.symbolIds.id(symbol)
}

type symbolLinkStore[V any] struct {
	links core.LinkStore[ast.SymbolId, V]
	pages core.PagedLinkStore[V]
}

func (s *symbolLinkStore[V]) Get(symbol ast.SymbolId) *V {
	if symbol < maxPageLinkCount {
		return s.pages.Get(uint64(symbol))
	}
	return s.links.Get(symbol)
}

func (s *symbolLinkStore[V]) Has(symbol ast.SymbolId) bool {
	return s.TryGet(symbol) != nil
}

func (s *symbolLinkStore[V]) TryGet(symbol ast.SymbolId) *V {
	if symbol < maxPageLinkCount {
		return s.pages.TryGet(uint64(symbol))
	}
	return s.links.TryGet(symbol)
}

// symbolMapLinks holds links that few symbols have, keyed by SymbolId.
type symbolMapLinks[V any] struct {
	links core.LinkStore[ast.SymbolId, V]
}

func (s *symbolMapLinks[V]) Get(symbol ast.SymbolId) *V {
	return s.links.Get(symbol)
}

func (s *symbolMapLinks[V]) Has(symbol ast.SymbolId) bool {
	return s.links.Has(symbol)
}

func (s *symbolMapLinks[V]) TryGet(symbol ast.SymbolId) *V {
	return s.links.TryGet(symbol)
}

// nodeOf resolves a ref held by a checker record.
func (c *Checker) nodeOf(r ast.NodeRef) ast.Node {
	return c.refs.Table().Node(r)
}

// refOf makes a ref for a checker record to hold.
func (c *Checker) refOf(n ast.Node) ast.NodeRef {
	return c.refs.Ref(n)
}
