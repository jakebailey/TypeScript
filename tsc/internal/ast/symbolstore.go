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
	// is an index in it plus one, with rareGeneral set when the symbol's
	// declarations are not all in the file's node store.
	rare []symbolRare

	// In a checker's store, the program's binder symbols have ids from 1 to
	// binderCount, read through binder; the checker's own symbols follow
	// them from binderCount, held in trecs by id minus binderCount.
	binder      *BinderSymbols
	binderCount uint32
	trecs       []checkerSymbolRecord

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

// rareGeneral marks, in a record's rare, a symbol whose declarations and
// value declaration are held whole in its rare fields.
const rareGeneral = 1 << 31

type symbolRare struct {
	// rest holds the declarations after the first, or, under rareGeneral,
	// all declarations and the value declaration, which are then not all in
	// the file's node store.
	rest    *declRest
	members SymbolTable
	exports SymbolTable
}

// A checker's symbol is 40 bytes and holds no pointers. Its declarations
// are refs in the checker's file table: one inline, or a list kept aside,
// or, for a symbol that shares another's fields, none of its own.
var _ [0]struct{} = [unsafe.Sizeof(checkerSymbolRecord{}) - 40]struct{}{}

type checkerSymbolRecord struct {
	flags      SymbolFlags
	checkFlags CheckFlags
	parent     SymbolId
	name       SymbolName
	value      NodeRef
	decl       NodeRef
}

// Markers in the file of a checker symbol's decl.
const (
	// declShared marks a symbol that shares the fields of the symbol whose
	// id is decl's index.
	declShared = ^uint32(0)
	// declForeign marks a symbol that stands for the symbol of another
	// store at decl's index in the checker's foreign symbols.
	declForeign = ^uint32(0) - 1
	// declList marks a symbol whose declarations are the list at decl's
	// index in the checker's lists.
	declList = ^uint32(0) - 2
)

type symbolExtra struct {
	members      SymbolTable
	exports      SymbolTable
	exportSymbol SymbolId
}

type checkerSymbols struct {
	// lists holds the declarations of symbols with more than one.
	lists core.Pages[Declarations]
	// extra holds the rare fields of the checker's symbols, by index.
	extraMu sync.RWMutex
	extra   map[uint32]*symbolExtra

	// table numbers the stores of the checker's symbols' declarations.
	table *FileTable

	// A symbol of a store outside the program gets a checker symbol that
	// stands for it the first time it is met.
	mu         sync.Mutex
	foreign    core.Pages[Symbol]
	foreignIds map[Symbol]SymbolId
}

// NewCheckerSymbolStore returns a store for a checker's symbols, whose ids
// number the symbols of the files of t first.
func NewCheckerSymbolStore(t *FileTable) *SymbolStore {
	if t == nil {
		t = NewFileTable(nil)
	}
	b := t.BinderSymbols()
	return &SymbolStore{binder: b, binderCount: b.count, checker: &checkerSymbols{table: t}}
}

// add appends a record to a checker's store and returns its index. Records
// move when the store grows, so a record's address is good only until the
// next add.
func (st *SymbolStore) add(r checkerSymbolRecord) uint32 {
	st.trecs = append(st.trecs, r)
	return uint32(len(st.trecs) - 1)
}

// rec returns the record at index i of a checker's store.
func (st *SymbolStore) rec(i uint32) *checkerSymbolRecord {
	return &st.trecs[i]
}

// NewSymbol returns a new symbol of a checker's store.
func (st *SymbolStore) NewSymbol(flags SymbolFlags, name SymbolName) Symbol {
	return Symbol{st, st.add(checkerSymbolRecord{flags: flags, name: name})}
}

// NewSharedDataSymbol returns a new symbol of a checker's store that shares
// the fields other has besides its flags.
func (st *SymbolStore) NewSharedDataSymbol(flags SymbolFlags, other Symbol) Symbol {
	if other.s.checker != nil {
		if o := other.s.rec(other.i); o.decl.file == declShared {
			other = other.s.SymbolOf(SymbolId(o.decl.node))
		}
	}
	owner := st.IdOf(other)
	return Symbol{st, st.add(checkerSymbolRecord{flags: flags, decl: NodeRef{declShared, uint32(owner)}})}
}

// Count returns the number of symbols the store holds.
func (st *SymbolStore) Count() int {
	if st.checker != nil {
		return len(st.trecs)
	}
	return max(len(st.recs)-1, 0)
}

// BinderSymbolCount returns the first id of a checker store's own symbols.
func (st *SymbolStore) BinderSymbolCount() uint32 {
	return st.binderCount
}

// IdOf returns the id of s in a checker's store.
func (st *SymbolStore) IdOf(s Symbol) SymbolId {
	if s.s == nil {
		return 0
	}
	if base, ok := st.BaseOf(s.s); ok {
		return SymbolId(base + s.i)
	}
	return st.foreignId(s)
}

// BaseOf returns the id of index 0 of a store of the program's files or of
// the checker's store itself, for callers that cache it by store.
func (st *SymbolStore) BaseOf(s *SymbolStore) (uint32, bool) {
	if s == st {
		return st.binderCount, true
	}
	return st.binder.BaseOf(s)
}

// foreignId returns the id of a symbol of a store that is neither the
// checker's nor one of the program's files' (a file outside the program, a
// lazily parsed JSDoc store, a symbol made outside a checker). Such stores
// may still grow, so their symbols can't be numbered as a block. Each gets
// a checker symbol that stands for it: SymbolOf turns the id back into the
// foreign symbol, and the record keeps a copy of its flags and check flags,
// which the setters keep in step.
func (st *SymbolStore) foreignId(s Symbol) SymbolId {
	c := st.checker
	c.mu.Lock()
	defer c.mu.Unlock()
	if id, ok := c.foreignIds[s]; ok {
		return id
	}
	if c.foreignIds == nil {
		c.foreignIds = make(map[Symbol]SymbolId)
	}
	k, p := c.foreign.Add()
	*p = s
	i := st.add(checkerSymbolRecord{flags: s.Flags(), checkFlags: s.CheckFlags(), decl: NodeRef{declForeign, k}})
	id := SymbolId(st.binderCount + i)
	c.foreignIds[s] = id
	return id
}

// SymbolOf returns the symbol id names in a checker's store.
func (st *SymbolStore) SymbolOf(id SymbolId) Symbol {
	if id == 0 {
		return Symbol{}
	}
	i := uint32(id)
	if i >= st.binderCount {
		if r := st.rec(i - st.binderCount); r.decl.file == declForeign {
			return *st.checker.foreign.At(r.decl.node)
		}
		return Symbol{st, i - st.binderCount}
	}
	return st.binder.Symbol(id)
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
		return &s.s.rare[r&^rareGeneral-1]
	}
	return nil
}

func (s Symbol) ensureRare() *symbolRare {
	r := &s.s.recs[s.i]
	if r.rare == 0 {
		s.s.rare = append(s.s.rare, symbolRare{})
		r.rare = uint32(len(s.s.rare))
	}
	return &s.s.rare[r.rare&^rareGeneral-1]
}

// isGeneral reports whether a binder symbol's declarations are held whole
// in its rare fields.
func (s Symbol) isGeneral() bool {
	return s.s.recs[s.i].rare&rareGeneral != 0
}

// setRest sets the declarations a binder symbol holds in its rare fields.
func (s Symbol) setRest(rest *declRest, general bool) {
	rare := s.ensureRare()
	rare.rest = rest
	r := &s.s.recs[s.i]
	if general {
		r.rare |= rareGeneral
	} else {
		r.rare &^= rareGeneral
	}
}

// fields returns the record holding the fields of a symbol of a checker's
// store besides its flags, and the symbol it belongs to, which differs from
// s if s shares another's fields. The record is nil if that symbol is a
// binder symbol.
func (s Symbol) fields() (*checkerSymbolRecord, Symbol) {
	r := s.s.rec(s.i)
	if r.decl.file != declShared {
		return r, s
	}
	o := s.s.SymbolOf(SymbolId(r.decl.node))
	if o.s.checker == nil {
		return nil, o
	}
	return o.s.rec(o.i), o
}

// NewSymbol returns a symbol in a store of its own, for symbols made outside
// a checker.
func NewSymbol() Symbol {
	return NewCheckerSymbolStore(nil).NewSymbol(SymbolFlagsNone, EmptySymbolName)
}

func (s Symbol) Flags() SymbolFlags {
	if s.s.checker != nil {
		return s.s.rec(s.i).flags
	}
	return s.s.recs[s.i].flags
}

func (s Symbol) CheckFlags() CheckFlags {
	if s.s.checker != nil {
		return s.s.rec(s.i).checkFlags
	}
	return 0
}

func (s Symbol) Name() SymbolName {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			return o.Name()
		}
		return r.name
	}
	return s.s.recs[s.i].name
}

func (s Symbol) Parent() Symbol {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			return o.Parent()
		}
		return o.s.SymbolOf(r.parent)
	}
	if p := s.s.recs[s.i].parent; p != 0 {
		return Symbol{s.s, p}
	}
	return Symbol{}
}

func (s Symbol) Declarations() Declarations {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			return o.Declarations()
		}
		c := o.s.checker
		switch {
		case r.decl.file == declList:
			return *c.lists.At(r.decl.node)
		case r.decl.node == 0:
			return Declarations{}
		}
		return Declarations{s: c.table.Store(r.decl.file), first: r.decl.node}
	}
	r := s.record()
	if rare := s.rareOf(); rare != nil && rare.rest != nil {
		if s.isGeneral() {
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
		r, o := s.fields()
		if r == nil {
			return o.ValueDeclaration()
		}
		return o.s.checker.table.Node(r.value)
	}
	if s.isGeneral() {
		return s.rareOf().rest.value
	}
	if v := s.s.recs[s.i].value; v != 0 {
		return Node{s.s.nodes, v}
	}
	return Node{}
}

// extraOf returns the rare fields of a symbol of a checker's store, or nil,
// and the symbol they belong to.
func (s Symbol) extraOf() (*symbolExtra, Symbol) {
	r, o := s.fields()
	if r == nil {
		return nil, o
	}
	c := o.s.checker
	c.extraMu.RLock()
	e := c.extra[o.i]
	c.extraMu.RUnlock()
	return e, o
}

// ensureExtra returns the rare fields of a symbol of a checker's store,
// making them if need be, or nil if the symbol is a binder symbol's.
func (s Symbol) ensureExtra() (*symbolExtra, Symbol) {
	e, o := s.extraOf()
	if e == nil && o.s.checker != nil {
		c := o.s.checker
		e = &symbolExtra{}
		c.extraMu.Lock()
		if c.extra == nil {
			c.extra = make(map[uint32]*symbolExtra)
		}
		c.extra[o.i] = e
		c.extraMu.Unlock()
	}
	return e, o
}

func (s Symbol) Members() SymbolTable {
	if s.s.checker != nil {
		e, o := s.extraOf()
		if e == nil {
			if o.s.checker == nil {
				return o.Members()
			}
			return nil
		}
		return e.members
	}
	if rare := s.rareOf(); rare != nil {
		return rare.members
	}
	return nil
}

func (s Symbol) Exports() SymbolTable {
	if s.s.checker != nil {
		e, o := s.extraOf()
		if e == nil {
			if o.s.checker == nil {
				return o.Exports()
			}
			return nil
		}
		return e.exports
	}
	if rare := s.rareOf(); rare != nil {
		return rare.exports
	}
	return nil
}

func (s Symbol) ExportSymbol() Symbol {
	if s.s.checker != nil {
		e, o := s.extraOf()
		if e == nil {
			if o.s.checker == nil {
				return o.ExportSymbol()
			}
			return Symbol{}
		}
		return o.s.SymbolOf(e.exportSymbol)
	}
	if e := s.s.recs[s.i].exportSymbol; e != 0 {
		return Symbol{s.s, e}
	}
	return Symbol{}
}

func (s Symbol) SetFlags(value SymbolFlags) {
	if s.s.checker != nil {
		s.s.rec(s.i).flags = value
		return
	}
	s.s.recs[s.i].flags = value
}

func (s Symbol) SetCheckFlags(value CheckFlags) {
	if s.s.checker != nil {
		s.s.rec(s.i).checkFlags = value
		return
	}
	debug.Assert(value == 0, "A binder symbol has no check flags")
}

func (s Symbol) SetName(value SymbolName) {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			o.SetName(value)
			return
		}
		r.name = value
		return
	}
	s.s.recs[s.i].name = value
}

func (s Symbol) SetDeclarations(value Declarations) {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			o.SetDeclarations(value)
			return
		}
		c := o.s.checker
		switch {
		case value.rest == nil && (value.s == nil || value.first == 0):
			r.decl = NodeRef{}
		case value.rest == nil:
			r.decl = c.table.Ref(Node{value.s, value.first})
		case r.decl.file == declList:
			*c.lists.At(r.decl.node) = value
		default:
			i, p := c.lists.Add()
			*p = value
			r.decl = NodeRef{declList, i}
		}
		return
	}
	valueDeclaration := s.ValueDeclaration()
	r := s.record()
	switch {
	case value.s == s.s.nodes:
		r.decl = value.first
		if value.rest != nil || r.rare != 0 {
			s.setRest(value.rest, false)
		}
	case value.s == nil && value.rest == nil:
		r.decl = 0
		if r.rare != 0 {
			s.setRest(nil, false)
		}
	default:
		r.decl = 0
		s.setRest(&declRest{nodes: value.Nodes()}, true)
	}
	r.value = 0
	s.SetValueDeclaration(valueDeclaration)
}

func (s Symbol) SetValueDeclaration(value Node) {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			o.SetValueDeclaration(value)
			return
		}
		r.value = o.s.checker.table.Ref(value)
		return
	}
	r := s.record()
	switch {
	case s.isGeneral():
		rare := s.rareOf()
		rare.rest = &declRest{nodes: rare.rest.nodes, value: value}
	case value.IsNil():
		r.value = 0
	case value.s == s.s.nodes:
		r.value = value.node
	default:
		nodes := s.Declarations().Nodes()
		r.decl, r.value = 0, 0
		s.setRest(&declRest{nodes: nodes, value: value}, true)
	}
}

func (s Symbol) SetParent(value Symbol) {
	if s.s.checker != nil {
		r, o := s.fields()
		if r == nil {
			o.SetParent(value)
			return
		}
		// Finding value's id may add a record, moving r.
		id := o.s.IdOf(value)
		o.s.rec(o.i).parent = id
		return
	}
	debug.Assert(value.IsNil() || value.s == s.s, "A binder symbol's parent is in its file")
	s.s.recs[s.i].parent = value.i
}

func (s Symbol) SetMembers(value SymbolTable) {
	if s.s.checker != nil {
		if value == nil && s.Members() == nil {
			return
		}
		e, o := s.ensureExtra()
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
		e, o := s.ensureExtra()
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
		e, o := s.ensureExtra()
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
