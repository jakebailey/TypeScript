package ast

import (
	"math/bits"
	"slices"
)

// The binder's fields of nodes are not in their records but in sparse side
// tables of their store, keyed by node index. Declarations' symbols and
// nodes' flow nodes, which many nodes have, are in bitmap-ranked maps. The
// fields of containers, which few nodes have, share one record per node in
// another bitmap-ranked map, since name resolution reads them on every
// enclosing node. Local symbols, which are rare and seldom read, are in a
// map of sorted node indices.
type binderData struct {
	symbols      sparseMap[uint32]
	flowNodes    sparseMap[flowRef]
	containers   sparseMap[containerData]
	localSymbols sortedMap[uint32]

	// foreignSymbols holds the symbols, and foreignLocalSymbols the local
	// symbols, of nodes whose symbol is not one of the store's own, which
	// symbols and localSymbols hold by index.
	foreignSymbols      map[uint32]Symbol
	foreignLocalSymbols map[uint32]Symbol

	// done is set once a parsed file's binding is finished and its sparse
	// maps are packed.
	done bool
}

// A sparseMap maps node indices to values. Packed, it has a block per 32
// nodes, holding a bit for each node having a value and the index in
// values of the first of the block's values; a node's value is found by
// counting the bits below its own. While a parsed file is being bound, its
// maps are dense, with a value for every node; binding a large file sets
// values for most of its nodes, in no useful order. Values set in a packed
// map for nodes having none are kept in extra, which is also all that a
// store of nodes not from a parsed file uses. A map that is never dense,
// set only through extra while binding, is packed from extra.
type sparseMap[T comparable] struct {
	blocks []sparseBlock
	values []T
	dense  []T
	extra  map[uint32]T
}

type sparseBlock struct {
	bits uint32
	base uint32
}

func (m *sparseMap[T]) get(i uint32) T {
	if b := i >> 5; b < uint32(len(m.blocks)) {
		block := m.blocks[b]
		bit := uint32(1) << (i & 31)
		if block.bits&bit != 0 {
			return m.values[block.base+uint32(bits.OnesCount32(block.bits&(bit-1)))]
		}
	}
	if i < uint32(len(m.dense)) {
		return m.dense[i]
	}
	if m.extra != nil {
		return m.extra[i]
	}
	var zero T
	return zero
}

// set sets i's value to v. If dense is set, the map is dense, with that
// many values.
func (m *sparseMap[T]) set(i uint32, v T, dense int) {
	if b := i >> 5; b < uint32(len(m.blocks)) {
		block := m.blocks[b]
		bit := uint32(1) << (i & 31)
		if block.bits&bit != 0 {
			m.values[block.base+uint32(bits.OnesCount32(block.bits&(bit-1)))] = v
			return
		}
	}
	var zero T
	if dense != 0 {
		if m.dense == nil {
			if v == zero {
				return
			}
			m.dense = make([]T, dense)
		}
		m.dense[i] = v
		return
	}
	if v == zero {
		delete(m.extra, i)
		return
	}
	if m.extra == nil {
		m.extra = make(map[uint32]T)
	}
	m.extra[i] = v
}

// pack turns a dense map into a packed one.
func (m *sparseMap[T]) pack() {
	if m.dense == nil {
		m.packExtra()
		return
	}
	var zero T
	count := 0
	last := -1
	for i, v := range m.dense {
		if v != zero {
			count++
			last = i
		}
	}
	if count != 0 {
		m.blocks = make([]sparseBlock, last>>5+1)
		m.values = make([]T, 0, count)
		for i, v := range m.dense[:last+1] {
			if v != zero {
				block := &m.blocks[i>>5]
				if block.bits == 0 {
					block.base = uint32(len(m.values))
				}
				block.bits |= 1 << (i & 31)
				m.values = append(m.values, v)
			}
		}
	}
	m.dense = nil
}

// packExtra turns a map set only through extra into a packed one.
func (m *sparseMap[T]) packExtra() {
	if len(m.extra) == 0 || m.blocks != nil {
		return
	}
	keys := make([]uint32, 0, len(m.extra))
	for i := range m.extra {
		keys = append(keys, i)
	}
	slices.Sort(keys)
	m.blocks = make([]sparseBlock, keys[len(keys)-1]>>5+1)
	m.values = make([]T, 0, len(keys))
	for _, i := range keys {
		block := &m.blocks[i>>5]
		if block.bits == 0 {
			block.base = uint32(len(m.values))
		}
		block.bits |= 1 << (i & 31)
		m.values = append(m.values, m.extra[i])
	}
	m.extra = nil
}

// containerData holds the binder's fields of a container, a function or a
// case clause.
type containerData struct {
	locals              uint32 // as the store's symbols keep tables; see tableIndex
	nextContainer       nodeRef
	endFlowNode         flowRef
	returnFlowNode      flowRef
	fallthroughFlowNode flowRef
}

// A sortedMap maps node indices to values, for fields few nodes have.
// Packed, it holds the indices sorted, with their values in the same order,
// found by binary search. Values are set in extra while a file is being
// bound, and set later for nodes having none, which is also all that a
// store of nodes not from a parsed file uses.
type sortedMap[T comparable] struct {
	keys   []uint32
	values []T
	extra  map[uint32]T
}

func (m *sortedMap[T]) get(i uint32) T {
	if j, ok := slices.BinarySearch(m.keys, i); ok {
		return m.values[j]
	}
	if m.extra != nil {
		return m.extra[i]
	}
	var zero T
	return zero
}

func (m *sortedMap[T]) set(i uint32, v T) {
	if j, ok := slices.BinarySearch(m.keys, i); ok {
		m.values[j] = v
		return
	}
	var zero T
	if v == zero {
		delete(m.extra, i)
		return
	}
	if m.extra == nil {
		m.extra = make(map[uint32]T)
	}
	m.extra[i] = v
}

// pack moves the values set while binding into the sorted arrays.
func (m *sortedMap[T]) pack() {
	if len(m.extra) == 0 || m.keys != nil {
		return
	}
	m.keys = make([]uint32, 0, len(m.extra))
	for i := range m.extra {
		m.keys = append(m.keys, i)
	}
	slices.Sort(m.keys)
	m.values = make([]T, len(m.keys))
	for j, i := range m.keys {
		m.values[j] = m.extra[i]
	}
	m.extra = nil
}

// finishBinding packs the binder's data of a parsed file. It must be
// called before the file is shared with other goroutines.
func (s *NodeStore) finishBinding() {
	b := &s.binder
	b.symbols.pack()
	b.flowNodes.pack()
	b.containers.pack()
	b.localSymbols.pack()
	s.symbols.finishBinding()
	b.done = true
	if len(s.flows) != cap(s.flows) {
		s.flows = slices.Clip(slices.Clone(s.flows))
	}
	if len(s.flowLists) != cap(s.flowLists) {
		s.flowLists = slices.Clip(slices.Clone(s.flowLists))
	}
	if len(s.reusedFlows) != cap(s.reusedFlows) {
		s.reusedFlows = slices.Clip(slices.Clone(s.reusedFlows))
	}
}

// denseBinderSize returns the size of the dense maps of s while it is
// being bound, and zero otherwise.
func (s *NodeStore) denseBinderSize() int {
	if s.file == nil || s.binder.done {
		return 0
	}
	return len(s.nodes)
}

func setBinderMap[T comparable](m *map[uint32]T, i uint32, v T) {
	var zero T
	if v == zero {
		delete(*m, i)
		return
	}
	if *m == nil {
		*m = make(map[uint32]T)
	}
	(*m)[i] = v
}

func (n Node) binderSymbol() Symbol {
	s := n.store()
	if i := s.binder.symbols.get(n.node); i != 0 {
		return Symbol{&s.symbols, i}
	}
	if s.binder.foreignSymbols != nil {
		return s.binder.foreignSymbols[n.node]
	}
	return Symbol{}
}

func (n Node) setBinderSymbol(v Symbol) {
	s := n.store()
	var i uint32
	if v.s == &s.symbols {
		i = v.i
	}
	s.binder.symbols.set(n.node, i, s.denseBinderSize())
	if i == 0 && (!v.IsNil() || s.binder.foreignSymbols != nil) {
		setBinderMap(&s.binder.foreignSymbols, n.node, v)
	}
}

func (n Node) binderFlowNode() FlowNode {
	s := n.store()
	return s.derefFlow(s.binder.flowNodes.get(n.node))
}

func (n Node) setBinderFlowNode(v FlowNode) {
	s := n.store()
	r := s.binder.flowNodes.get(n.node)
	s.setFlowRef(&r, v)
	s.binder.flowNodes.set(n.node, r, s.denseBinderSize())
}

func (n Node) binderLocalSymbol() Symbol {
	s := n.store()
	if i := s.binder.localSymbols.get(n.node); i != 0 {
		return Symbol{&s.symbols, i}
	}
	if s.binder.foreignLocalSymbols != nil {
		return s.binder.foreignLocalSymbols[n.node]
	}
	return Symbol{}
}

func (n Node) setBinderLocalSymbol(v Symbol) {
	s := n.store()
	var i uint32
	if v.s == &s.symbols {
		i = v.i
	}
	s.binder.localSymbols.set(n.node, i)
	if i == 0 && (!v.IsNil() || s.binder.foreignLocalSymbols != nil) {
		setBinderMap(&s.binder.foreignLocalSymbols, n.node, v)
	}
}

func (n Node) binderContainer() containerData {
	return n.store().binder.containers.get(n.node)
}

func (n Node) updateBinderContainer(update func(d *containerData)) {
	m := &n.store().binder.containers
	d := m.get(n.node)
	update(&d)
	m.set(n.node, d, 0)
}

func (n Node) binderLocals() SymbolTable {
	return n.store().symbols.tableAt(n.binderContainer().locals)
}

func (n Node) setBinderLocals(v SymbolTable) {
	x := n.store().Symbols().tableIndex(v)
	n.updateBinderContainer(func(d *containerData) { d.locals = x })
}

func (n Node) binderNextContainer() Node {
	r := n.binderContainer().nextContainer
	return n.deref(&r)
}

func (n Node) setBinderNextContainer(v Node) {
	n.updateBinderContainer(func(d *containerData) { setNodeRef(&d.nextContainer, n, v) })
}

func (n Node) binderEndFlowNode() FlowNode {
	return n.store().derefFlow(n.binderContainer().endFlowNode)
}

func (n Node) setBinderEndFlowNode(v FlowNode) {
	n.updateBinderContainer(func(d *containerData) { n.store().setFlowRef(&d.endFlowNode, v) })
}

func (n Node) binderReturnFlowNode() FlowNode {
	return n.store().derefFlow(n.binderContainer().returnFlowNode)
}

func (n Node) setBinderReturnFlowNode(v FlowNode) {
	n.updateBinderContainer(func(d *containerData) { n.store().setFlowRef(&d.returnFlowNode, v) })
}

func (n Node) binderFallthroughFlowNode() FlowNode {
	return n.store().derefFlow(n.binderContainer().fallthroughFlowNode)
}

func (n Node) setBinderFallthroughFlowNode(v FlowNode) {
	n.updateBinderContainer(func(d *containerData) { n.store().setFlowRef(&d.fallthroughFlowNode, v) })
}
