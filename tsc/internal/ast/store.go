package ast

import (
	"slices"
	"sync/atomic"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// A NodeStore holds the nodes created by one NodeFactory: for a parsed
// file, every node of that file. A node is its index in the store's table
// of fixed-size headers, nodes. A header holds the node's kind, flags,
// location and parent, and a payload word: the kind's fields themselves
// if they fit in four bytes, and otherwise the offset of the words in data
// where they start. A list is its index in lists, whose elements are a
// span of listNodes.
//
// The tables hold no Go pointers, so the garbage collector never scans
// them. Fields refer to nodes and lists of the same store by index; a
// field holding a Go value refers to a slot in one of the store's side
// tables, and a field referring to a node or list of another store, such
// as an original node reused by a transformed tree, refers to a slot in
// far or farLists.
//
// A store is a plain Go value: a Node holds a pointer to its store, so the
// store lives as long as any of its nodes, or anything else, refers to it.
//
// The tables grow by copying. The store publishes their current addresses
// atomically, since a few stores are read by other goroutines while they
// grow (a source file's token cache), so that a Node, which is the store
// and an index, finds its header with two dependent loads.
type NodeStore struct {
	nodesBase     atomic.Pointer[nodeHeader]
	dataBase      atomic.Pointer[uint32]
	listsBase     atomic.Pointer[nodeList]
	listNodesBase atomic.Pointer[uint32]

	// serial numbers the store for GetNodeId, once it is asked; zero until
	// then. Nothing maps a serial back to its store.
	serial atomic.Uint32

	nodes     []nodeHeader
	data      []uint32
	lists     []nodeList
	listNodes []uint32

	texts       textStore
	stringLists sideTable[[]string]
	nodeSlices  sideTable[[]Node]
	values      sideTable[any]
	sourceFiles sideTable[*SourceFile]

	// file is the source file whose tree the store holds, if it holds a
	// parsed file. Every node of the store that is in that tree is in the
	// file.
	file *SourceFile

	// far holds the nodes of other stores that fields refer to, and
	// farLists the lists. They keep those stores alive.
	far      []Node
	farLists []NodeList

	// names holds the symbol names that records cannot hold inline; see
	// nameRef.
	names []SymbolName

	// missing holds the indices of the lists that stand for lists missing
	// from the source.
	missing map[uint32]struct{}

	binder binderData

	// facts caches the subtree facts of nodes, by index; see factsCache.
	facts atomic.Pointer[[]atomic.Uint32]

	flows       []flowNodeData
	flowLists   []flowListData
	reusedFlows []FlowNode

	// frozen is set once the nodes are shared, after which no more may be
	// created.
	frozen bool
}

// A nodeHeader is the fixed part of every node.
type nodeHeader struct {
	kind   Kind
	_      uint16
	flags  NodeFlags
	loc    core.TextRange
	parent nodeRef
	// word holds the node's fields if its kind's take at most four bytes;
	// otherwise, it is the index in the store's data of the first word of
	// its fields.
	word uint32
}

const nodeHeaderSize = unsafe.Sizeof(nodeHeader{})

var (
	_ [nodeHeaderSize - 24]struct{}
	_ [24 - nodeHeaderSize]struct{}
)

// A nodeList is a list of nodes: a span of its store's listNodes.
type nodeList struct {
	loc   core.TextRange
	start uint32
	len   uint32
}

const nodeListSize = unsafe.Sizeof(nodeList{})

var (
	_ [nodeListSize - 16]struct{}
	_ [16 - nodeListSize]struct{}
)

// A nodeRef is a field referring to a node. Zero is no node. With refFar
// clear, it is the node's index in the store of the field; otherwise, the
// rest is an index into that store's far nodes. A node's parent and the
// elements of lists are nodeRefs too.
type nodeRef uint32

const refFar = 1 << 31

// Index returns n's index in its store, which is never zero.
func (n Node) Index() uint32 { return n.node }

// Store returns the store holding n.
func (n Node) Store() *NodeStore { return n.s }

// NewNodeStore returns a new, empty store.
func NewNodeStore() *NodeStore {
	s := &NodeStore{
		stringLists: newSideTable[[]string](),
		nodeSlices:  newSideTable[[]Node](),
		values:      newSideTable[any](),
		sourceFiles: newSideTable[*SourceFile](),
	}
	s.setNodes(make([]nodeHeader, 1, 8))
	return s
}

var lastStoreSerial atomic.Uint32

// serialNumber returns s's serial, giving it one if it has none.
func (s *NodeStore) serialNumber() uint32 {
	if v := s.serial.Load(); v != 0 {
		return v
	}
	v := lastStoreSerial.Add(1)
	if v == 0 {
		panic("too many node stores")
	}
	if !s.serial.CompareAndSwap(0, v) {
		return s.serial.Load()
	}
	return v
}

// setNodes replaces s's nodes, publishing the new ones first, so that a
// reader that loaded the old ones still holds them.
func (s *NodeStore) setNodes(nodes []nodeHeader) {
	// SAFETY: The base points at the first element of the table, which it
	// keeps alive; indexes into it are below the table's length.
	s.nodesBase.Store(unsafe.SliceData(nodes))
	s.nodes = nodes
}

func (s *NodeStore) setData(data []uint32) {
	// SAFETY: The base points at the first element of the table, which it
	// keeps alive; indexes into it are below the table's length.
	s.dataBase.Store(unsafe.SliceData(data))
	s.data = data
}

func (s *NodeStore) setLists(lists []nodeList) {
	// SAFETY: The base points at the first element of the table, which it
	// keeps alive; indexes into it are below the table's length.
	s.listsBase.Store(unsafe.SliceData(lists))
	s.lists = lists
}

func (s *NodeStore) setListNodes(listNodes []uint32) {
	// SAFETY: The base points at the first element of the table, which it
	// keeps alive; indexes into it are below the table's length.
	s.listNodesBase.Store(unsafe.SliceData(listNodes))
	s.listNodes = listNodes
}

func (s *NodeStore) checkMutable() {
	if s.frozen {
		panic("node store is frozen")
	}
}

// addNode adds a zero node and returns its index.
func (s *NodeStore) addNode() uint32 {
	s.checkMutable()
	i := len(s.nodes)
	if i == cap(s.nodes) {
		if uint64(i) >= refFar {
			panic("node store overflow")
		}
		s.setNodes(slices.Grow(s.nodes, 1))
	}
	s.nodes = s.nodes[:i+1]
	s.nodes[i] = nodeHeader{}
	return uint32(i)
}

// addData adds n zero words to s's data and returns the index of the first.
func (s *NodeStore) addData(n int) uint32 {
	off := len(s.data)
	if off+n > cap(s.data) {
		if uint64(off+n) > 1<<32 {
			panic("node store data overflow")
		}
		s.setData(slices.Grow(s.data, n))
	}
	s.data = s.data[:off+n]
	clear(s.data[off:])
	return uint32(off)
}

func (s *NodeStore) newNode(kind Kind, layout nodeLayout, n Node) Node {
	if kindLayout[kind] != layout {
		panic("node kind does not match its record layout")
	}
	s.nodes[n.node].kind = kind
	return n
}

// newRecord adds a node whose fields are a zero T, returning them and the
// node. The fields stay where they are only until the store's next node.
func newRecord[T any](s *NodeStore, layout nodeLayout) (*T, Node) {
	size := unsafe.Sizeof(*new(T))
	if unsafe.Alignof(*new(T)) > 4 || layoutInline[layout] != (size <= 4) {
		panic("node record does not fit its layout")
	}
	i := s.addNode()
	n := Node{s, i}
	if size <= 4 {
		// SAFETY: T fits in the aligned header word, and holds no pointers.
		return (*T)(unsafe.Pointer(&s.nodes[i].word)), n
	}
	off := s.addData(int(size+3) / 4)
	s.nodes[i].word = off
	// SAFETY: addData reserved T's size in aligned words; T holds no pointers.
	return (*T)(unsafe.Pointer(&s.data[off])), n
}

// sizeHint is the source bytes per node that parsing typically yields;
// stores for parsing are sized from it, and trimmed when finished.
const sizeHint = 6

// sizeForSource sizes the store's tables for parsing its source text.
func (s *NodeStore) sizeForSource() {
	n := len(s.texts.source)/sizeHint + 16
	if len(s.nodes) > 1 {
		panic("node store already has records")
	}
	s.setNodes(slices.Grow(s.nodes, n))
	s.setData(slices.Grow(s.data, n*3/2))
	s.setLists(slices.Grow(append(s.lists, nodeList{}), n/8))
	s.setListNodes(slices.Grow(s.listNodes, n/2))
}

// finish freezes the nodes if asked to, and trims the tables. It must only
// be called before the store is shared.
func (s *NodeStore) finish(freeze bool) {
	s.frozen = freeze
	s.compact()
}

// compact trims the tables to size. It must only be called while the
// store is not being read concurrently.
func (s *NodeStore) compact() {
	if len(s.nodes) != cap(s.nodes) {
		s.setNodes(exact(s.nodes))
	}
	if len(s.data) != cap(s.data) {
		s.setData(exact(s.data))
	}
	if len(s.lists) != cap(s.lists) {
		s.setLists(exact(s.lists))
	}
	if len(s.listNodes) != cap(s.listNodes) {
		s.setListNodes(exact(s.listNodes))
	}
	s.compactSide()
}

func exact[T any](s []T) []T {
	c := make([]T, len(s))
	copy(c, s)
	return c
}

// compactSide trims the side tables to size. Side values are never
// addressed, so this is safe whenever the store is not being read
// concurrently.
func (s *NodeStore) compactSide() {
	s.stringLists.compact()
	s.nodeSlices.compact()
	s.values.compact()
	if len(s.names) != cap(s.names) {
		s.names = slices.Clip(slices.Clone(s.names))
	}
	if len(s.far) != cap(s.far) {
		s.far = slices.Clip(slices.Clone(s.far))
	}
	if len(s.farLists) != cap(s.farLists) {
		s.farLists = slices.Clip(slices.Clone(s.farLists))
	}
}

func (n Node) hdr() *nodeHeader {
	// SAFETY: A node's index is within its store's nodes.
	return (*nodeHeader)(unsafe.Add(unsafe.Pointer(n.s.nodesBase.Load()), uintptr(n.node)*nodeHeaderSize))
}

// store returns the store holding n.
func (n Node) store() *NodeStore { return n.s }

// payload returns the address of n's fields, which are in its header word
// if inline is set.
func (n Node) payload(inline bool) unsafe.Pointer {
	s := n.s
	// SAFETY: A node's index is within its store's nodes.
	h := (*nodeHeader)(unsafe.Add(unsafe.Pointer(s.nodesBase.Load()), uintptr(n.node)*nodeHeaderSize))
	if inline {
		// SAFETY: Inline fields are in the header's word.
		return unsafe.Pointer(&h.word)
	}
	// SAFETY: A node whose fields are not inline has them at word in data.
	return unsafe.Add(unsafe.Pointer(s.dataBase.Load()), uintptr(h.word)*4)
}

// fields returns the address of n's fields, whatever its kind.
func (n Node) fields() unsafe.Pointer {
	s := n.s
	// SAFETY: A node's index is within its store's nodes.
	h := (*nodeHeader)(unsafe.Add(unsafe.Pointer(s.nodesBase.Load()), uintptr(n.node)*nodeHeaderSize))
	if kindInline[h.kind] {
		// SAFETY: Inline fields are in the header's word.
		return unsafe.Pointer(&h.word)
	}
	// SAFETY: A node whose fields are not inline has them at word in data.
	return unsafe.Add(unsafe.Pointer(s.dataBase.Load()), uintptr(h.word)*4)
}

// kindInline tells for each kind whether its fields fit in the header word.
var kindInline = func() (t [kindFlowReduceLabelData + 1]bool) {
	for k, layout := range kindLayout {
		t[k] = layoutInline[layout]
	}
	return t
}()

// deref returns the node that the field f of n refers to.
func (n Node) deref(f *nodeRef) Node {
	r := *f
	if r&refFar == 0 {
		if r == 0 {
			return Node{}
		}
		return Node{n.s, uint32(r)}
	}
	return n.store().far[r&^refFar]
}

// refAt returns the node that the field at offset off-1 of n's fields
// refers to.
func (n Node) refAt(off uint8) Node {
	// SAFETY: The kind's table gives the offset of an aligned nodeRef.
	return n.deref((*nodeRef)(unsafe.Add(n.fields(), off-1)))
}

// setRef makes the field f of n refer to v.
func (n Node) setRef(f *nodeRef, v Node) { setNodeRef(f, n, v) }

// setNodeRef makes the field f of owner refer to v.
func setNodeRef(f *nodeRef, owner Node, v Node) {
	if v.s == owner.s {
		*f = nodeRef(v.node)
		return
	}
	if v.node == 0 {
		*f = 0
		return
	}
	owner.store().setRef(f, v)
}

// node returns the node that r, a ref held by s, refers to.
func (s *NodeStore) node(r nodeRef) Node {
	if r&refFar == 0 {
		if r == 0 {
			return Node{}
		}
		return Node{s, uint32(r)}
	}
	return s.far[r&^refFar]
}

// setRef makes the ref f, held by s, refer to v.
func (s *NodeStore) setRef(f *nodeRef, v Node) {
	switch {
	case v.node == 0:
		*f = 0
	case v.s == s:
		*f = nodeRef(v.node)
	case *f&refFar != 0:
		// Slots are never shared, so the field's own slot can be reused.
		s.far[*f&^refFar] = v
	default:
		*f = nodeRef(farSlot(len(s.far)))
		s.far = append(s.far, v)
	}
}

func farSlot(i int) uint32 {
	if uint64(i) >= refFar {
		panic("node store far ref overflow")
	}
	return uint32(i) | refFar
}

// A sideRef is a record field's slot in a side table. The first slot holds
// the zero value and is never written, so a zero ref reads as the zero value
// without a slot of its own.
type sideRef uint32

// A sideTable is appended to only by its store's owner, before the store
// is shared.
type sideTable[T any] struct {
	items []T
}

func newSideTable[T any]() sideTable[T] {
	return sideTable[T]{items: make([]T, 1)}
}

func (t *sideTable[T]) get(r sideRef) T {
	return t.items[r]
}

// set stores v in r's slot, allocating one unless v is the zero value.
func (t *sideTable[T]) set(r *sideRef, v T, isZero bool) {
	if *r != 0 {
		t.items[*r] = v
		return
	}
	if !isZero {
		*r = sideRef(len(t.items))
		t.items = append(t.items, v)
	}
}

func (t *sideTable[T]) compact() {
	if len(t.items) != cap(t.items) {
		t.items = slices.Clip(slices.Clone(t.items))
	}
}

// A nameRef is a SymbolName field. Nearly every name in a record is
// interned with an ID below nameFar, and is held as is; any other name, such
// as a derived one, is held in the store's names, at the index in the rest
// of the ref.
type nameRef uint32

const nameFar = 1 << 31

func (n Node) derefName(f *nameRef) SymbolName {
	r := *f
	if r&nameFar == 0 {
		return SymbolName(r)
	}
	return n.store().names[r&^nameFar]
}

// setNameRef makes the field f of n's record hold v.
func (n Node) setNameRef(f *nameRef, v SymbolName) { setNameRef(f, n, v) }

// setNameRef makes the field f of the record of owner hold v.
func setNameRef(f *nameRef, owner Node, v SymbolName) {
	if v < nameFar {
		*f = nameRef(v)
		return
	}
	s := owner.store()
	if *f&nameFar != 0 {
		// Slots are never shared, so the field's own slot can be reused.
		s.names[*f&^nameFar] = v
		return
	}
	if uint64(len(s.names)) >= nameFar {
		panic("node store name overflow")
	}
	*f = nameRef(len(s.names)) | nameFar
	s.names = append(s.names, v)
}

// A textRef is a string field: a range of the store's text, located by
// textStore.get.
type textRef struct {
	off uint32
	len uint32
}

const (
	// The text of a store is addressed in pieces of textPieceSize bytes; a
	// textRef's offset names a piece in its top bits and a position in the
	// piece in the rest.
	textPieceShift = 27
	textPieceSize  = 1 << textPieceShift
	textPieceMask  = textPieceSize - 1
	textPieces     = 1 << (32 - textPieceShift)
	// textArenaMinSize sizes the first piece of a text arena; later pieces
	// double up to textPieceSize.
	textArenaMinSize = 1 << 12
)

// textStore holds the text of a store's string fields. Text found within
// the source is referenced in place; other text is copied into an arena of
// pieces that are never moved or modified, so strings may alias them.
//
// The first pieces are the source's, so that a position in the source is
// its own offset; the arena's pieces follow. Every string is then found the
// same way, from the start of its piece.
type textStore struct {
	pieces [textPieces]unsafe.Pointer
	source string
	// arena is the piece being filled, if arenaSize is not zero, and next is
	// where in it the next string goes.
	arena     uint32
	arenaSize uint32
	next      uint32
}

func (t *textStore) get(r textRef) string {
	// SAFETY: set records a valid source or arena range; retained pieces keep the bytes alive,
	// and allocated string ranges are never modified after publication. A zero ref is empty.
	return unsafe.String((*byte)(unsafe.Add(t.pieces[r.off>>textPieceShift], r.off&textPieceMask)), r.len)
}

func (t *textStore) setSource(text string) {
	if t.arenaSize != 0 || t.source != "" {
		panic("node store source set after text was stored")
	}
	if uint64(len(text)) > (textPieces-1)*textPieceSize {
		panic("node store source too large")
	}
	t.source = text
	// SAFETY: source retains the immutable string backing these piece pointers.
	p := unsafe.Pointer(unsafe.StringData(text))
	for k := 0; uint64(k)*textPieceSize < uint64(len(text)); k++ {
		// SAFETY: The loop bounds each piece start within the retained source string.
		t.pieces[k] = unsafe.Add(p, k*textPieceSize)
	}
}

func (t *textStore) set(r *textRef, v string, _ bool) {
	if len(v) == 0 {
		*r = textRef{}
		return
	}
	if len(t.source) != 0 {
		// SAFETY: Only the address is inspected; source retains the backing bytes.
		start := uintptr(unsafe.Pointer(unsafe.StringData(t.source)))
		// SAFETY: Only the address is inspected, to recognize a range within the retained source.
		p := uintptr(unsafe.Pointer(unsafe.StringData(v)))
		if p >= start && p+uintptr(len(v)) <= start+uintptr(len(t.source)) {
			*r = textRef{off: uint32(p - start), len: uint32(len(v))}
			return
		}
	}
	*r = textRef{off: t.alloc(v), len: uint32(len(v))}
}

// alloc copies v into the arena, returning its offset.
func (t *textStore) alloc(v string) uint32 {
	n := uint32(len(v))
	if n > textPieceSize {
		panic("node text too large")
	}
	if t.arenaSize == 0 || t.next+n > t.arenaSize {
		size := uint32(textArenaMinSize)
		if t.arenaSize != 0 {
			size = min(t.arenaSize*2, textPieceSize)
			t.arena++
		} else {
			// The arena starts after the source's pieces.
			t.arena = uint32((uint64(len(t.source)) + textPieceMask) >> textPieceShift)
		}
		if t.arena >= textPieces {
			panic("node text arena overflow")
		}
		size = max(size, n)
		// SAFETY: The piece pointer retains the newly allocated size-byte backing array.
		t.pieces[t.arena] = unsafe.Pointer(unsafe.SliceData(make([]byte, size)))
		t.arenaSize = size
		t.next = 0
	}
	off := t.next
	// SAFETY: The capacity check reserves n bytes at off in this retained piece;
	// only unused bytes are written, so previously returned strings remain immutable.
	copy(unsafe.Slice((*byte)(unsafe.Add(t.pieces[t.arena], off)), n), v)
	t.next += n
	return t.arena<<textPieceShift | off
}

// factsCache returns the word caching n's subtree facts. The store's table
// of them is made on the first query, since most programs never make one;
// a store still growing makes a larger one when it must, losing at worst
// some cached facts.
func (n Node) factsCache() *atomic.Uint32 {
	s := n.store()
	t := s.facts.Load()
	if t == nil || n.node >= uint32(len(*t)) {
		t = s.growFacts(n.node)
	}
	return &(*t)[n.node]
}

func (s *NodeStore) growFacts(i uint32) *[]atomic.Uint32 {
	for {
		old := s.facts.Load()
		if old != nil && i < uint32(len(*old)) {
			return old
		}
		t := make([]atomic.Uint32, max(len(s.nodes), int(i)+1))
		if old != nil {
			for j := range *old {
				t[j].Store((*old)[j].Load())
			}
		}
		if s.facts.CompareAndSwap(old, &t) {
			return &t
		}
	}
}
