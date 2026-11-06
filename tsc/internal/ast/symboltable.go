package ast

import (
	"iter"
	"math/bits"
	"unsafe"
)

// SymbolTable
//
// A SymbolTable maps names to symbols and iterates in insertion order, like Strada's Map, so that iterating a table
// is deterministic and visits symbols in the order the binder and checker added them. The zero SymbolTable is an
// empty table that can be read and iterated but not written, like a nil map.
//
// A table belongs to a symbol store, which holds the tables it makes in pools that hold no pointers, so that the
// garbage collector does not trace them: a record per table, and the tables' entries and hash indices in blocks of
// pages shared by all the store's tables. A SymbolTable is a view of one, the store plus the record's index; what
// stores a table for long keeps the index alone when the store is known, or a TableRef.
//
// The entries are kept in insertion order in one block of 12-byte (name, id) pairs. Most tables are small, so a table
// is searched linearly until it has more than symbolTableLinearMax entries, and only then gets a hash index of entry
// positions.
//
// An id means a symbol through the table's store. In a binder store's table it is the symbol's index in that store,
// and a symbol of any other store is kept aside as a view, with its position there as the id. In a checker store's
// table it is the symbol's id in the checker, which names any symbol.
//
// A table must not be changed with Set or Delete while it is being iterated.
type SymbolTable struct {
	st *SymbolStore
	t  uint32
}

const symbolTableLinearMax = 8

// foreignSymbol marks an id that is a position in a store's foreign symbols.
const foreignSymbol = 1 << 31

// symbolTables holds the tables of a store.
type symbolTables struct {
	// recs holds the tables' records; the first is unused, so that index 0
	// is no table.
	recs    []tableRec
	entries tablePool[symbolEntry]
	index   tablePool[int32]
	// foreign holds the symbols of other stores in a binder store's tables.
	foreign []Symbol
	// held holds the tables of other stores that the store's symbols and
	// nodes refer to.
	held []SymbolTable
}

// A table's record: the positions of its blocks of entries and of index
// slots, of 1<<capLog and 1<<idxLog elements, and its length. A table with
// no block has capLog 0, and one with no index idxLog 0.
type tableRec struct {
	entries uint32
	index   uint32
	n       uint32
	capLog  uint8
	idxLog  uint8
}

type symbolEntry struct {
	lo, hi uint32 // the name
	id     uint32
}

func (e *symbolEntry) name() SymbolName { return SymbolName(uint64(e.hi)<<32 | uint64(e.lo)) }

func hashSymbolTableKey(name SymbolName) uint64 {
	h := uint64(name) * 0x9e3779b97f4a7c15
	return h ^ h>>32
}

const (
	tablePageLog  = 12
	tablePageSize = 1 << tablePageLog
	// bigBlock marks the position of a block larger than a page, which is
	// allocated alone.
	bigBlock = 1 << 31
)

// A tablePool allocates blocks of power of two sizes from pages of
// tablePageSize elements. A position is a page's number shifted by
// tablePageLog plus an offset in it. Only the last page grows; the others
// are full. Freed blocks are kept in lists by size, linked through their
// first word.
type tablePool[T symbolEntry | int32] struct {
	pages [][]T
	big   [][]T
	free  [tablePageLog + 1]uint32 // position plus one of the first free block of each size
}

func (p *tablePool[T]) block(pos uint32, log uint8) []T {
	if pos&bigBlock != 0 {
		return p.big[pos&^bigBlock]
	}
	size := uint32(1) << log
	page := p.pages[pos>>tablePageLog]
	off := pos & (tablePageSize - 1)
	return page[off : off+size : off+size]
}

// link returns the first word of a free block, which holds the next free
// block's position plus one.
func link[T symbolEntry | int32](block []T) *uint32 {
	// SAFETY: both element types are at least four bytes, and start with a
	// four byte field.
	return (*uint32)(unsafe.Pointer(&block[0]))
}

// alloc returns the position of a cleared block of 1<<log elements.
func (p *tablePool[T]) alloc(log uint8) uint32 {
	size := 1 << log
	if size > tablePageSize {
		p.big = append(p.big, make([]T, size))
		return bigBlock | uint32(len(p.big)-1)
	}
	if head := p.free[log]; head != 0 {
		block := p.block(head-1, log)
		p.free[log] = *link(block)
		clear(block)
		return head - 1
	}
	n := len(p.pages)
	if n == 0 || len(p.pages[n-1])+size > tablePageSize {
		if n != 0 {
			p.freeRest(n - 1)
		}
		capacity := tablePageSize
		if n == 0 {
			capacity = max(size, 64)
		}
		p.pages = append(p.pages, make([]T, 0, capacity))
		n++
	}
	page := p.pages[n-1]
	pos := uint32(n-1)<<tablePageLog | uint32(len(page))
	if len(page)+size > cap(page) {
		grown := make([]T, len(page), min(max(2*cap(page), len(page)+size), tablePageSize))
		copy(grown, page)
		page = grown
	}
	p.pages[n-1] = page[:len(page)+size]
	return pos
}

// freeRest fills the last page and frees what it had left.
func (p *tablePool[T]) freeRest(i int) {
	page := p.pages[i]
	off := len(page)
	if off == tablePageSize {
		return
	}
	if cap(page) < tablePageSize {
		grown := make([]T, off, tablePageSize)
		copy(grown, page)
		page = grown
	}
	p.pages[i] = page[:tablePageSize]
	for rest := tablePageSize - off; rest != 0; {
		log := uint8(bits.Len(uint(rest)) - 1)
		p.release(uint32(i)<<tablePageLog|uint32(off), log)
		off += 1 << log
		rest -= 1 << log
	}
}

func (p *tablePool[T]) release(pos uint32, log uint8) {
	if pos&bigBlock != 0 {
		p.big[pos&^bigBlock] = nil
		return
	}
	*link(p.block(pos, log)) = p.free[log]
	p.free[log] = pos + 1
}

// trim drops the last page's spare capacity.
func (p *tablePool[T]) trim() {
	if n := len(p.pages); n != 0 {
		if page := p.pages[n-1]; len(page) != cap(page) {
			p.pages[n-1] = append([]T(nil), page...)
		}
	}
}

func (ts *symbolTables) trim() {
	if len(ts.recs) != cap(ts.recs) {
		ts.recs = append([]tableRec(nil), ts.recs...)
	}
	ts.entries.trim()
	ts.index.trim()
}

func ceilLog(n int) uint8 {
	return uint8(bits.Len(uint(n - 1)))
}

func indexSizeFor(n int) int {
	size := 16
	for size*3 < n*4 {
		size *= 2
	}
	return size
}

// NewSymbolTable returns a table of a store of its own, for tables made
// outside the binder and the checker.
func NewSymbolTable() SymbolTable {
	return NewSymbolTableWithCapacity(0)
}

func NewSymbolTableWithCapacity(capacity int) SymbolTable {
	return (&SymbolStore{}).NewTable(capacity)
}

// NewTable returns a table of st.
func (st *SymbolStore) NewTable(capacity int) SymbolTable {
	ts := &st.tables
	if len(ts.recs) == 0 {
		ts.recs = make([]tableRec, 1, 4)
	}
	ts.recs = append(ts.recs, tableRec{})
	t := SymbolTable{st, uint32(len(ts.recs) - 1)}
	if capacity > 0 {
		r := &ts.recs[t.t]
		r.capLog = max(ceilLog(capacity), 1)
		r.entries = ts.entries.alloc(r.capLog)
		if capacity > symbolTableLinearMax {
			r.idxLog = ceilLog(indexSizeFor(capacity))
			r.index = ts.index.alloc(r.idxLog)
		}
	}
	return t
}

// TableOf returns a table of st holding symbol under name.
func (st *SymbolStore) TableOf(name SymbolName, symbol Symbol) SymbolTable {
	t := st.NewTable(1)
	t.Set(name, symbol)
	return t
}

func SymbolTableOf(name SymbolName, symbol Symbol) SymbolTable {
	t := NewSymbolTableWithCapacity(1)
	t.Set(name, symbol)
	return t
}

// IsNil reports whether t is the zero table.
func (t SymbolTable) IsNil() bool { return t.t == 0 }

// Store returns the store t belongs to.
func (t SymbolTable) Store() *SymbolStore { return t.st }

func (t SymbolTable) entries() []symbolEntry {
	if t.t == 0 {
		return nil
	}
	ts := &t.st.tables
	r := &ts.recs[t.t]
	if r.capLog == 0 {
		return nil
	}
	return ts.entries.block(r.entries, r.capLog)[:r.n]
}

func (t SymbolTable) KeyAt(i int) SymbolName {
	return t.entries()[i].name()
}

func (t SymbolTable) SymbolAt(i int) Symbol {
	return t.symbolOf(t.entries()[i].id)
}

func (t SymbolTable) symbolOf(id uint32) Symbol {
	st := t.st
	if st.checker != nil {
		return st.SymbolOf(SymbolId(id))
	}
	if id&foreignSymbol != 0 {
		return st.tables.foreign[id&^foreignSymbol]
	}
	if id == 0 {
		return Symbol{}
	}
	return Symbol{st, id}
}

func (t SymbolTable) idOf(symbol Symbol) uint32 {
	st := t.st
	switch {
	case symbol.s == nil:
		return 0
	case st.checker != nil:
		return uint32(st.IdOf(symbol))
	case symbol.s == st:
		return symbol.i
	}
	ts := &st.tables
	ts.foreign = append(ts.foreign, symbol)
	return uint32(len(ts.foreign)-1) | foreignSymbol
}

func (t SymbolTable) Find(key SymbolName) int {
	if t.t == 0 {
		return -1
	}
	ts := &t.st.tables
	r := &ts.recs[t.t]
	if r.n == 0 {
		return -1
	}
	entries := ts.entries.block(r.entries, r.capLog)
	lo, hi := uint32(key), uint32(key>>32)
	if r.idxLog != 0 {
		index := ts.index.block(r.index, r.idxLog)
		mask := uint64(len(index) - 1)
		for slot := hashSymbolTableKey(key) & mask; ; slot = (slot + 1) & mask {
			p := index[slot]
			if p == 0 {
				return -1
			}
			if e := &entries[p-1]; e.lo == lo && e.hi == hi {
				return int(p - 1)
			}
		}
	}
	entries = entries[:r.n]
	for i := range entries {
		if e := &entries[i]; e.lo == lo && e.hi == hi {
			return i
		}
	}
	return -1
}

func (t SymbolTable) Len() int {
	if t.t == 0 {
		return 0
	}
	return int(t.st.tables.recs[t.t].n)
}

func (t SymbolTable) Get(key SymbolName) Symbol {
	if i := t.Find(key); i >= 0 {
		return t.SymbolAt(i)
	}
	return Symbol{}
}

func (t SymbolTable) Lookup(key SymbolName) (Symbol, bool) {
	if i := t.Find(key); i >= 0 {
		return t.SymbolAt(i), true
	}
	return Symbol{}, false
}

func (t SymbolTable) Set(key SymbolName, symbol Symbol) {
	t.setOrAdd(key, symbol, true)
}

func (t SymbolTable) AddIfAbsent(key SymbolName, symbol Symbol) bool {
	return t.setOrAdd(key, symbol, false)
}

func (t SymbolTable) setOrAdd(key SymbolName, symbol Symbol, replace bool) bool {
	if i := t.Find(key); i >= 0 {
		if replace {
			t.entries()[i].id = t.idOf(symbol)
		}
		return replace
	}
	t.add(key, t.idOf(symbol))
	return true
}

// add appends an entry for key, which t does not hold.
func (t SymbolTable) add(key SymbolName, id uint32) {
	ts := &t.st.tables
	r := ts.recs[t.t]
	if r.capLog == 0 || r.n == 1<<r.capLog {
		log := max(r.capLog+1, 1)
		pos := ts.entries.alloc(log)
		if r.capLog != 0 {
			copy(ts.entries.block(pos, log), ts.entries.block(r.entries, r.capLog))
			ts.entries.release(r.entries, r.capLog)
		}
		r.entries, r.capLog = pos, log
	}
	ts.entries.block(r.entries, r.capLog)[r.n] = symbolEntry{uint32(key), uint32(key >> 32), id}
	r.n++
	if r.idxLog != 0 && r.n*4 <= 3<<r.idxLog {
		insertIndex(ts.index.block(r.index, r.idxLog), key, r.n-1)
	} else if r.n > symbolTableLinearMax {
		ts.buildIndex(&r)
	}
	ts.recs[t.t] = r
}

func insertIndex(index []int32, key SymbolName, i uint32) {
	mask := uint64(len(index) - 1)
	slot := hashSymbolTableKey(key) & mask
	for index[slot] != 0 {
		slot = (slot + 1) & mask
	}
	index[slot] = int32(i + 1)
}

// buildIndex indexes r's entries, in a new block of index slots if they
// have outgrown the one r has.
func (ts *symbolTables) buildIndex(r *tableRec) {
	if log := ceilLog(indexSizeFor(int(r.n))); log > r.idxLog {
		if r.idxLog != 0 {
			ts.index.release(r.index, r.idxLog)
		}
		r.index, r.idxLog = ts.index.alloc(log), log
	} else {
		clear(ts.index.block(r.index, r.idxLog))
	}
	index := ts.index.block(r.index, r.idxLog)
	entries := ts.entries.block(r.entries, r.capLog)[:r.n]
	for i := range entries {
		insertIndex(index, entries[i].name(), uint32(i))
	}
}

func (t SymbolTable) Delete(key SymbolName) {
	d := t.Find(key)
	if d < 0 {
		return
	}
	ts := &t.st.tables
	r := &ts.recs[t.t]
	entries := ts.entries.block(r.entries, r.capLog)[:r.n]
	copy(entries[d:], entries[d+1:])
	r.n--
	if r.idxLog != 0 {
		ts.buildIndex(r)
	}
}

func (t SymbolTable) All() iter.Seq2[SymbolName, Symbol] {
	return func(yield func(SymbolName, Symbol) bool) {
		entries := t.entries()
		for i := range entries {
			if !yield(entries[i].name(), t.symbolOf(entries[i].id)) {
				return
			}
		}
	}
}

func (t SymbolTable) Keys() iter.Seq[SymbolName] {
	return func(yield func(SymbolName) bool) {
		entries := t.entries()
		for i := range entries {
			if !yield(entries[i].name()) {
				return
			}
		}
	}
}

func (t SymbolTable) Values() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		entries := t.entries()
		for i := range entries {
			if !yield(t.symbolOf(entries[i].id)) {
				return
			}
		}
	}
}

// Clone returns a copy of t in t's store.
func (t SymbolTable) Clone() SymbolTable {
	if t.t == 0 {
		return SymbolTable{}
	}
	return t.CloneTo(t.st)
}

// CloneTo returns a copy of t in st, which must be t's store or a checker's
// store.
func (t SymbolTable) CloneTo(st *SymbolStore) SymbolTable {
	if t.t == 0 {
		return SymbolTable{}
	}
	n := t.Len()
	clone := st.NewTable(n)
	if n == 0 {
		return clone
	}
	if st != t.st && st.checker == nil {
		for name, symbol := range t.All() {
			clone.Set(name, symbol)
		}
		return clone
	}
	ts := &st.tables
	r := ts.recs[clone.t]
	dst := ts.entries.block(r.entries, r.capLog)[:n]
	copy(dst, t.entries())
	if st != t.st {
		for i := range dst {
			dst[i].id = uint32(t.checkerId(st, dst[i].id))
		}
	}
	r.n = uint32(n)
	if r.idxLog != 0 {
		ts.buildIndex(&r)
	}
	ts.recs[clone.t] = r
	return clone
}

func (t SymbolTable) Clear() {
	if t.t == 0 {
		return
	}
	ts := &t.st.tables
	r := &ts.recs[t.t]
	r.n = 0
	if r.idxLog != 0 {
		clear(ts.index.block(r.index, r.idxLog))
	}
}

// The methods below read and write a table in the ids of a checker's store
// st. A table of another store is read through its own store, and written
// with the symbols the ids name.

// checkerId returns the id in st of the symbol id names in t.
func (t SymbolTable) checkerId(st *SymbolStore, id uint32) SymbolId {
	s := t.st
	switch {
	case s == st:
		return SymbolId(id)
	case id == 0:
		return 0
	case s.checker == nil && id&foreignSymbol == 0:
		if base, ok := st.BaseOf(s); ok {
			return SymbolId(base + id)
		}
	}
	return st.IdOf(t.symbolOf(id))
}

func (t SymbolTable) GetId(st *SymbolStore, key SymbolName) SymbolId {
	if i := t.Find(key); i >= 0 {
		return t.checkerId(st, t.entries()[i].id)
	}
	return 0
}

func (t SymbolTable) LookupId(st *SymbolStore, key SymbolName) (SymbolId, bool) {
	if i := t.Find(key); i >= 0 {
		return t.checkerId(st, t.entries()[i].id), true
	}
	return 0, false
}

func (t SymbolTable) IdAt(st *SymbolStore, i int) SymbolId {
	return t.checkerId(st, t.entries()[i].id)
}

func (t SymbolTable) AllIds(st *SymbolStore) iter.Seq2[SymbolName, SymbolId] {
	return func(yield func(SymbolName, SymbolId) bool) {
		entries := t.entries()
		for i := range entries {
			if !yield(entries[i].name(), t.checkerId(st, entries[i].id)) {
				return
			}
		}
	}
}

func (t SymbolTable) ValueIds(st *SymbolStore) iter.Seq[SymbolId] {
	return func(yield func(SymbolId) bool) {
		entries := t.entries()
		for i := range entries {
			if !yield(t.checkerId(st, entries[i].id)) {
				return
			}
		}
	}
}

func (t SymbolTable) SetId(st *SymbolStore, key SymbolName, id SymbolId) {
	t.setOrAddId(st, key, id, true)
}

func (t SymbolTable) AddIfAbsentId(st *SymbolStore, key SymbolName, id SymbolId) bool {
	return t.setOrAddId(st, key, id, false)
}

func (t SymbolTable) setOrAddId(st *SymbolStore, key SymbolName, id SymbolId, replace bool) bool {
	if t.st != st {
		var symbol Symbol
		if id != 0 {
			symbol = st.SymbolOf(id)
		}
		return t.setOrAdd(key, symbol, replace)
	}
	if i := t.Find(key); i >= 0 {
		if replace {
			t.entries()[i].id = uint32(id)
		}
		return replace
	}
	t.add(key, uint32(id))
	return true
}

// TableOfId returns a table of a checker's store st holding the symbol id
// names under name.
func (st *SymbolStore) TableOfId(name SymbolName, id SymbolId) SymbolTable {
	t := st.NewTable(1)
	t.SetId(st, name, id)
	return t
}

// tableIndex returns what st's records keep for t: its index if it is st's
// own, or else its position in st's held tables, marked with heldTable.
func (st *SymbolStore) tableIndex(t SymbolTable) uint32 {
	switch {
	case t.t == 0:
		return 0
	case t.st == st:
		return t.t
	}
	ts := &st.tables
	ts.held = append(ts.held, t)
	return uint32(len(ts.held)-1) | heldTable
}

const heldTable = 1 << 31

// tableAt returns the table st's records keep as x.
func (st *SymbolStore) tableAt(x uint32) SymbolTable {
	switch {
	case x == 0:
		return SymbolTable{}
	case x&heldTable != 0:
		return st.tables.held[x&^heldTable]
	}
	return SymbolTable{st, x}
}

// A TableRef names a symbol table for a checker's store: a table of the
// store itself, of a store numbered in its file table, or one of the
// tables it holds aside. The zero TableRef is no table. It holds no
// pointers, for records the garbage collector should not trace.
type TableRef struct {
	store uint32
	t     uint32
}

const (
	tableRefSelf = ^uint32(0)
	tableRefHeld = ^uint32(0) - 1
)

func (r TableRef) IsNil() bool { return r.t == 0 }

// RefOfTable returns the ref of t in a checker's store st.
func (st *SymbolStore) RefOfTable(t SymbolTable) TableRef {
	switch {
	case t.t == 0:
		return TableRef{}
	case t.st == st:
		return TableRef{tableRefSelf, t.t}
	case t.st.nodes != nil && t.st.checker == nil:
		return TableRef{st.checker.table.StoreNumber(t.st.nodes), t.t}
	}
	c := st.checker
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.heldIndex == nil {
		c.heldIndex = make(map[SymbolTable]uint32)
	}
	i, ok := c.heldIndex[t]
	if !ok {
		st.tables.held = append(st.tables.held, t)
		i = uint32(len(st.tables.held))
		c.heldIndex[t] = i
	}
	return TableRef{tableRefHeld, i}
}

// TableOfRef returns the table r names in a checker's store st.
func (st *SymbolStore) TableOfRef(r TableRef) SymbolTable {
	switch {
	case r.t == 0:
		return SymbolTable{}
	case r.store == tableRefSelf:
		return SymbolTable{st, r.t}
	case r.store == tableRefHeld:
		c := st.checker
		c.mu.Lock()
		defer c.mu.Unlock()
		return st.tables.held[r.t-1]
	}
	return SymbolTable{&st.checker.table.Store(r.store).symbols, r.t}
}
