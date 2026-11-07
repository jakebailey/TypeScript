package ast

import (
	"errors"
	"iter"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/microsoft/TypeScript/tsc/internal/json"
)

// SymbolName is an interned name used for symbol identity and lookup.
// Its zero value is EmptySymbolName.
//
// The low bits hold an intern table ID; the upper bits encode derived names
// and tag internal names. See symbolname_derived.go.
//
// Fixed names, such as SymbolNamePrototype, are constants generated from the
// list in tools/scripts/tsc/symbolNames.ts by `npx hereby generate:ast`.
type SymbolName uint64

func MakeSymbolName(text string) SymbolName {
	return symbolNames.make(text, hashSymbolNameText(text))
}

func (name SymbolName) String() string {
	if name.derivedKind() != derivedNamePlain {
		return name.derivedValue()
	}
	return symbolNames.text(uint32(name))
}

// MarshalText uses the escaped spelling so internal names remain valid UTF-8.
func (name SymbolName) MarshalText() ([]byte, error) {
	text := EscapeSymbolName(name)
	if !utf8.ValidString(text) {
		return nil, errors.New("symbol name contains invalid UTF-8")
	}
	return []byte(text), nil
}

func (name *SymbolName) UnmarshalText(text []byte) error {
	if !utf8.Valid(text) {
		return errors.New("symbol name contains invalid UTF-8")
	}
	*name = UnescapeSymbolName(string(text))
	return nil
}

func (name *SymbolName) UnmarshalJSON(data []byte) error {
	if !utf8.Valid(data) {
		return errors.New("symbol name contains invalid UTF-8")
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	return name.UnmarshalText([]byte(text))
}

// SymbolTable
//
// A SymbolTable maps names to symbols and iterates in insertion order, like Strada's Map, so that iterating a table
// is deterministic and visits symbols in the order the binder and checker added them. A nil SymbolTable is an empty
// table that can be read and iterated but not written, like a nil map.
//
// The entries are kept in insertion order in one slice of 12-byte (name, id) pairs. Most tables are small, so a table
// is searched linearly until it has more than symbolTableLinearMax entries, and only then gets a hash index of entry
// positions.
//
// An id means a symbol through the table's store. In a binder store's table it is the symbol's index in that store,
// and a symbol of any other store is kept aside as a view, with its position there as the id. In a checker store's
// table it is the symbol's id in the checker, which names any symbol. A table takes the store of the first symbol
// added unless made for one, and moves to a checker's store when it is given one of the checker's symbols.
//
// A table must not be changed with Set or Delete while it is being iterated.
type SymbolTable = *SymbolMap

const symbolTableLinearMax = 8

// foreignSymbol marks an id that is a position in a table's foreign symbols.
const foreignSymbol = 1 << 31

type SymbolMap struct {
	store   *SymbolStore
	entries []symbolEntry
	rare    *symbolMapRare
}

type symbolMapRare struct {
	index   []int32 // open addressing table of entry positions plus one (0 is empty), only past symbolTableLinearMax
	foreign []Symbol
}

func (m *SymbolMap) ensureRare() *symbolMapRare {
	if m.rare == nil {
		m.rare = &symbolMapRare{}
	}
	return m.rare
}

func (m *SymbolMap) index() []int32 {
	if m.rare == nil {
		return nil
	}
	return m.rare.index
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

func NewSymbolTable() SymbolTable {
	return NewSymbolTableWithCapacity(0)
}

func NewSymbolTableWithCapacity(capacity int) SymbolTable {
	return newSymbolTable(nil, capacity)
}

// NewTable returns a table whose ids are those of st.
func (st *SymbolStore) NewTable(capacity int) SymbolTable {
	return newSymbolTable(st, capacity)
}

// TableOf returns a table of st holding symbol under name.
func (st *SymbolStore) TableOf(name SymbolName, symbol Symbol) SymbolTable {
	m := newSymbolTable(st, 1)
	m.Set(name, symbol)
	return m
}

func newSymbolTable(st *SymbolStore, capacity int) SymbolTable {
	m := &SymbolMap{store: st}
	if capacity > 0 {
		m.entries = make([]symbolEntry, 0, capacity)
		if capacity > symbolTableLinearMax {
			m.ensureRare().index = make([]int32, indexSizeFor(capacity))
		}
	}
	return m
}

func SymbolTableOf(name SymbolName, symbol Symbol) SymbolTable {
	m := NewSymbolTableWithCapacity(1)
	m.Set(name, symbol)
	return m
}

func (m *SymbolMap) KeyAt(i int) SymbolName {
	return m.entries[i].name()
}

func (m *SymbolMap) SymbolAt(i int) Symbol {
	return m.symbolOf(m.entries[i].id)
}

func (m *SymbolMap) symbolOf(id uint32) Symbol {
	st := m.store
	if st == nil {
		return Symbol{}
	}
	if st.checker != nil {
		return st.SymbolOf(SymbolId(id))
	}
	if id&foreignSymbol != 0 {
		return m.rare.foreign[id&^foreignSymbol]
	}
	if id == 0 {
		return Symbol{}
	}
	return Symbol{st, id}
}

func (m *SymbolMap) idOf(symbol Symbol) uint32 {
	if symbol.s == nil {
		return 0
	}
	if m.store == nil {
		m.store = symbol.s
	}
	st := m.store
	if st.checker == nil && symbol.s.checker != nil {
		m.moveTo(symbol.s)
		st = m.store
	}
	if st.checker != nil {
		return uint32(st.IdOf(symbol))
	}
	if symbol.s == st {
		return symbol.i
	}
	r := m.ensureRare()
	r.foreign = append(r.foreign, symbol)
	return uint32(len(r.foreign)-1) | foreignSymbol
}

// moveTo renumbers the entries of a binder store's table with the ids of a
// checker's store.
func (m *SymbolMap) moveTo(st *SymbolStore) {
	for i := range m.entries {
		e := &m.entries[i]
		e.id = uint32(st.IdOf(m.symbolOf(e.id)))
	}
	m.store = st
	if m.rare != nil {
		m.rare.foreign = nil
	}
}

func (m *SymbolMap) Find(key SymbolName) int {
	if m == nil {
		return -1
	}
	lo, hi := uint32(key), uint32(key>>32)
	if index := m.index(); len(index) != 0 {
		mask := uint64(len(index) - 1)
		for slot := hashSymbolTableKey(key) & mask; ; slot = (slot + 1) & mask {
			p := index[slot]
			if p == 0 {
				return -1
			}
			if e := &m.entries[p-1]; e.lo == lo && e.hi == hi {
				return int(p - 1)
			}
		}
	}
	for i := range m.entries {
		if e := &m.entries[i]; e.lo == lo && e.hi == hi {
			return i
		}
	}
	return -1
}

func (m *SymbolMap) Len() int {
	if m == nil {
		return 0
	}
	return len(m.entries)
}

func (m *SymbolMap) Get(key SymbolName) Symbol {
	if i := m.Find(key); i >= 0 {
		return m.SymbolAt(i)
	}
	return Symbol{}
}

func (m *SymbolMap) Lookup(key SymbolName) (Symbol, bool) {
	if i := m.Find(key); i >= 0 {
		return m.SymbolAt(i), true
	}
	return Symbol{}, false
}

func (m *SymbolMap) Set(key SymbolName, symbol Symbol) {
	m.setOrAdd(key, symbol, true)
}

func (m *SymbolMap) AddIfAbsent(key SymbolName, symbol Symbol) bool {
	return m.setOrAdd(key, symbol, false)
}

func (m *SymbolMap) setOrAdd(key SymbolName, symbol Symbol, replace bool) bool {
	if i := m.Find(key); i >= 0 {
		if replace {
			m.entries[i].id = m.idOf(symbol)
		}
		return replace
	}
	i := len(m.entries)
	if cap(m.entries) == 0 {
		m.entries = make([]symbolEntry, 0, 2)
	}
	id := m.idOf(symbol)
	m.entries = append(m.entries, symbolEntry{uint32(key), uint32(key >> 32), id})
	m.grewTo(key, i)
	return true
}

// grewTo indexes the entry for key just added at position i.
func (m *SymbolMap) grewTo(key SymbolName, i int) {
	if index := m.index(); len(index) != 0 && len(m.entries)*4 <= len(index)*3 {
		m.insertIndex(key, i)
	} else if len(m.entries) > symbolTableLinearMax {
		m.buildIndex()
	}
}

func indexSizeFor(n int) int {
	size := 16
	for size*3 < n*4 {
		size *= 2
	}
	return size
}

func (m *SymbolMap) insertIndex(key SymbolName, i int) {
	index := m.rare.index
	mask := uint64(len(index) - 1)
	slot := hashSymbolTableKey(key) & mask
	for index[slot] != 0 {
		slot = (slot + 1) & mask
	}
	index[slot] = int32(i + 1)
}

func (m *SymbolMap) buildIndex() {
	indexLen := indexSizeFor(len(m.entries))
	if r := m.ensureRare(); len(r.index) < indexLen {
		r.index = make([]int32, indexLen)
	}
	for i := range m.entries {
		m.insertIndex(m.KeyAt(i), i)
	}
}

func (m *SymbolMap) Delete(key SymbolName) {
	d := m.Find(key)
	if d < 0 {
		return
	}
	m.entries = slices.Delete(m.entries, d, d+1)
	if index := m.index(); len(index) != 0 {
		clear(index)
		m.buildIndex()
	}
}

func (m *SymbolMap) All() iter.Seq2[SymbolName, Symbol] {
	return func(yield func(SymbolName, Symbol) bool) {
		if m == nil {
			return
		}
		entries := m.entries
		for i := range entries {
			if !yield(entries[i].name(), m.symbolOf(entries[i].id)) {
				return
			}
		}
	}
}

func (m *SymbolMap) Keys() iter.Seq[SymbolName] {
	return func(yield func(SymbolName) bool) {
		if m == nil {
			return
		}
		entries := m.entries
		for i := range entries {
			if !yield(entries[i].name()) {
				return
			}
		}
	}
}

func (m *SymbolMap) Values() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		if m == nil {
			return
		}
		entries := m.entries
		for i := range entries {
			if !yield(m.symbolOf(entries[i].id)) {
				return
			}
		}
	}
}

func (m *SymbolMap) Clone() SymbolTable {
	if m == nil {
		return nil
	}
	clone := &SymbolMap{store: m.store}
	clone.entries = slices.Clone(m.entries)
	if m.rare != nil && m.rare.foreign != nil {
		clone.ensureRare().foreign = slices.Clone(m.rare.foreign)
	}
	if len(m.entries) > symbolTableLinearMax {
		clone.buildIndex()
	}
	return clone
}

func (m *SymbolMap) Clear() {
	if m != nil && len(m.entries) != 0 {
		m.entries = m.entries[:0]
		if m.rare != nil {
			clear(m.rare.index)
			m.rare.foreign = nil
		}
	}
}

const InternalSymbolNamePrefix = "\xFE" // Invalid UTF8 sequence, will never occur as IdentifierName

// SymbolNameText returns a display name, preserving private identifier spelling.
func SymbolNameText(symbol Symbol) string {
	if !symbol.ValueDeclaration().IsNil() && IsPrivateIdentifierClassElementDeclaration(symbol.ValueDeclaration()) {
		return symbol.ValueDeclaration().Name().Text()
	}
	return symbol.Name().String()
}

// EscapeAllInternalSymbolNames replaces internal symbol name markers ("\xFE") with "__".
func EscapeAllInternalSymbolNames(name string) string {
	return strings.ReplaceAll(name, InternalSymbolNamePrefix, "__")
}

func EscapeInternalSymbolName(symbolName SymbolName) string {
	name := symbolName.String()
	if rest, ok := strings.CutPrefix(name, InternalSymbolNamePrefix); ok {
		return "__" + rest
	}
	return name
}

// EscapeSymbolName converts a binder symbol name into its escaped "__String"
// form. Internal names (prefixed with the "\xFE" sentinel) become "__"-prefixed,
// and user names that already begin with "__" gain an extra leading underscore
// so they can be distinguished from internal names.
func EscapeSymbolName(symbolName SymbolName) string {
	name := symbolName.String()
	if rest, ok := strings.CutPrefix(name, InternalSymbolNamePrefix); ok {
		return "__" + rest
	}
	if len(name) >= 2 && name[0] == '_' && name[1] == '_' {
		return "_" + name
	}
	return name
}

// UnescapeSymbolName reverses EscapeSymbolName and interns the decoded name.
func UnescapeSymbolName(name string) SymbolName {
	if strings.HasPrefix(name, "___") {
		name = name[1:]
	} else if rest, ok := strings.CutPrefix(name, "__"); ok {
		name = InternalSymbolNamePrefix + rest
	}
	return MakeSymbolName(name)
}

// The methods below read and write a table in the ids of a checker's store
// st. A table that is not yet st's, or a binder store's, is read through its
// own store; writing an id moves it to st.

// checkerId returns the id in st of the symbol id names in m.
func (m *SymbolMap) checkerId(st *SymbolStore, id uint32) SymbolId {
	s := m.store
	switch {
	case s == st:
		return SymbolId(id)
	case s == nil || id == 0:
		return 0
	case s.checker == nil && id&foreignSymbol == 0:
		if base, ok := st.BaseOf(s); ok {
			return SymbolId(base + id)
		}
	}
	return st.IdOf(m.symbolOf(id))
}

// GetId searches and converts in one call, since it is the lookup the
// checker makes most and Find with checkerId would be two more calls.
func (m *SymbolMap) GetId(st *SymbolStore, key SymbolName) SymbolId {
	id, _ := m.LookupId(st, key)
	return id
}

func (m *SymbolMap) LookupId(st *SymbolStore, key SymbolName) (SymbolId, bool) {
	if m == nil {
		return 0, false
	}
	lo, hi := uint32(key), uint32(key>>32)
	var e *symbolEntry
	if m.rare != nil && len(m.rare.index) != 0 {
		index := m.rare.index
		mask := uint64(len(index) - 1)
		for slot := hashSymbolTableKey(key) & mask; ; slot = (slot + 1) & mask {
			p := index[slot]
			if p == 0 {
				return 0, false
			}
			if e = &m.entries[p-1]; e.lo == lo && e.hi == hi {
				break
			}
		}
	} else {
		i := 0
		for ; i < len(m.entries); i++ {
			if e = &m.entries[i]; e.lo == lo && e.hi == hi {
				break
			}
		}
		if i == len(m.entries) {
			return 0, false
		}
	}
	if m.store == st {
		return SymbolId(e.id), true
	}
	return m.checkerId(st, e.id), true
}

func (m *SymbolMap) IdAt(st *SymbolStore, i int) SymbolId {
	return m.checkerId(st, m.entries[i].id)
}

func (m *SymbolMap) AllIds(st *SymbolStore) iter.Seq2[SymbolName, SymbolId] {
	return func(yield func(SymbolName, SymbolId) bool) {
		if m == nil {
			return
		}
		entries := m.entries
		for i := range entries {
			if !yield(entries[i].name(), m.checkerId(st, entries[i].id)) {
				return
			}
		}
	}
}

func (m *SymbolMap) ValueIds(st *SymbolStore) iter.Seq[SymbolId] {
	return func(yield func(SymbolId) bool) {
		if m == nil {
			return
		}
		entries := m.entries
		for i := range entries {
			if !yield(m.checkerId(st, entries[i].id)) {
				return
			}
		}
	}
}

func (m *SymbolMap) SetId(st *SymbolStore, key SymbolName, id SymbolId) {
	m.setOrAddId(st, key, id, true)
}

func (m *SymbolMap) AddIfAbsentId(st *SymbolStore, key SymbolName, id SymbolId) bool {
	return m.setOrAddId(st, key, id, false)
}

func (m *SymbolMap) setOrAddId(st *SymbolStore, key SymbolName, id SymbolId, replace bool) bool {
	if m.store != st {
		if m.store == nil || m.store.checker == nil {
			if id == 0 {
				return m.setOrAdd(key, Symbol{}, replace)
			}
			if m.store != nil {
				m.moveTo(st)
			}
			m.store = st
		} else {
			return m.setOrAdd(key, st.SymbolOf(id), replace)
		}
	}
	if i := m.Find(key); i >= 0 {
		if replace {
			m.entries[i].id = uint32(id)
		}
		return replace
	}
	if cap(m.entries) == 0 {
		m.entries = make([]symbolEntry, 0, 2)
	}
	m.entries = append(m.entries, symbolEntry{uint32(key), uint32(key >> 32), uint32(id)})
	m.grewTo(key, len(m.entries)-1)
	return true
}

// TableOfId returns a table of a checker's store st holding the symbol id
// names under name.
func (st *SymbolStore) TableOfId(name SymbolName, id SymbolId) SymbolTable {
	m := newSymbolTable(st, 1)
	m.SetId(st, name, id)
	return m
}
