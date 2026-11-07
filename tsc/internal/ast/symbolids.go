package ast

import "unsafe"

// BinderSymbols numbers the binder symbols of a program's files with
// program-wide SymbolIds: file by file, from 1, without gaps. Each id has an
// entry in fixed chunks behind a small directory, so finding it is one load
// from the directory and one from the chunk. An entry points at the symbol's
// record and copies its flags and name, the fields read most, so reading
// them needs no further load; the other fields are one load away. Binder
// symbols don't change once their file is bound, so the copies stay true.
// The checkers of a program share its table.
type BinderSymbols struct {
	table *FileTable
	dir   []*binderSymbolChunk
	// stores and bases give each file's symbol store and the id of its index
	// 0, by the file's number in table.
	stores []*SymbolStore
	bases  []uint32
	// baseOf gives the same for stores whose slot names another table.
	baseOf map[*SymbolStore]uint32
	// count is the first id after the binder symbols'.
	count uint32
}

const (
	binderSymbolChunkShift = 12
	binderSymbolChunkSize  = 1 << binderSymbolChunkShift
	binderSymbolChunkMask  = binderSymbolChunkSize - 1
)

type binderSymbolChunk [binderSymbolChunkSize]binderSymbolEntry

// An entry is 32 bytes: the symbol's record and the node store of its
// file, the id of the file's index 0, and copies of the symbol's flags and
// name.
var _ [0]struct{} = [unsafe.Sizeof(binderSymbolEntry{}) - 32]struct{}{}

type binderSymbolEntry struct {
	rec   *symbolRecord
	nodes *NodeStore
	flags SymbolFlags
	base  uint32
	name  SymbolName
}

// noSymbolRecord is the record of id 0, so that reading a field of no
// symbol reads zero.
var noSymbolRecord symbolRecord

// BinderSymbols returns the table numbering the binder symbols of t's
// files, made on first use, once the files are bound.
func (t *FileTable) BinderSymbols() *BinderSymbols {
	t.binderOnce.Do(func() {
		t.binder = newBinderSymbols(t)
	})
	return t.binder
}

func newBinderSymbols(t *FileTable) *BinderSymbols {
	b := &BinderSymbols{
		table:  t,
		stores: make([]*SymbolStore, len(t.files)),
		bases:  make([]uint32, len(t.files)),
		baseOf: make(map[*SymbolStore]uint32, len(t.files)),
	}
	next := uint32(1)
	for file, f := range t.files {
		s := &f.symbols
		if len(s.recs) <= 1 {
			continue
		}
		base := next - 1
		b.stores[file] = s
		b.bases[file] = base
		b.baseOf[s] = base
		for i := 1; i < len(s.recs); i++ {
			id := base + uint32(i)
			k := int(id >> binderSymbolChunkShift)
			if k == len(b.dir) {
				b.dir = append(b.dir, new(binderSymbolChunk))
			}
			r := &s.recs[i]
			b.dir[k][id&binderSymbolChunkMask] = binderSymbolEntry{
				rec:   r,
				nodes: f,
				flags: r.flags,
				base:  base,
				name:  r.name,
			}
		}
		next += uint32(len(s.recs) - 1)
	}
	if len(b.dir) != 0 {
		b.dir[0][0].rec = &noSymbolRecord
	}
	b.count = next
	return b
}

// Count returns the first id after the binder symbols'.
func (b *BinderSymbols) Count() uint32 { return b.count }

func (b *BinderSymbols) entry(id SymbolId) *binderSymbolEntry {
	return &b.dir[id>>binderSymbolChunkShift][id&binderSymbolChunkMask]
}

// Flags returns the flags of the binder symbol id names.
func (b *BinderSymbols) Flags(id SymbolId) SymbolFlags {
	return b.entry(id).flags
}

// Name returns the name of the binder symbol id names.
func (b *BinderSymbols) Name(id SymbolId) SymbolName {
	return b.entry(id).name
}

// Symbol returns a view of the binder symbol id names.
func (b *BinderSymbols) Symbol(id SymbolId) Symbol {
	e := b.entry(id)
	if e.nodes == nil {
		return Symbol{}
	}
	return Symbol{&e.nodes.symbols, uint32(id) - e.base}
}

// BaseOf returns the id of index 0 of a store of the table's files.
func (b *BinderSymbols) BaseOf(s *SymbolStore) (uint32, bool) {
	if s.nodes != nil {
		if slot := s.nodes.slot.Load(); slot != nil && slot.table == b.table && slot.file < uint32(len(b.stores)) && b.stores[slot.file] == s {
			return b.bases[slot.file], true
		}
	}
	base, ok := b.baseOf[s]
	return base, ok
}

// The accessors below read and write the symbol id names in a checker's
// store. A binder symbol is read through the program's table; a symbol of
// the checker's own is read from its record, unless it shares another
// symbol's fields or stands for a symbol of another store, which the
// markers at the top of its decl's file range say.

// isOwn reports whether a checker symbol's record holds its own fields.
func (r *checkerSymbolRecord) isOwn() bool {
	return r.decl.file < declForeign
}

func (st *SymbolStore) FlagsOf(id SymbolId) SymbolFlags {
	if uint32(id) < st.binderCount {
		return st.binder.entry(id).flags
	}
	return st.trecs[uint32(id)-st.binderCount].flags
}

func (st *SymbolStore) NameOf(id SymbolId) SymbolName {
	if uint32(id) < st.binderCount {
		return st.binder.entry(id).name
	}
	return st.checkerSymbolName(id)
}

func (st *SymbolStore) checkerSymbolName(id SymbolId) SymbolName {
	if r := &st.trecs[uint32(id)-st.binderCount]; r.isOwn() {
		return r.name
	}
	return st.SymbolOf(id).Name()
}

func (st *SymbolStore) CheckFlagsOf(id SymbolId) CheckFlags {
	if uint32(id) < st.binderCount {
		return 0
	}
	return st.trecs[uint32(id)-st.binderCount].checkFlags
}

// The accessors below are too large for the compiler to inline, so each
// reads its common cases without further calls.

func (st *SymbolStore) ParentOf(id SymbolId) SymbolId {
	if uint32(id) < st.binderCount {
		e := st.binder.entry(id)
		if p := e.rec.parent; p != 0 {
			return SymbolId(e.base + p)
		}
		return 0
	}
	if r := &st.trecs[uint32(id)-st.binderCount]; r.isOwn() {
		return r.parent
	}
	return st.IdOf(st.SymbolOf(id).Parent())
}

func (st *SymbolStore) DeclarationsOf(id SymbolId) Declarations {
	if uint32(id) < st.binderCount {
		e := st.binder.entry(id)
		if r := e.rec; r.rare == 0 {
			if r.decl == 0 {
				return Declarations{}
			}
			return Declarations{s: e.nodes, first: r.decl}
		}
	} else if r := &st.trecs[uint32(id)-st.binderCount]; r.decl.file < declList {
		if r.decl.node == 0 {
			return Declarations{}
		}
		return Declarations{s: st.checker.table.Store(r.decl.file), first: r.decl.node}
	}
	return st.SymbolOf(id).Declarations()
}

func (st *SymbolStore) ValueDeclarationOf(id SymbolId) Node {
	if uint32(id) < st.binderCount {
		e := st.binder.entry(id)
		if r := e.rec; r.rare&rareGeneral == 0 {
			if r.value == 0 {
				return Node{}
			}
			return Node{e.nodes, r.value}
		}
	} else if r := &st.trecs[uint32(id)-st.binderCount]; r.isOwn() {
		if r.value.node == 0 {
			return Node{}
		}
		return Node{st.checker.table.Store(r.value.file), r.value.node}
	}
	return st.SymbolOf(id).ValueDeclaration()
}

func (st *SymbolStore) MembersOf(id SymbolId) SymbolTable {
	if uint32(id) < st.binderCount {
		e := st.binder.entry(id)
		if r := e.rec.rare; r != 0 {
			return e.nodes.symbols.rare[r&^rareGeneral-1].members
		}
		return nil
	}
	return st.SymbolOf(id).Members()
}

func (st *SymbolStore) ExportsOf(id SymbolId) SymbolTable {
	if uint32(id) < st.binderCount {
		e := st.binder.entry(id)
		if r := e.rec.rare; r != 0 {
			return e.nodes.symbols.rare[r&^rareGeneral-1].exports
		}
		return nil
	}
	return st.SymbolOf(id).Exports()
}

func (st *SymbolStore) ExportSymbolOf(id SymbolId) SymbolId {
	if uint32(id) < st.binderCount {
		e := st.binder.entry(id)
		if p := e.rec.exportSymbol; p != 0 {
			return SymbolId(e.base + p)
		}
		return 0
	}
	return st.IdOf(st.SymbolOf(id).ExportSymbol())
}

func (st *SymbolStore) SetFlagsOf(id SymbolId, value SymbolFlags) {
	if uint32(id) < st.binderCount {
		st.binder.entry(id).flags = value
	} else {
		// A foreign symbol's record keeps a copy of its flags.
		st.trecs[uint32(id)-st.binderCount].flags = value
	}
	st.SymbolOf(id).SetFlags(value)
}

func (st *SymbolStore) SetCheckFlagsOf(id SymbolId, value CheckFlags) {
	if uint32(id) >= st.binderCount {
		st.trecs[uint32(id)-st.binderCount].checkFlags = value
	}
	st.SymbolOf(id).SetCheckFlags(value)
}

func (st *SymbolStore) SetNameOf(id SymbolId, value SymbolName) {
	if uint32(id) < st.binderCount {
		st.binder.entry(id).name = value
	}
	st.SymbolOf(id).SetName(value)
}

func (st *SymbolStore) SetDeclarationsOf(id SymbolId, value Declarations) {
	st.SymbolOf(id).SetDeclarations(value)
}

func (st *SymbolStore) SetValueDeclarationOf(id SymbolId, value Node) {
	st.SymbolOf(id).SetValueDeclaration(value)
}

func (st *SymbolStore) SetParentOf(id SymbolId, value SymbolId) {
	st.SymbolOf(id).SetParent(st.SymbolOf(value))
}

func (st *SymbolStore) SetMembersOf(id SymbolId, value SymbolTable) {
	st.SymbolOf(id).SetMembers(value)
}

func (st *SymbolStore) SetExportsOf(id SymbolId, value SymbolTable) {
	st.SymbolOf(id).SetExports(value)
}

func (st *SymbolStore) SetExportSymbolOf(id SymbolId, value SymbolId) {
	st.SymbolOf(id).SetExportSymbol(st.SymbolOf(value))
}

// NewSymbolId returns the id of a new symbol of a checker's store.
func (st *SymbolStore) NewSymbolId(flags SymbolFlags, name SymbolName) SymbolId {
	return SymbolId(st.binderCount + st.add(checkerSymbolRecord{flags: flags, name: name}))
}

// NewSharedDataSymbolId returns the id of a new symbol of a checker's store
// that shares the fields of the symbol other names besides its flags.
func (st *SymbolStore) NewSharedDataSymbolId(flags SymbolFlags, other SymbolId) SymbolId {
	return st.IdOf(st.NewSharedDataSymbol(flags, st.SymbolOf(other)))
}
