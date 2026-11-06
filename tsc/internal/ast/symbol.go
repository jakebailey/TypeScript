package ast

import (
	"errors"
	"iter"
	"maps"
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
// The entries are kept in insertion order in one slice. Most tables are small, so a table is searched linearly until
// it has more than symbolTableLinearMax entries, and only then gets a hash index of entry positions. An entry does not
// store its key as the key is almost always the symbol's own name. In the rare cases when a symbol has a different name,
// the table stores a proxy symbol with the new name and a reference to the original symbol in the parent field.
//
// A table must not be changed with Set or Delete while it is being iterated.
type SymbolTable = *SymbolMap

const symbolTableLinearMax = 16

type SymbolMap struct {
	entries []Symbol
	// names holds the keys of the entries whose key is not their symbol's
	// name, by position.
	names map[int32]SymbolName
	index []int32 // open addressing table of entry positions plus one (0 is empty), only past symbolTableLinearMax
}

func hashSymbolTableKey(name SymbolName) uint64 {
	h := uint64(name) * 0x9e3779b97f4a7c15
	return h ^ h>>32
}

func NewSymbolTable() SymbolTable {
	return NewSymbolTableWithCapacity(0)
}

func NewSymbolTableWithCapacity(capacity int) SymbolTable {
	m := &SymbolMap{}
	if capacity > 0 {
		m.entries = make([]Symbol, 0, capacity)
		if capacity > symbolTableLinearMax {
			m.index = make([]int32, indexSizeFor(capacity))
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
	if m.names != nil {
		if name, ok := m.names[int32(i)]; ok {
			return name
		}
	}
	return m.entries[i].Name()
}

func (m *SymbolMap) SymbolAt(i int) Symbol {
	return m.entries[i]
}

func (m *SymbolMap) Find(key SymbolName) int {
	if m == nil {
		return -1
	}
	if len(m.index) != 0 {
		mask := uint64(len(m.index) - 1)
		for slot := hashSymbolTableKey(key) & mask; ; slot = (slot + 1) & mask {
			p := m.index[slot]
			if p == 0 {
				return -1
			}
			if i := int(p - 1); m.KeyAt(i) == key {
				return i
			}
		}
	}
	for i := range m.entries {
		if m.KeyAt(i) == key {
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
			m.setEntry(i, key, symbol)
		}
		return replace
	}
	i := len(m.entries)
	if cap(m.entries) == 0 {
		m.entries = make([]Symbol, 0, 2)
	}
	m.entries = append(m.entries, Symbol{})
	m.setEntry(i, key, symbol)
	if len(m.index) != 0 && len(m.entries)*4 <= len(m.index)*3 {
		m.insertIndex(key, i)
	} else if len(m.entries) > symbolTableLinearMax {
		m.buildIndex()
	}
	return true
}

func (m *SymbolMap) setEntry(i int, key SymbolName, symbol Symbol) {
	m.entries[i] = symbol
	if key != symbol.Name() {
		if m.names == nil {
			m.names = make(map[int32]SymbolName)
		}
		m.names[int32(i)] = key
	} else if m.names != nil {
		delete(m.names, int32(i))
	}
}

func indexSizeFor(n int) int {
	size := 32
	for size*3 < n*4 {
		size *= 2
	}
	return size
}

func (m *SymbolMap) insertIndex(key SymbolName, i int) {
	mask := uint64(len(m.index) - 1)
	slot := hashSymbolTableKey(key) & mask
	for m.index[slot] != 0 {
		slot = (slot + 1) & mask
	}
	m.index[slot] = int32(i + 1)
}

func (m *SymbolMap) buildIndex() {
	indexLen := indexSizeFor(len(m.entries))
	if len(m.index) < indexLen {
		m.index = make([]int32, indexLen)
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
	if m.names != nil {
		names := m.names
		m.names = nil
		for i, name := range names {
			switch {
			case int(i) < d:
				m.names = setName(m.names, i, name)
			case int(i) > d:
				m.names = setName(m.names, i-1, name)
			}
		}
	}
	m.entries = slices.Delete(m.entries, d, d+1)
	if len(m.index) != 0 {
		clear(m.index)
		m.buildIndex()
	}
}

func setName(names map[int32]SymbolName, i int32, name SymbolName) map[int32]SymbolName {
	if names == nil {
		names = make(map[int32]SymbolName)
	}
	names[i] = name
	return names
}

func (m *SymbolMap) All() iter.Seq2[SymbolName, Symbol] {
	return func(yield func(SymbolName, Symbol) bool) {
		if m == nil {
			return
		}
		entries := m.entries
		for i := range entries {
			if !yield(m.KeyAt(i), m.SymbolAt(i)) {
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
			if !yield(m.KeyAt(i)) {
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
			if !yield(m.SymbolAt(i)) {
				return
			}
		}
	}
}

func (m *SymbolMap) Clone() SymbolTable {
	if m == nil {
		return nil
	}
	clone := &SymbolMap{}
	clone.entries = slices.Clone(m.entries)
	if m.names != nil {
		clone.names = maps.Clone(m.names)
	}
	if len(m.entries) > symbolTableLinearMax {
		clone.buildIndex()
	}
	return clone
}

func (m *SymbolMap) Clear() {
	if m != nil && len(m.entries) != 0 {
		clear(m.entries)
		clear(m.index)
		m.entries = m.entries[:0]
		m.names = nil
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
