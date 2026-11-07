package ast

import "unsafe"

// BinderSymbols numbers the binder symbols of a program's files with
// program-wide SymbolIds: file by file, from 1, without gaps. Each id has an
// entry in fixed chunks behind a small directory, so finding it is one load
// from the directory and one from the chunk. An entry copies the symbol's
// flags and name, the fields read most, so reading them never touches the
// symbol's record. Binder symbols don't change once their file is bound, so
// the copies stay true. The checkers of a program share its table.
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

// An entry is 24 bytes and holds no pointers: the symbol's file number, the
// id of the file's index 0 and the symbol's index there, and copies of its
// flags and name.
var _ [0]struct{} = [unsafe.Sizeof(binderSymbolEntry{}) - 24]struct{}{}

type binderSymbolEntry struct {
	flags SymbolFlags
	file  uint32
	base  uint32
	index uint32
	name  SymbolName
}

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
				flags: r.flags,
				file:  uint32(file),
				base:  base,
				index: uint32(i),
				name:  r.name,
			}
		}
		next += uint32(len(s.recs) - 1)
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
	return b.dir[id>>binderSymbolChunkShift][id&binderSymbolChunkMask].flags
}

// Name returns the name of the binder symbol id names.
func (b *BinderSymbols) Name(id SymbolId) SymbolName {
	return b.dir[id>>binderSymbolChunkShift][id&binderSymbolChunkMask].name
}

// Symbol returns a view of the binder symbol id names.
func (b *BinderSymbols) Symbol(id SymbolId) Symbol {
	e := b.entry(id)
	return Symbol{b.stores[e.file], e.index}
}

// record returns the record of the binder symbol id names, and the id of its
// store's index 0.
func (b *BinderSymbols) record(id SymbolId) (*SymbolStore, *symbolRecord, uint32) {
	e := b.entry(id)
	st := b.stores[e.file]
	return st, &st.recs[e.index], e.base
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
// store. The flags and name of a binder symbol come from the program's
// table, without touching the symbol's record.

func (st *SymbolStore) FlagsOf(id SymbolId) SymbolFlags {
	c := st.checker
	i := uint32(id)
	if i < c.binderCount {
		return c.binder.Flags(id)
	}
	return c.recs.At(i - c.binderCount).flags
}

func (st *SymbolStore) NameOf(id SymbolId) SymbolName {
	if uint32(id) < st.checker.binderCount {
		return st.checker.binder.Name(id)
	}
	return st.SymbolOf(id).Name()
}

func (st *SymbolStore) CheckFlagsOf(id SymbolId) CheckFlags {
	c := st.checker
	i := uint32(id)
	if i < c.binderCount {
		return 0
	}
	return c.recs.At(i - c.binderCount).checkFlags
}

func (st *SymbolStore) ParentOf(id SymbolId) SymbolId {
	if uint32(id) < st.checker.binderCount {
		_, r, base := st.checker.binder.record(id)
		if r.parent == 0 {
			return 0
		}
		return SymbolId(base + r.parent)
	}
	return st.IdOf(st.SymbolOf(id).Parent())
}

func (st *SymbolStore) DeclarationsOf(id SymbolId) Declarations {
	return st.SymbolOf(id).Declarations()
}

func (st *SymbolStore) ValueDeclarationOf(id SymbolId) Node {
	if uint32(id) < st.checker.binderCount {
		s, r, _ := st.checker.binder.record(id)
		if r.rare == 0 {
			if r.value == 0 {
				return Node{}
			}
			return Node{s.nodes, r.value}
		}
	}
	return st.SymbolOf(id).ValueDeclaration()
}

func (st *SymbolStore) MembersOf(id SymbolId) SymbolTable {
	return st.SymbolOf(id).Members()
}

func (st *SymbolStore) ExportsOf(id SymbolId) SymbolTable {
	return st.SymbolOf(id).Exports()
}

func (st *SymbolStore) ExportSymbolOf(id SymbolId) SymbolId {
	if uint32(id) < st.checker.binderCount {
		_, r, base := st.checker.binder.record(id)
		if r.exportSymbol == 0 {
			return 0
		}
		return SymbolId(base + r.exportSymbol)
	}
	return st.IdOf(st.SymbolOf(id).ExportSymbol())
}

func (st *SymbolStore) SetFlagsOf(id SymbolId, value SymbolFlags) {
	c := st.checker
	if uint32(id) < c.binderCount {
		c.binder.entry(id).flags = value
	} else {
		// A foreign symbol's record keeps a copy of its flags.
		c.recs.At(uint32(id) - c.binderCount).flags = value
	}
	st.SymbolOf(id).SetFlags(value)
}

func (st *SymbolStore) SetCheckFlagsOf(id SymbolId, value CheckFlags) {
	c := st.checker
	if uint32(id) >= c.binderCount {
		c.recs.At(uint32(id) - c.binderCount).checkFlags = value
	}
	st.SymbolOf(id).SetCheckFlags(value)
}

func (st *SymbolStore) SetNameOf(id SymbolId, value SymbolName) {
	if uint32(id) < st.checker.binderCount {
		st.checker.binder.entry(id).name = value
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
	c := st.checker
	i, r := c.recs.Add()
	r.flags = flags
	r.name = c.nameOf(name)
	return SymbolId(c.binderCount + i)
}

// NewSharedDataSymbolId returns the id of a new symbol of a checker's store
// that shares the fields of the symbol other names besides its flags.
func (st *SymbolStore) NewSharedDataSymbolId(flags SymbolFlags, other SymbolId) SymbolId {
	return st.IdOf(st.NewSharedDataSymbol(flags, st.SymbolOf(other)))
}
