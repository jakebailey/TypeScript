package ast

import (
	"iter"
	"slices"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// A NodeList is a list of nodes: the store holding it and its index in the
// store's lists. The zero NodeList is the absent list, which is empty.
type NodeList struct {
	s    *NodeStore
	list uint32
}

// A ModifierList is a NodeList of modifiers.
type ModifierList struct {
	NodeList
}

// A listRef is a field referring to a list, encoded like a nodeRef, whose
// far lists are the store's farLists.
type listRef uint32

func (f *NodeFactory) NewNodeList(nodes []Node) NodeList {
	return f.nodeStore().newNodeList(nodes, core.UndefinedTextRange())
}

func (f *NodeFactory) NewModifierList(nodes []Node) ModifierList {
	return ModifierList{f.nodeStore().newNodeList(nodes, core.UndefinedTextRange())}
}

// NewNodeListAt returns a new list of nodes spanning loc.
func (f *NodeFactory) NewNodeListAt(loc core.TextRange, nodes []Node) NodeList {
	return f.nodeStore().newNodeList(nodes, loc)
}

// NewModifierListAt is like NewNodeListAt for modifiers.
func (f *NodeFactory) NewModifierListAt(loc core.TextRange, nodes []Node) ModifierList {
	return ModifierList{f.nodeStore().newNodeList(nodes, loc)}
}

// newNodeList returns a new list of nodes spanning loc.
func (s *NodeStore) newNodeList(nodes []Node, loc core.TextRange) NodeList {
	s.checkMutable()
	if len(s.lists) == 0 {
		// The first list is the absent one, so that no list ref is zero.
		s.setLists(make([]nodeList, 1, 8))
	}
	i := len(s.lists)
	if uint64(i) >= refFar {
		panic("node store list overflow")
	}
	start := len(s.listNodes)
	if uint64(start+len(nodes)) > 1<<32 {
		panic("node store list overflow")
	}
	if start+len(nodes) > cap(s.listNodes) {
		s.setListNodes(slices.Grow(s.listNodes, len(nodes)))
	}
	s.listNodes = s.listNodes[:start+len(nodes)]
	for k, node := range nodes {
		s.setRef((*nodeRef)(&s.listNodes[start+k]), node)
	}
	if i == cap(s.lists) {
		s.setLists(slices.Grow(s.lists, 1))
	}
	s.lists = append(s.lists, nodeList{loc: loc, start: uint32(start), len: uint32(len(nodes))})
	return NodeList{s, uint32(i)}
}

func (l NodeList) IsNil() bool { return l.list == 0 }

// nilList is the header of the absent list.
var nilList nodeList

func (l NodeList) hdr() *nodeList {
	if l.s == nil {
		return &nilList
	}
	// SAFETY: A list's index is within its store's lists.
	return (*nodeList)(unsafe.Add(unsafe.Pointer(l.s.listsBase.Load()), uintptr(l.list)*nodeListSize))
}

func (l NodeList) store() *NodeStore { return l.s }

// slot returns the ref of the element at index k of l, which must be in
// range.
func (l NodeList) slot(k int) *nodeRef {
	s := l.s
	// SAFETY: A list's index is within its store's lists.
	h := (*nodeList)(unsafe.Add(unsafe.Pointer(s.listsBase.Load()), uintptr(l.list)*nodeListSize))
	// SAFETY: A list's elements are within its store's listNodes.
	return (*nodeRef)(unsafe.Add(unsafe.Pointer(s.listNodesBase.Load()), (uintptr(h.start)+uintptr(k))*4))
}

func (l NodeList) Loc() core.TextRange { return l.hdr().loc }

func (l NodeList) SetLoc(loc core.TextRange) { l.hdr().loc = loc }

func (l NodeList) Pos() int { return l.Loc().Pos() }
func (l NodeList) End() int { return l.Loc().End() }

// Len returns the number of nodes in l, which is zero for the absent list.
func (l NodeList) Len() int {
	return int(l.hdr().len)
}

// At returns the node at index k of l.
func (l NodeList) At(k int) Node {
	if uint(k) >= uint(l.Len()) {
		panic("node list index out of range")
	}
	return l.at(k)
}

func (l NodeList) at(k int) Node {
	r := *l.slot(k)
	if r&refFar == 0 {
		if r == 0 {
			return Node{}
		}
		return Node{l.s, uint32(r)}
	}
	return l.store().far[r&^refFar]
}

// SetAt replaces the node at index k of l.
func (l NodeList) SetAt(k int, node Node) {
	if uint(k) >= uint(l.Len()) {
		panic("node list index out of range")
	}
	l.store().setRef(l.slot(k), node)
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
func (l NodeList) IsMissing() bool {
	if l.list == 0 {
		return false
	}
	_, ok := l.store().missing[l.list]
	return ok
}

func (l NodeList) SetMissing() {
	s := l.store()
	if s.missing == nil {
		s.missing = make(map[uint32]struct{})
	}
	s.missing[l.list] = struct{}{}
}

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

// ModifierFlags returns the combined flags of the modifiers in l.
func (l ModifierList) ModifierFlags() ModifierFlags {
	var flags ModifierFlags
	for k, n := 0, l.Len(); k < n; k++ {
		flags |= ModifierToFlag(l.at(k).Kind())
	}
	return flags
}

func (l ModifierList) Clone(f *NodeFactory) ModifierList {
	return ModifierList{l.NodeList.Clone(f)}
}

// derefList returns the list that the field f of n refers to.
func (n Node) derefList(f *listRef) NodeList {
	r := *f
	if r&refFar == 0 {
		if r == 0 {
			return NodeList{}
		}
		return NodeList{n.s, uint32(r)}
	}
	return n.store().farLists[r&^refFar]
}

func (n Node) derefModifiers(f *listRef) ModifierList {
	return ModifierList{n.derefList(f)}
}

// setListRef makes the field f of n refer to v.
func (n Node) setListRef(f *listRef, v NodeList) { setListRefAt(f, n, v) }

func (n Node) setModifiersRef(f *listRef, v ModifierList) { setListRefAt(f, n, v.NodeList) }

// setListRefAt makes the field f of owner refer to v.
func setListRefAt(f *listRef, owner Node, v NodeList) {
	switch {
	case v.s == owner.s:
		*f = listRef(v.list)
	case v.list == 0:
		*f = 0
	default:
		s := owner.store()
		if *f&refFar != 0 {
			s.farLists[*f&^refFar] = v
		} else {
			*f = listRef(farSlot(len(s.farLists)))
			s.farLists = append(s.farLists, v)
		}
	}
}

func setModifiersRefAt(f *listRef, owner Node, v ModifierList) {
	setListRefAt(f, owner, v.NodeList)
}

// listAt returns the list that the field at offset off-1 of n's fields
// refers to.
func (n Node) listAt(off uint8) NodeList {
	// SAFETY: The kind's table gives the offset of an aligned listRef.
	return n.derefList((*listRef)(unsafe.Add(n.fields(), off-1)))
}
