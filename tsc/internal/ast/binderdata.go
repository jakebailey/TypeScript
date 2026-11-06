package ast

import (
	"math/bits"
	"slices"
)

// The binder's fields of nodes are not in their records but in side tables
// of their store, keyed by node index. Declarations' symbols and nodes'
// flow nodes, which many nodes have, are in sparse maps; the other fields,
// which few nodes have, are in hash maps.
type binderData struct {
	symbols   sparseMap[*Symbol]
	flowNodes sparseMap[flowRef]

	localSymbols         map[uint32]*Symbol
	locals               map[uint32]SymbolTable
	nextContainers       map[uint32]nodeRef
	endFlowNodes         map[uint32]flowRef
	returnFlowNodes      map[uint32]flowRef
	fallthroughFlowNodes map[uint32]flowRef

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
// store of nodes not from a parsed file uses.
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

// finishBinding packs the binder's data of a parsed file. It must be
// called before the file is shared with other goroutines.
func (s *NodeStore) finishBinding() {
	b := &s.binder
	b.symbols.pack()
	b.flowNodes.pack()
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

func (n Node) binderSymbol() *Symbol {
	return n.store().binder.symbols.get(n.node)
}

func (n Node) setBinderSymbol(v *Symbol) {
	s := n.store()
	s.binder.symbols.set(n.node, v, s.denseBinderSize())
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

func (n Node) binderLocalSymbol() *Symbol {
	return n.store().binder.localSymbols[n.node]
}

func (n Node) setBinderLocalSymbol(v *Symbol) {
	setBinderMap(&n.store().binder.localSymbols, n.node, v)
}

func (n Node) binderLocals() SymbolTable {
	return n.store().binder.locals[n.node]
}

func (n Node) setBinderLocals(v SymbolTable) {
	setBinderMap(&n.store().binder.locals, n.node, v)
}

func (n Node) binderNextContainer() Node {
	r := n.store().binder.nextContainers[n.node]
	return n.deref(&r)
}

func (n Node) setBinderNextContainer(v Node) {
	m := &n.store().binder.nextContainers
	r := (*m)[n.node]
	setNodeRef(&r, n, v)
	setBinderMap(m, n.node, r)
}

func (n Node) binderFlowMap(m map[uint32]flowRef) FlowNode {
	return n.store().derefFlow(m[n.node])
}

func (n Node) setBinderFlowMap(m *map[uint32]flowRef, v FlowNode) {
	r := (*m)[n.node]
	n.store().setFlowRef(&r, v)
	setBinderMap(m, n.node, r)
}

func (n Node) binderEndFlowNode() FlowNode {
	return n.binderFlowMap(n.store().binder.endFlowNodes)
}

func (n Node) setBinderEndFlowNode(v FlowNode) {
	n.setBinderFlowMap(&n.store().binder.endFlowNodes, v)
}

func (n Node) binderReturnFlowNode() FlowNode {
	return n.binderFlowMap(n.store().binder.returnFlowNodes)
}

func (n Node) setBinderReturnFlowNode(v FlowNode) {
	n.setBinderFlowMap(&n.store().binder.returnFlowNodes, v)
}

func (n Node) binderFallthroughFlowNode() FlowNode {
	return n.binderFlowMap(n.store().binder.fallthroughFlowNodes)
}

func (n Node) setBinderFallthroughFlowNode(v FlowNode) {
	n.setBinderFlowMap(&n.store().binder.fallthroughFlowNodes, v)
}
