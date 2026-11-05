package ast

import (
	"iter"
	"math/bits"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// A NodeList is a handle to a list of nodes: the address of its record.
// The zero NodeList is the absent list.
//
// A list record is a listHeader, then the elements as nodeRefs, then
// those of the list's pos, end and aux that its header says it stores.
// A parsed list mostly spans exactly its elements, so it derives pos and
// end from them instead of storing them.
type NodeList struct {
	p unsafe.Pointer
}

type listHeader struct {
	// info holds the record's offset in bytes from the start of its chunk
	// in its low listChunkBits bits, and listInfo flags above them.
	info uint32
	n    uint32
}

const listHeaderSize = unsafe.Sizeof(listHeader{})

const listChunkBits = 21

// Every chunk offset fits in listChunkBits bits.
var _ [1<<listChunkBits - int(chunkHeaderSize) - maxChunkRecordWords*int(wordSize)]struct{}

const (
	listChunkMask = 1<<listChunkBits - 1

	// listPos and listEnd mark a list record that stores its pos or end;
	// otherwise the pos is that of its first element, and the end is
	// that of its last element, or its pos if listEndAtPos is set.
	listPos uint32 = 1 << (listChunkBits + iota)
	listEnd
	listEndAtPos
	// listAux marks a list record that stores its aux.
	listAux
	// listSpilled marks a list whose pos, end and aux are in its store's
	// spilledLists, because they changed in a way its record has no room
	// for.
	listSpilled
)

// spilledList holds the pos, end and aux of a list with listSpilled set.
type spilledList struct {
	loc core.TextRange
	aux uint32
}

// A ModifierList is a NodeList of modifiers, whose record also holds their
// combined flags.
type ModifierList struct {
	NodeList
}

// A listRef is a record field referring to a list, encoded like a
// nodeRef, whose far lists are the store's farLists.
type listRef uint32

// listMissing marks a list that was expected but missing from the source.
const listMissing = 1

func (f *NodeFactory) NewNodeList(nodes []Node) NodeList {
	return newNodeList(f.nodeStore(), nodes, 0, core.UndefinedTextRange(), false)
}

func (f *NodeFactory) NewModifierList(nodes []Node) ModifierList {
	return ModifierList{newNodeList(f.nodeStore(), nodes, uint32(ModifiersToFlags(nodes)), core.UndefinedTextRange(), false)}
}

// NewNodeListAt returns a new list of nodes spanning loc, whose nodes'
// locations must not change afterwards.
func (f *NodeFactory) NewNodeListAt(loc core.TextRange, nodes []Node) NodeList {
	return newNodeList(f.nodeStore(), nodes, 0, loc, true)
}

// NewModifierListAt is like NewNodeListAt for modifiers.
func (f *NodeFactory) NewModifierListAt(loc core.TextRange, nodes []Node) ModifierList {
	return ModifierList{newNodeList(f.nodeStore(), nodes, uint32(ModifiersToFlags(nodes)), loc, true)}
}

// newNodeList returns a new list of nodes spanning loc. If derive is set,
// the record omits a pos or end that it can derive from the nodes, whose
// locations therefore must not change; otherwise it stores both.
func newNodeList(s *nodeStore, nodes []Node, aux uint32, loc core.TextRange, derive bool) NodeList {
	n := len(nodes)
	var flags uint32
	pos, end := int32(loc.Pos()), int32(loc.End())
	switch {
	case !derive:
		flags = listPos | listEnd
	case n == 0:
		flags = listPos
		if end == pos {
			flags |= listEndAtPos
		} else {
			flags |= listEnd
		}
	default:
		if int(pos) != nodes[0].Pos() {
			flags |= listPos
		}
		if int(end) != nodes[n-1].End() {
			flags |= listEnd
		}
	}
	if aux != 0 {
		flags |= listAux
	}
	extra := bits.OnesCount32(flags & (listPos | listEnd | listAux))
	p, chunk := s.alloc(listHeaderSize + uintptr(n+extra)*unsafe.Sizeof(nodeRef(0)))
	l := NodeList{p}
	*l.hdr() = listHeader{info: chunk | flags, n: uint32(n)}
	c := l.chunk()
	for k, node := range nodes {
		setRef(c, l.slot(k), node)
	}
	k := n
	if flags&listPos != 0 {
		*l.slot(k) = nodeRef(pos)
		k++
	}
	if flags&listEnd != 0 {
		*l.slot(k) = nodeRef(end)
		k++
	}
	if flags&listAux != 0 {
		*l.slot(k) = nodeRef(aux)
	}
	return l
}

func (l NodeList) IsNil() bool { return l.p == nil }

func (l NodeList) hdr() *listHeader {
	// SAFETY: Every list record starts with an aligned listHeader.
	return (*listHeader)(l.p)
}

// slot returns the element ref at index k of l's record, or for k past
// the elements, the word holding the next stored pos, end or aux.
func (l NodeList) slot(k int) *nodeRef {
	// SAFETY: Callers bound k by the list length and stored words; newNodeList reserves that many aligned words after the header.
	return (*nodeRef)(unsafe.Add(l.p, listHeaderSize+uintptr(k)*unsafe.Sizeof(nodeRef(0))))
}

func (l NodeList) chunk() *chunkHeader { return chunkOf(l.p, l.hdr().info&listChunkMask) }

func (l NodeList) store() *nodeStore { return l.chunk().store }

func (l NodeList) Loc() core.TextRange {
	h := l.hdr()
	info := h.info
	if info&listSpilled != 0 {
		return l.store().spilledLists[l.p].loc
	}
	n := int(h.n)
	var pos, end int
	k := n
	if info&listPos != 0 {
		pos = int(int32(*l.slot(k)))
		k++
	} else {
		pos = l.at(0).Pos()
	}
	switch {
	case info&listEnd != 0:
		end = int(int32(*l.slot(k)))
	case info&listEndAtPos != 0:
		end = pos
	default:
		end = l.at(n - 1).End()
	}
	return core.NewTextRange(pos, end)
}

func (l NodeList) SetLoc(loc core.TextRange) {
	h := l.hdr()
	info := h.info
	if info&listSpilled == 0 && info&(listPos|listEnd) == listPos|listEnd {
		*l.slot(int(h.n)) = nodeRef(int32(loc.Pos()))
		*l.slot(int(h.n) + 1) = nodeRef(int32(loc.End()))
		return
	}
	if info&listSpilled == 0 && l.Loc() == loc {
		return
	}
	l.spill().loc = loc
}

// spill moves l's pos, end and aux to its store's spilledLists, if they
// are not there already, and returns them. It is for lists that change
// in a way their record has no room for, which is rare.
func (l NodeList) spill() *spilledList {
	s := l.store()
	if s.spilledLists == nil {
		s.spilledLists = make(map[unsafe.Pointer]*spilledList)
	}
	h := l.hdr()
	if h.info&listSpilled == 0 {
		s.spilledLists[l.p] = &spilledList{loc: l.Loc(), aux: l.aux()}
		h.info |= listSpilled
	}
	return s.spilledLists[l.p]
}

func (l NodeList) Pos() int { return l.Loc().Pos() }
func (l NodeList) End() int { return l.Loc().End() }

func (l NodeList) aux() uint32 {
	h := l.hdr()
	info := h.info
	switch {
	case info&listSpilled != 0:
		return l.store().spilledLists[l.p].aux
	case info&listAux == 0:
		return 0
	}
	return uint32(*l.slot(int(h.n) + bits.OnesCount32(info&(listPos|listEnd))))
}

func (l NodeList) setAux(v uint32) {
	h := l.hdr()
	info := h.info
	switch {
	case info&listSpilled == 0 && info&listAux != 0:
		*l.slot(int(h.n) + bits.OnesCount32(info&(listPos|listEnd))) = nodeRef(v)
	case info&listSpilled != 0 || v != 0:
		l.spill().aux = v
	}
}

// Len returns the number of nodes in l, which is zero for the absent list.
func (l NodeList) Len() int {
	if l.p == nil {
		return 0
	}
	return int(l.hdr().n)
}

// At returns the node at index k of l.
func (l NodeList) At(k int) Node {
	if uint(k) >= uint(l.Len()) {
		panic("node list index out of range")
	}
	return l.at(k)
}

func (l NodeList) at(k int) Node {
	f := l.slot(k)
	r := *f
	if r&refNear != 0 {
		// SAFETY: setRef encodes a near offset only to a node in the list's retained chunk.
		return Node{unsafe.Add(unsafe.Pointer(f), int(int32(r))-refNear)}
	}
	return l.store().far[r>>1]
}

// SetAt replaces the node at index k of l.
func (l NodeList) SetAt(k int, node Node) {
	if uint(k) >= uint(l.Len()) {
		panic("node list index out of range")
	}
	if info := l.hdr().info; info&listSpilled == 0 &&
		(k == 0 && info&listPos == 0 || k == l.Len()-1 && info&(listEnd|listEndAtPos) == 0) &&
		node.Loc() != l.at(k).Loc() {
		// The list derives its loc from the node being replaced.
		l.spill()
	}
	setRef(l.chunk(), l.slot(k), node)
}

// First returns the first node of l, or the absent node if l is empty.
func (l NodeList) First() Node {
	if l.Len() == 0 {
		return Node{}
	}
	return l.at(0)
}

// Last returns the last node of l, or the absent node if l is empty.
func (l NodeList) Last() Node {
	n := l.Len()
	if n == 0 {
		return Node{}
	}
	return l.at(n - 1)
}

// All returns an iterator over the indices and nodes of l.
func (l NodeList) All() iter.Seq2[int, Node] {
	return func(yield func(int, Node) bool) {
		n := l.Len()
		for k := range n {
			if !yield(k, l.at(k)) {
				return
			}
		}
	}
}

// Some reports whether f is true for any node of l.
func (l NodeList) Some(f func(Node) bool) bool {
	for k, n := 0, l.Len(); k < n; k++ {
		if f(l.at(k)) {
			return true
		}
	}
	return false
}

// Every reports whether f is true for every node of l.
func (l NodeList) Every(f func(Node) bool) bool {
	for k, n := 0, l.Len(); k < n; k++ {
		if !f(l.at(k)) {
			return false
		}
	}
	return true
}

// Slice returns the nodes of l in a new slice, or nil if l is empty.
func (l NodeList) Slice() []Node {
	n := l.Len()
	if n == 0 {
		return nil
	}
	nodes := make([]Node, n)
	for k := range nodes {
		nodes[k] = l.at(k)
	}
	return nodes
}

// IsMissing reports whether l stands for a list missing from the source.
func (l NodeList) IsMissing() bool { return l.p != nil && l.aux()&listMissing != 0 }

func (l NodeList) SetMissing() { l.setAux(l.aux() | listMissing) }

func (l NodeList) HasTrailingComma() bool {
	n := l.Len()
	if n == 0 {
		return false
	}
	return l.at(n-1).End() < l.End()
}

func (l NodeList) Clone(f NodeFactoryCoercible) NodeList {
	result := f.AsNodeFactory().NewNodeList(l.Slice())
	result.SetLoc(l.Loc())
	return result
}

func (l ModifierList) ModifierFlags() ModifierFlags {
	if l.p == nil {
		return ModifierFlagsNone
	}
	return ModifierFlags(l.aux())
}

func (l ModifierList) SetModifierFlags(flags ModifierFlags) { l.setAux(uint32(flags)) }

func (l ModifierList) Clone(f *NodeFactory) ModifierList {
	res := ModifierList{newNodeList(f.nodeStore(), l.Slice(), uint32(l.ModifierFlags()), core.UndefinedTextRange(), false)}
	res.SetLoc(l.Loc())
	return res
}

// derefList returns the list that the field f of n's record refers to.
func (n Node) derefList(f *listRef) NodeList {
	r := *f
	if r&refNear != 0 {
		// SAFETY: setListRef encodes a near offset only to a list in the same retained chunk.
		return NodeList{unsafe.Add(unsafe.Pointer(f), int32(r-refNear))}
	}
	if r == 0 {
		// Absent fields are common, and checking for them is cheaper than
		// reading the absent far slot, which is far from the record.
		return NodeList{}
	}
	// SAFETY: The node header's chunk offset leads back to its containing allocation's header.
	return (*chunkHeader)(unsafe.Add(n.p, -int((*nodeBaseData)(n.p).chunk))).store.farLists[r>>1]
}

func (n Node) derefModifiers(f *listRef) ModifierList {
	return ModifierList{n.derefList(f)}
}

// setListRef makes the field f of n's record refer to v.
func (n Node) setListRef(f *listRef, v NodeList) { setListRef(n.chunk(), f, v) }

func (n Node) setModifiersRef(f *listRef, v ModifierList) { setListRef(n.chunk(), f, v.NodeList) }

// setListRefAt makes the field f of the record of owner refer to v.
func setListRefAt(f *listRef, owner Node, v NodeList) { setListRef(owner.chunk(), f, v) }

func setModifiersRefAt(f *listRef, owner Node, v ModifierList) {
	setListRef(owner.chunk(), f, v.NodeList)
}

// setListRef makes the field f of a record in chunk c refer to v.
func setListRef(c *chunkHeader, f *listRef, v NodeList) {
	if v.p == nil {
		*f = 0
		return
	}
	if v.chunk() == c {
		*f = listRef(near(f, v.p))
		return
	}
	s := c.store
	if *f != 0 && *f&refNear == 0 {
		s.farLists[*f>>1] = v
		return
	}
	*f = listRef(farSlot(len(s.farLists)))
	s.farLists = append(s.farLists, v)
}

// listAt returns the list that the field at offset off of n's record
// refers to.
func (n Node) listAt(off uint8) NodeList {
	// SAFETY: The kind's layout table identifies an aligned listRef within this record.
	return n.derefList((*listRef)(unsafe.Add(n.p, off)))
}
