package ast

import (
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/debug"
)

// A Symbol is a view of a symbol: the store that holds it and its index
// there. Like a Node, it is a value that is passed around and compared; the
// zero Symbol is the absent symbol. The store keeps the symbol's record, and
// the view keeps the store alive.
//
// A file's symbols are made by the binder and kept by the file's node store;
// a checker keeps the symbols it makes in a store of its own. Structures that
// hold many symbols hold a SymbolId instead, which a checker resolves.
type Symbol struct {
	s *SymbolStore
	i uint32
}

func (s Symbol) IsNil() bool { return s.s == nil }

// Store returns the store that holds s.
func (s Symbol) Store() *SymbolStore { return s.s }

// Index returns the index of s in its store.
func (s Symbol) Index() uint32 { return s.i }

// SymbolId names a symbol among the symbols one checker sees: the binder
// symbols of the program's files, file by file, then the checker's own. The
// zero SymbolId is no symbol. Only the checker's store resolves it.
type SymbolId uint32

// SymbolSerial is a process-wide number for a symbol, given the first time
// it is asked for. Unlike a SymbolId it means the same in every checker.
type SymbolSerial uint64

var nextSymbolSerial atomic.Uint64

// A SymbolStore holds symbols: the binder symbols of a file, kept by the
// file's node store, or the symbols a checker makes.
type SymbolStore struct {
	// nodes is the node store whose declarations the binder symbols name by
	// index.
	nodes *NodeStore
	// recs holds the binder symbols; the first is unused, so that index 0
	// is no symbol.
	recs []symbolRecord
	// rare holds the binder symbols' rarely present fields; a record's rare
	// is an index in it plus one.
	rare []symbolRare

	// checker is set in a checker's store.
	checker *checkerSymbols

	serialsMu sync.Mutex
	serials   map[uint32]SymbolSerial
}

// A binder symbol's record. It holds no pointers: its parent and export
// symbol are indices in the same store, and its declarations indices in the
// file's node store, the first inline and any others in its rare fields.
// A binder symbol is 32 bytes.
var _ [0]struct{} = [unsafe.Sizeof(symbolRecord{}) - 32]struct{}{}

type symbolRecord struct {
	flags        SymbolFlags
	value        uint32 // index of the value declaration
	name         SymbolName
	parent       uint32
	exportSymbol uint32
	decl         uint32 // index of the first declaration
	rare         uint32
}

type symbolRare struct {
	// rest holds the declarations after the first, or, when general is set,
	// all declarations and the value declaration, which are then not all in
	// the file's node store.
	rest    *declRest
	general bool
	members SymbolTable
	exports SymbolTable
}

// The symbols a checker makes are kept in fixed-size pages, so that they
// never move and readers in other goroutines need no lock.
const (
	checkerSymbolPageShift = 10
	checkerSymbolPageSize  = 1 << checkerSymbolPageShift
	symbolIdPageShift      = 6
	// lateSymbolIdBase is the first id of the symbols numbered as they are
	// met.
	lateSymbolIdBase = 1 << 31
)

type checkerSymbolPage [checkerSymbolPageSize]checkerSymbolRecord

type checkerSymbolRecord struct {
	flags      SymbolFlags
	checkFlags CheckFlags
	// data holds the symbol's fields, or is nil if the symbol shares the
	// fields of owner.
	data  *symbolData
	owner Symbol
}

type symbolData struct {
	name   SymbolName
	s      *NodeStore
	rest   *declRest
	first  uint32
	value  uint32
	parent SymbolId
	extra  *symbolExtra
}

type symbolExtra struct {
	members      SymbolTable
	exports      SymbolTable
	exportSymbol SymbolId
}

type symbolIdPage struct {
	s    *SymbolStore
	base uint32
}

type checkerSymbols struct {
	pages atomic.Pointer[[]*checkerSymbolPage]
	count uint32
	data  core.Arena[symbolData]
	extra core.Arena[symbolExtra]

	// The program's files' symbols have ids from 1, each file's from a
	// multiple of 1<<symbolIdPageShift, so that dir finds a file's store from
	// an id in one index. The checker's own symbols follow them from
	// binderCount. Symbols of other stores are numbered from
	// lateSymbolIdBase, as they are met.
	dir         []symbolIdPage
	bases       map[*SymbolStore]uint32
	binderCount uint32
	// table and fileBases give the base of a file's store from the number
	// its node store has in table, without a map lookup.
	table     *FileTable
	fileBases []uint32

	mu        sync.Mutex
	latePages atomic.Pointer[[]*lateSymbolPage]
	lateIds   map[Symbol]SymbolId
	lateCount uint32
}

type lateSymbolPage [checkerSymbolPageSize]Symbol

// NewCheckerSymbolStore returns a store for a checker's symbols, whose ids
// number the symbols of the files of t first.
func NewCheckerSymbolStore(t *FileTable) *SymbolStore {
	c := &checkerSymbols{}
	var files []*NodeStore
	if t != nil {
		files = t.files
	}
	c.bases = make(map[*SymbolStore]uint32, len(files))
	c.table = t
	c.fileBases = make([]uint32, len(files))
	next := uint32(1)
	for i, f := range files {
		s := &f.symbols
		if len(s.recs) == 0 {
			continue
		}
		base := (next + 1<<symbolIdPageShift - 1) &^ (1<<symbolIdPageShift - 1)
		c.bases[s] = base
		c.fileBases[i] = base
		for int(base+uint32(len(s.recs))) > len(c.dir)<<symbolIdPageShift {
			c.dir = append(c.dir, symbolIdPage{s, base})
		}
		next = base + uint32(len(s.recs))
	}
	c.binderCount = uint32(len(c.dir)) << symbolIdPageShift
	if c.binderCount == 0 {
		c.binderCount = 1
	}
	return &SymbolStore{checker: c}
}

// NewSymbol returns a new symbol of a checker's store.
func (st *SymbolStore) NewSymbol(flags SymbolFlags, name SymbolName) Symbol {
	r := st.newCheckerRecord()
	r.flags = flags
	r.data = st.checker.data.New()
	r.data.name = name
	return Symbol{st, st.checker.count - 1}
}

// NewSharedDataSymbol returns a new symbol of a checker's store that shares
// the fields other has besides its flags.
func (st *SymbolStore) NewSharedDataSymbol(flags SymbolFlags, other Symbol) Symbol {
	r := st.newCheckerRecord()
	r.flags = flags
	if other.s.checker != nil {
		if o := other.s.checker.rec(other.i); o.data == nil {
			other = o.owner
		} else if other.s == st {
			r.data = o.data
			return Symbol{st, st.checker.count - 1}
		}
	}
	r.owner = other
	return Symbol{st, st.checker.count - 1}
}

func (st *SymbolStore) newCheckerRecord() *checkerSymbolRecord {
	c := st.checker
	i := c.count
	var pages []*checkerSymbolPage
	if p := c.pages.Load(); p != nil {
		pages = *p
	}
	if int(i>>checkerSymbolPageShift) == len(pages) {
		pages = append(slices.Clip(pages), new(checkerSymbolPage))
		c.pages.Store(&pages)
	}
	c.count++
	return &pages[i>>checkerSymbolPageShift][i&(checkerSymbolPageSize-1)]
}

func (c *checkerSymbols) rec(i uint32) *checkerSymbolRecord {
	return &(*c.pages.Load())[i>>checkerSymbolPageShift][i&(checkerSymbolPageSize-1)]
}

// Count returns the number of symbols the store holds.
func (st *SymbolStore) Count() int {
	if st.checker != nil {
		return int(st.checker.count)
	}
	return max(len(st.recs)-1, 0)
}

// BinderSymbolCount returns the first id of a checker store's own symbols.
func (st *SymbolStore) BinderSymbolCount() uint32 {
	return st.checker.binderCount
}

// IdOf returns the id of s in a checker's store.
func (st *SymbolStore) IdOf(s Symbol) SymbolId {
	if s.s == nil {
		return 0
	}
	if base, ok := st.BaseOf(s.s); ok {
		return SymbolId(base + s.i)
	}
	return st.checker.late(s)
}

// BaseOf returns the id of index 0 of a store of the program's files or of
// the checker's store itself, for callers that cache it by store.
func (st *SymbolStore) BaseOf(s *SymbolStore) (uint32, bool) {
	c := st.checker
	if s == st {
		return c.binderCount, true
	}
	if s.nodes != nil {
		if slot := s.nodes.slot.Load(); slot != nil && slot.table == c.table && slot.file < uint32(len(c.fileBases)) {
			if base := c.fileBases[slot.file]; base != 0 {
				return base, true
			}
		}
	}
	base, ok := c.bases[s]
	return base, ok
}

// late numbers a symbol of a store that is neither the checker's nor one of
// the program's files'. Such stores may still grow, so their symbols are
// numbered one by one, as they are met.
func (c *checkerSymbols) late(s Symbol) SymbolId {
	c.mu.Lock()
	defer c.mu.Unlock()
	if id, ok := c.lateIds[s]; ok {
		return id
	}
	if c.lateIds == nil {
		c.lateIds = make(map[Symbol]SymbolId)
	}
	k := c.lateCount
	var pages []*lateSymbolPage
	if p := c.latePages.Load(); p != nil {
		pages = *p
	}
	if int(k>>checkerSymbolPageShift) == len(pages) {
		pages = append(slices.Clip(pages), new(lateSymbolPage))
		c.latePages.Store(&pages)
	}
	pages[k>>checkerSymbolPageShift][k&(checkerSymbolPageSize-1)] = s
	c.lateCount++
	id := SymbolId(lateSymbolIdBase + k)
	c.lateIds[s] = id
	return id
}

// SymbolOf returns the symbol id names in a checker's store.
func (st *SymbolStore) SymbolOf(id SymbolId) Symbol {
	if id == 0 {
		return Symbol{}
	}
	c := st.checker
	i := uint32(id)
	if i >= lateSymbolIdBase {
		k := i - lateSymbolIdBase
		return (*c.latePages.Load())[k>>checkerSymbolPageShift][k&(checkerSymbolPageSize-1)]
	}
	if i >= c.binderCount {
		return Symbol{st, i - c.binderCount}
	}
	p := c.dir[i>>symbolIdPageShift]
	return Symbol{p.s, i - p.base}
}

// Symbols returns the symbol store of the file whose tree s holds, made on
// first use. Only the binder makes symbols in it.
func (s *NodeStore) Symbols() *SymbolStore {
	st := &s.symbols
	if st.nodes == nil {
		st.nodes = s
		st.recs = append(st.recs, symbolRecord{})
	}
	return st
}

// NewSymbol returns a new binder symbol of the file whose symbols st holds.
func (st *SymbolStore) NewBinderSymbol(flags SymbolFlags, name SymbolName) Symbol {
	st.recs = append(st.recs, symbolRecord{flags: flags, name: name})
	return Symbol{st, uint32(len(st.recs) - 1)}
}

// FinishBinding trims the store's slices once the binder is done.
func (st *SymbolStore) finishBinding() {
	if len(st.recs) != cap(st.recs) {
		st.recs = slices.Clip(slices.Clone(st.recs))
	}
	if len(st.rare) != cap(st.rare) {
		st.rare = slices.Clip(slices.Clone(st.rare))
	}
}

func (s Symbol) record() *symbolRecord { return &s.s.recs[s.i] }

func (s Symbol) rareOf() *symbolRare {
	if r := s.s.recs[s.i].rare; r != 0 {
		return &s.s.rare[r-1]
	}
	return nil
}

func (s Symbol) ensureRare() *symbolRare {
	r := &s.s.recs[s.i]
	if r.rare == 0 {
		s.s.rare = append(s.s.rare, symbolRare{})
		r.rare = uint32(len(s.s.rare))
	}
	return &s.s.rare[r.rare-1]
}

// data returns the fields of a symbol of a checker's store, and the symbol
// whose fields they are, which differs from s if s shares another's.
func (s Symbol) data() (*symbolData, Symbol) {
	r := s.s.checker.rec(s.i)
	if r.data == nil {
		if r.owner.s.checker == nil {
			return nil, r.owner
		}
		return r.owner.s.checker.rec(r.owner.i).data, r.owner
	}
	return r.data, s
}

// NewSymbol returns a symbol in a store of its own, for symbols made outside
// a checker.
func NewSymbol() Symbol {
	return NewCheckerSymbolStore(nil).NewSymbol(SymbolFlagsNone, EmptySymbolName)
}

func (s Symbol) Flags() SymbolFlags {
	if c := s.s.checker; c != nil {
		return c.rec(s.i).flags
	}
	return s.s.recs[s.i].flags
}

func (s Symbol) CheckFlags() CheckFlags {
	if c := s.s.checker; c != nil {
		return c.rec(s.i).checkFlags
	}
	return 0
}

func (s Symbol) Name() SymbolName {
	if s.s.checker != nil {
		if d, o := s.data(); d != nil {
			return d.name
		} else {
			return o.Name()
		}
	}
	return s.s.recs[s.i].name
}

func (s Symbol) Parent() Symbol {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			return o.Parent()
		}
		return o.s.SymbolOf(d.parent)
	}
	if p := s.s.recs[s.i].parent; p != 0 {
		return Symbol{s.s, p}
	}
	return Symbol{}
}

func (s Symbol) Declarations() Declarations {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			return o.Declarations()
		}
		return Declarations{s: d.s, rest: d.rest, first: d.first}
	}
	r := s.record()
	if rare := s.rareOf(); rare != nil && rare.rest != nil {
		if rare.general {
			return Declarations{rest: rare.rest}
		}
		return Declarations{s: s.s.nodes, rest: rare.rest, first: r.decl}
	}
	if r.decl == 0 {
		return Declarations{}
	}
	return Declarations{s: s.s.nodes, first: r.decl}
}

func (s Symbol) ValueDeclaration() Node {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			return o.ValueDeclaration()
		}
		if d.s != nil {
			if d.value == 0 {
				return Node{}
			}
			return Node{d.s, d.value}
		}
		if d.rest != nil {
			return d.rest.value
		}
		return Node{}
	}
	if rare := s.rareOf(); rare != nil && rare.general {
		return rare.rest.value
	}
	if v := s.s.recs[s.i].value; v != 0 {
		return Node{s.s.nodes, v}
	}
	return Node{}
}

func (s Symbol) Members() SymbolTable {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			return o.Members()
		}
		if d.extra == nil {
			return nil
		}
		return d.extra.members
	}
	if rare := s.rareOf(); rare != nil {
		return rare.members
	}
	return nil
}

func (s Symbol) Exports() SymbolTable {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			return o.Exports()
		}
		if d.extra == nil {
			return nil
		}
		return d.extra.exports
	}
	if rare := s.rareOf(); rare != nil {
		return rare.exports
	}
	return nil
}

func (s Symbol) ExportSymbol() Symbol {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			return o.ExportSymbol()
		}
		if d.extra == nil {
			return Symbol{}
		}
		return o.s.SymbolOf(d.extra.exportSymbol)
	}
	if e := s.s.recs[s.i].exportSymbol; e != 0 {
		return Symbol{s.s, e}
	}
	return Symbol{}
}

func (s Symbol) SetFlags(value SymbolFlags) {
	if c := s.s.checker; c != nil {
		c.rec(s.i).flags = value
		return
	}
	s.s.recs[s.i].flags = value
}

func (s Symbol) SetCheckFlags(value CheckFlags) {
	if c := s.s.checker; c != nil {
		c.rec(s.i).checkFlags = value
		return
	}
	debug.Assert(value == 0, "A binder symbol has no check flags")
}

func (s Symbol) SetName(value SymbolName) {
	if s.s.checker != nil {
		if d, o := s.data(); d != nil {
			d.name = value
		} else {
			o.SetName(value)
		}
		return
	}
	s.s.recs[s.i].name = value
}

func (s Symbol) SetDeclarations(value Declarations) {
	valueDeclaration := s.ValueDeclaration()
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			o.SetDeclarations(value)
			return
		}
		d.s, d.rest, d.first, d.value = value.s, value.rest, value.first, 0
		o.SetValueDeclaration(valueDeclaration)
		return
	}
	r := s.record()
	switch {
	case value.s == s.s.nodes:
		r.decl = value.first
		if value.rest != nil || r.rare != 0 {
			rare := s.ensureRare()
			rare.rest, rare.general = value.rest, false
		}
	case value.s == nil && value.rest == nil:
		r.decl = 0
		if rare := s.rareOf(); rare != nil {
			rare.rest, rare.general = nil, false
		}
	default:
		r.decl = 0
		rare := s.ensureRare()
		rare.rest, rare.general = &declRest{nodes: value.Nodes()}, true
	}
	r.value = 0
	s.SetValueDeclaration(valueDeclaration)
}

func (s Symbol) SetValueDeclaration(value Node) {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			o.SetValueDeclaration(value)
			return
		}
		switch {
		case d.s == nil && d.rest != nil:
			d.rest = &declRest{nodes: d.rest.nodes, value: value}
		case value.IsNil():
			d.value = 0
		case d.s == nil:
			d.s, d.value = value.s, value.node
		case d.s == value.s:
			d.value = value.node
		default:
			nodes := o.Declarations().Nodes()
			d.s, d.first, d.value = nil, 0, 0
			d.rest = &declRest{nodes: nodes, value: value}
		}
		return
	}
	r := s.record()
	rare := s.rareOf()
	switch {
	case rare != nil && rare.general:
		rare.rest = &declRest{nodes: rare.rest.nodes, value: value}
	case value.IsNil():
		r.value = 0
	case value.s == s.s.nodes:
		r.value = value.node
	default:
		nodes := s.Declarations().Nodes()
		r.decl, r.value = 0, 0
		rare = s.ensureRare()
		rare.rest, rare.general = &declRest{nodes: nodes, value: value}, true
	}
}

func (s Symbol) SetParent(value Symbol) {
	if s.s.checker != nil {
		d, o := s.data()
		if d == nil {
			o.SetParent(value)
			return
		}
		d.parent = o.s.IdOf(value)
		return
	}
	debug.Assert(value.IsNil() || value.s == s.s, "A binder symbol's parent is in its file")
	s.s.recs[s.i].parent = value.i
}

func (s Symbol) extraOf() (*symbolExtra, Symbol) {
	d, o := s.data()
	if d == nil {
		return nil, o
	}
	if d.extra == nil {
		d.extra = o.s.checker.extra.New()
	}
	return d.extra, o
}

func (s Symbol) SetMembers(value SymbolTable) {
	if s.s.checker != nil {
		if value == nil && s.Members() == nil {
			return
		}
		e, o := s.extraOf()
		if e == nil {
			o.SetMembers(value)
			return
		}
		e.members = value
		return
	}
	if value != nil || s.s.recs[s.i].rare != 0 {
		s.ensureRare().members = value
	}
}

func (s Symbol) SetExports(value SymbolTable) {
	if s.s.checker != nil {
		if value == nil && s.Exports() == nil {
			return
		}
		e, o := s.extraOf()
		if e == nil {
			o.SetExports(value)
			return
		}
		e.exports = value
		return
	}
	if value != nil || s.s.recs[s.i].rare != 0 {
		s.ensureRare().exports = value
	}
}

func (s Symbol) SetExportSymbol(value Symbol) {
	if s.s.checker != nil {
		if value.IsNil() && s.ExportSymbol().IsNil() {
			return
		}
		e, o := s.extraOf()
		if e == nil {
			o.SetExportSymbol(value)
			return
		}
		e.exportSymbol = o.s.IdOf(value)
		return
	}
	debug.Assert(value.IsNil() || value.s == s.s, "A binder symbol's export symbol is in its file")
	s.s.recs[s.i].exportSymbol = value.i
}

// GetSymbolSerial returns s's serial, giving it one if it has none.
func GetSymbolSerial(s Symbol) SymbolSerial {
	st := s.s
	st.serialsMu.Lock()
	defer st.serialsMu.Unlock()
	if id, ok := st.serials[s.i]; ok {
		return id
	}
	if st.serials == nil {
		st.serials = make(map[uint32]SymbolSerial)
	}
	id := SymbolSerial(nextSymbolSerial.Add(1))
	st.serials[s.i] = id
	return id
}

// GetSourceFileOfSymbol returns the owning file of a binder symbol, or nil
// for a symbol a checker made, even if it borrows declarations from a file.
func GetSourceFileOfSymbol(symbol Symbol) *SourceFile {
	debug.Assert(!symbol.IsNil(), "Expected a symbol")
	if symbol.s.checker != nil {
		return nil
	}
	file := symbol.s.nodes.file
	debug.Assert(file != nil, "Binder symbol has no source file")
	return file
}

func (s Symbol) IsExternalModule() bool {
	return s.Flags()&SymbolFlagsModule != 0 && IsAmbientModuleSymbolName(s.Name())
}

func (s Symbol) IsStatic() bool {
	if s.ValueDeclaration().IsNil() {
		return false
	}
	modifierFlags := s.ValueDeclaration().ModifierFlags()
	return modifierFlags&ModifierFlagsStatic != 0
}

// See comment on `declareModuleMember` in `binder.go`.
func (s Symbol) CombinedLocalAndExportSymbolFlags() SymbolFlags {
	if e := s.ExportSymbol(); !e.IsNil() {
		return s.Flags() | e.Flags()
	}
	return s.Flags()
}
