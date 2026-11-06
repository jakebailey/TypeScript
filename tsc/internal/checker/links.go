package checker

import (
	"math/bits"
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

// linkValues holds link records in chunks that double in size, the first holding 16, and names each by its index.
// Records never move, so a pointer to one stays valid. A map from a key to an index holds no pointers, so neither
// the map nor, when the record holds none, the chunks are scanned by the garbage collector.
type linkValues[V any] struct {
	chunks [][]V
	n      uint32
}

func (s *linkValues[V]) add() uint32 {
	if s.n == 16<<len(s.chunks)-16 {
		s.chunks = append(s.chunks, make([]V, 16<<len(s.chunks)))
	}
	s.n++
	return s.n - 1
}

func (s *linkValues[V]) at(i uint32) *V {
	k := bits.Len32((i+16)>>4) - 1
	return &s.chunks[k][i+16-16<<k]
}

// mapLinkStore holds links that few keys have, in a map from the key to the index of its record.
type mapLinkStore[K comparable, V any] struct {
	entries map[K]uint32
	values  linkValues[V]
}

func (s *mapLinkStore[K, V]) Get(key K) *V {
	if i, ok := s.entries[key]; ok {
		return s.values.at(i)
	}
	if s.entries == nil {
		s.entries = make(map[K]uint32)
	}
	i := s.values.add()
	s.entries[key] = i
	return s.values.at(i)
}

func (s *mapLinkStore[K, V]) Has(key K) bool {
	_, ok := s.entries[key]
	return ok
}

func (s *mapLinkStore[K, V]) TryGet(key K) *V {
	if i, ok := s.entries[key]; ok {
		return s.values.at(i)
	}
	return nil
}

// nodeLinkStore holds links of nodes, in a map per file from node index to
// the index of the node's record. Files are numbered by the checker's
// FileTable, so a key is 4 bytes rather than a 16-byte Node.
type nodeLinkStore[V any] struct {
	refs   *ast.RefCache
	files  []map[uint32]uint32
	values linkValues[V]
}

func (s *nodeLinkStore[V]) fileLinks(r ast.NodeRef) map[uint32]uint32 {
	if int(r.File()) < len(s.files) {
		return s.files[r.File()]
	}
	return nil
}

func (s *nodeLinkStore[V]) Get(node ast.Node) *V {
	r := s.refs.Ref(node)
	m := s.fileLinks(r)
	if i, ok := m[r.Index()]; ok {
		return s.values.at(i)
	}
	if m == nil {
		if int(r.File()) >= len(s.files) {
			s.files = slices.Grow(s.files, int(r.File())+1-len(s.files))[:r.File()+1]
		}
		m = make(map[uint32]uint32)
		s.files[r.File()] = m
	}
	i := s.values.add()
	m[r.Index()] = i
	return s.values.at(i)
}

func (s *nodeLinkStore[V]) Has(node ast.Node) bool {
	r := s.refs.Ref(node)
	_, ok := s.fileLinks(r)[r.Index()]
	return ok
}

func (s *nodeLinkStore[V]) TryGet(node ast.Node) *V {
	r := s.refs.Ref(node)
	if i, ok := s.fileLinks(r)[r.Index()]; ok {
		return s.values.at(i)
	}
	return nil
}

// nodeValueMap holds one small value per node, in a map per file keyed by
// node index. The value lives in the map itself rather than in a record the
// map points to, so a link of one field costs one map slot, and a map whose
// values hold no pointers is not scanned by the garbage collector.
type nodeValueMap[V any] struct {
	refs  *ast.RefCache
	files []map[uint32]V
}

func (s *nodeValueMap[V]) get(node ast.Node) V {
	r := s.refs.Ref(node)
	if int(r.File()) < len(s.files) {
		return s.files[r.File()][r.Index()]
	}
	var zero V
	return zero
}

func (s *nodeValueMap[V]) set(node ast.Node, value V) {
	r := s.refs.Ref(node)
	if int(r.File()) >= len(s.files) {
		s.files = slices.Grow(s.files, int(r.File())+1-len(s.files))[:r.File()+1]
	}
	m := s.files[r.File()]
	if m == nil {
		m = make(map[uint32]V)
		s.files[r.File()] = m
	}
	m[r.Index()] = value
}

func (s *nodeValueMap[V]) delete(node ast.Node) {
	r := s.refs.Ref(node)
	if int(r.File()) < len(s.files) {
		delete(s.files[r.File()], r.Index())
	}
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
	files []map[uint32]ast.SymbolId
}

func (s *symbolNodeLinkStore) get(node ast.Node) ast.SymbolId {
	r := s.refs.Ref(node)
	if int(r.File()) < len(s.files) {
		if id := s.files[r.File()][r.Index()]; id != 0 {
			return id
		}
	}
	return 0
}

func (s *symbolNodeLinkStore) set(node ast.Node, symbol ast.SymbolId) {
	r := s.refs.Ref(node)
	if int(r.File()) >= len(s.files) {
		s.files = slices.Grow(s.files, int(r.File())+1-len(s.files))[:r.File()+1]
	}
	m := s.files[r.File()]
	if m == nil {
		m = make(map[uint32]ast.SymbolId)
		s.files[r.File()] = m
	}
	m[r.Index()] = symbol
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
	links mapLinkStore[ast.SymbolId, V]
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
	links mapLinkStore[ast.SymbolId, V]
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
