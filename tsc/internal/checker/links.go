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
	c.signatureLinks.refs = &c.refs
	c.symbolNodeLinks.refs = &c.refs
	c.typeNodeLinks.refs = &c.refs
	c.enumMemberLinks.refs = &c.refs
	c.assertionLinks.refs = &c.refs
	c.arrayLiteralLinks.refs = &c.refs
	c.switchStatementLinks.refs = &c.refs
	c.jsxElementLinks.refs = &c.refs
	c.computedNameLinks.refs = &c.refs
}

// nodeLinkStore holds links of nodes, in a map per file keyed by node
// index. Files are numbered by the checker's FileTable, so a key is 4
// bytes rather than a 16-byte Node.
type nodeLinkStore[V any] struct {
	refs  *ast.RefCache
	files []map[uint32]*V
	arena core.Arena[V]
}

func (s *nodeLinkStore[V]) fileLinks(r ast.NodeRef) map[uint32]*V {
	if int(r.File()) < len(s.files) {
		return s.files[r.File()]
	}
	return nil
}

func (s *nodeLinkStore[V]) Get(node ast.Node) *V {
	r := s.refs.Ref(node)
	m := s.fileLinks(r)
	if value := m[r.Index()]; value != nil {
		return value
	}
	if m == nil {
		if int(r.File()) >= len(s.files) {
			s.files = slices.Grow(s.files, int(r.File())+1-len(s.files))[:r.File()+1]
		}
		m = make(map[uint32]*V)
		s.files[r.File()] = m
	}
	value := s.arena.New()
	m[r.Index()] = value
	return value
}

func (s *nodeLinkStore[V]) Has(node ast.Node) bool {
	r := s.refs.Ref(node)
	_, ok := s.fileLinks(r)[r.Index()]
	return ok
}

func (s *nodeLinkStore[V]) TryGet(node ast.Node) *V {
	r := s.refs.Ref(node)
	return s.fileLinks(r)[r.Index()]
}

// symbolNodeLinkStore holds the symbol a node resolves to, if any, in a
// map per file keyed by node index.
type symbolNodeLinkStore struct {
	refs  *ast.RefCache
	files []map[uint32]*ast.Symbol
}

func (s *symbolNodeLinkStore) get(node ast.Node) *ast.Symbol {
	r := s.refs.Ref(node)
	if int(r.File()) < len(s.files) {
		return s.files[r.File()][r.Index()]
	}
	return nil
}

func (s *symbolNodeLinkStore) set(node ast.Node, symbol *ast.Symbol) {
	r := s.refs.Ref(node)
	if int(r.File()) >= len(s.files) {
		s.files = slices.Grow(s.files, int(r.File())+1-len(s.files))[:r.File()+1]
	}
	m := s.files[r.File()]
	if m == nil {
		m = make(map[uint32]*ast.Symbol)
		s.files[r.File()] = m
	}
	m[r.Index()] = symbol
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

// nodeOf resolves a ref held by a checker record.
func (c *Checker) nodeOf(r ast.NodeRef) ast.Node {
	return c.refs.Table().Node(r)
}

// refOf makes a ref for a checker record to hold.
func (c *Checker) refOf(n ast.Node) ast.NodeRef {
	return c.refs.Ref(n)
}
