package ast

import (
	"slices"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// A nodeStore holds the nodes created by one NodeFactory: for a parsed
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
// The tables grow by copying. The store's entry in the store table holds
// their current addresses, so that a Node, which is the store's id and an
// index, finds its header with two dependent loads.
type nodeStore struct {
	id uint32

	nodes     []nodeHeader
	data      []uint32
	lists     []nodeList
	listNodes []uint32

	symbols      sideTable[*Symbol]
	symbolTables sideTable[SymbolTable]
	texts        textStore
	stringLists  sideTable[[]string]
	nodeSlices   sideTable[[]Node]
	values       sideTable[any]
	sourceFiles  sideTable[*SourceFile]

	// file is the source file whose tree the store holds, if it holds a
	// parsed file. Every node of the store that is in that tree is in the
	// file.
	file *SourceFile

	// far holds the nodes of other stores that fields refer to, and
	// farLists the lists. Their owners must keep them until this store is
	// released.
	far      []Node
	farLists []NodeList

	// names holds the symbol names that records cannot hold inline; see
	// nameRef.
	names []SymbolName

	// missing holds the indices of the lists that stand for lists missing
	// from the source.
	missing map[uint32]struct{}

	flows       []flowNodeData
	flowLists   []flowListData
	reusedFlows []FlowNode

	// frozen is set once the nodes are shared, after which no more may be
	// created.
	frozen bool

	released bool
}

// A NodeStore owns nodes: those that factories using it make. The nodes
// stay valid until the store is released, after which using them panics,
// so the owner of a store must outlive its nodes. A released store may be
// used again, for new nodes.
//
// A parsed file is an exception: its nodes are in a store of its own,
// which the file owns; see SourceFile.Release.
type NodeStore struct {
	s *nodeStore
}

// NewNodeStore returns a new store. It takes no memory until nodes are
// made in it.
func NewNodeStore() *NodeStore {
	return &NodeStore{}
}

func (o *NodeStore) get() *nodeStore {
	if o.s == nil {
		o.s = newNodeStore()
	}
	return o.s
}

// Release releases the nodes made in o. Any later use of them panics.
func (o *NodeStore) Release() {
	if o == nil {
		return
	}
	o.s.release()
	o.s = nil
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

// maxStores bounds the stores alive at once.
const maxStores = 1 << 22

// A store id is the index of its entry in the store table, plus, above
// that, a generation counting the reuses of the entry. A node of a released
// store thus never matches the entry's id again until the generation wraps,
// so using it panics instead of reading another store, and as a map key it
// differs from the nodes of a later store in the same entry. Ids stay below
// 1<<29, keeping node ids below 1<<61.
const (
	storeSlotMask  = maxStores - 1
	storeGenOne    = maxStores
	storeIDBitMask = 1<<29 - 1
)

// checkStoreIDs makes every access to a node check that its store has not
// been released.
const checkStoreIDs = true

// A storeEntry holds the id of the store using it, the store, and its
// tables, so that a Node, which is the store's id and an index, finds its
// header with two dependent loads. The entry holds the store alive until
// its owner releases it; see nodeStore.release.
type storeEntry struct {
	id        atomic.Uint32
	store     atomic.Pointer[nodeStore]
	nodes     atomic.Pointer[nodeHeader]
	data      atomic.Pointer[uint32]
	lists     atomic.Pointer[nodeList]
	listNodes atomic.Pointer[uint32]
}

// The store table is paged, so that it takes memory only for the entries
// used, and the garbage collector scans only those.
const (
	storePageShift = 10
	storePageSize  = 1 << storePageShift
)

type storePage [storePageSize]storeEntry

var storePages [maxStores / storePageSize]*storePage

// The zero Node and NodeList belong to entry zero, whose id is zero and
// whose tables read as zero values.
var (
	nilNodes     [1]nodeHeader
	nilData      [64]uint32
	nilLists     [1]nodeList
	nilListNodes [1]uint32
)

func init() {
	storePages[0] = new(storePage)
	e := &storePages[0][0]
	e.nodes.Store(&nilNodes[0])
	e.data.Store(&nilData[0])
	e.lists.Store(&nilLists[0])
	e.listNodes.Store(&nilListNodes[0])
}

// slotEntry returns the entry of the slot that id names.
func slotEntry(id uint32) *storeEntry {
	slot := id & storeSlotMask
	return &storePages[slot>>storePageShift][slot&(storePageSize-1)]
}

// storeEntryOf returns the entry of the store with the id, panicking if
// the store has been released.
func storeEntryOf(id uint32) *storeEntry {
	e := slotEntry(id)
	if checkStoreIDs && e.id.Load() != id {
		panicReleasedStore()
	}
	return e
}

func panicReleasedStore() {
	panic("ast: use of a node or list whose node store was released")
}

var storeIDs struct {
	mu   sync.Mutex
	next uint32
	live int
	// free queues the ids of released stores, reused first in first out,
	// and only once storeIDReuseDelay of them are waiting, so that the
	// generation of an entry wraps only after very many releases.
	free []uint32
	head int
}

const storeIDReuseDelay = 1024

// newStoreID returns the id for a new store, whose entry is clear.
func newStoreID() uint32 {
	storeIDs.mu.Lock()
	defer storeIDs.mu.Unlock()
	storeIDs.live++
	if len(storeIDs.free)-storeIDs.head > storeIDReuseDelay {
		id := (storeIDs.free[storeIDs.head] + storeGenOne) & storeIDBitMask
		if id&^storeSlotMask == 0 {
			// Generation zero is skipped, so that no id is zero.
			id += storeGenOne
		}
		storeIDs.head++
		if storeIDs.head >= 4096 && storeIDs.head*2 >= len(storeIDs.free) {
			storeIDs.free = slices.Delete(storeIDs.free, 0, storeIDs.head)
			storeIDs.head = 0
		}
		return id
	}
	if storeIDs.next == maxStores-1 {
		panic("too many node stores")
	}
	storeIDs.next++
	id := storeIDs.next
	if storePages[id>>storePageShift] == nil {
		storePages[id>>storePageShift] = new(storePage)
	}
	return id | storeGenOne
}

// LiveNodeStores returns the number of node stores not yet released.
func LiveNodeStores() int {
	storeIDs.mu.Lock()
	defer storeIDs.mu.Unlock()
	return storeIDs.live
}

// newNodeStore returns a new store. It holds its nodes until its owner
// releases it.
func newNodeStore() *nodeStore {
	s := &nodeStore{
		id:           newStoreID(),
		nodes:        make([]nodeHeader, 1, 8),
		symbols:      newSideTable[*Symbol](),
		symbolTables: newSideTable[SymbolTable](),
		stringLists:  newSideTable[[]string](),
		nodeSlices:   newSideTable[[]Node](),
		values:       newSideTable[any](),
		sourceFiles:  newSideTable[*SourceFile](),
	}
	e := slotEntry(s.id)
	e.store.Store(s)
	e.nodes.Store(&s.nodes[0])
	e.data.Store(&nilData[0])
	e.lists.Store(&nilLists[0])
	e.listNodes.Store(&nilListNodes[0])
	e.id.Store(s.id)
	return s
}

// release drops s's nodes. Any later use of them panics, until the
// generation of the entry wraps. It must only be called once nothing reads
// the nodes any more; calling it again does nothing.
func (s *nodeStore) release() {
	if s == nil || s.released {
		return
	}
	s.released = true
	e := slotEntry(s.id)
	e.id.Store(0)
	e.store.Store(nil)
	e.nodes.Store(nil)
	e.data.Store(nil)
	e.lists.Store(nil)
	e.listNodes.Store(nil)
	*s = nodeStore{id: s.id, released: true, file: s.file}
	storeIDs.mu.Lock()
	storeIDs.live--
	storeIDs.free = append(storeIDs.free, s.id)
	storeIDs.mu.Unlock()
}

func (s *nodeStore) entry() *storeEntry { return slotEntry(s.id) }

// setNodes replaces s's nodes, publishing the new ones first, so that a
// reader that loaded the old ones still holds them.
func (s *nodeStore) setNodes(nodes []nodeHeader) {
	// SAFETY: The entry points at the first element of nodes, which
	// stays reachable through it until the store is released.
	s.entry().nodes.Store(unsafe.SliceData(nodes))
	s.nodes = nodes
}

func (s *nodeStore) setData(data []uint32) {
	if cap(data) != 0 {
		// SAFETY: The entry points at the first element of data, which
		// stays reachable through it until the store is released.
		s.entry().data.Store(unsafe.SliceData(data))
	}
	s.data = data
}

func (s *nodeStore) setLists(lists []nodeList) {
	// SAFETY: The entry points at the first element of lists, which
	// stays reachable through it until the store is released.
	s.entry().lists.Store(unsafe.SliceData(lists))
	s.lists = lists
}

func (s *nodeStore) setListNodes(listNodes []uint32) {
	if cap(listNodes) != 0 {
		// SAFETY: The entry points at the first element of listNodes, which
		// stays reachable through it until the store is released.
		s.entry().listNodes.Store(unsafe.SliceData(listNodes))
	}
	s.listNodes = listNodes
}

func (s *nodeStore) checkMutable() {
	if s.frozen {
		panic("node store is frozen")
	}
}

// addNode adds a zero node and returns its index.
func (s *nodeStore) addNode() uint32 {
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
func (s *nodeStore) addData(n int) uint32 {
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

func (s *nodeStore) newNode(kind Kind, layout nodeLayout, n Node) Node {
	if kindLayout[kind] != layout {
		panic("node kind does not match its record layout")
	}
	s.nodes[n.node].kind = kind
	return n
}

// newRecord adds a node whose fields are a zero T, returning them and the
// node. The fields stay where they are only until the store's next node.
func newRecord[T any](s *nodeStore, layout nodeLayout) (*T, Node) {
	size := unsafe.Sizeof(*new(T))
	if unsafe.Alignof(*new(T)) > 4 || layoutInline[layout] != (size <= 4) {
		panic("node record does not fit its layout")
	}
	i := s.addNode()
	n := Node{s.id, i}
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
func (s *nodeStore) sizeForSource() {
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
func (s *nodeStore) finish(freeze bool) {
	s.frozen = freeze
	s.compact()
}

// compact trims the tables to size. It must only be called while the
// store is not being read concurrently.
func (s *nodeStore) compact() {
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
func (s *nodeStore) compactSide() {
	s.symbols.compact()
	s.symbolTables.compact()
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

// IsLive reports whether n is nil or its store has not been released.
func (n Node) IsLive() bool {
	return slotEntry(n.file).id.Load() == n.file
}

func (n Node) hdr() *nodeHeader {
	// SAFETY: A node's index is within its store's nodes.
	return (*nodeHeader)(unsafe.Add(unsafe.Pointer(storeEntryOf(n.file).nodes.Load()), uintptr(n.node)*nodeHeaderSize))
}

// store returns the store that owns n.
func (n Node) store() *nodeStore {
	return storeEntryOf(n.file).store.Load()
}

// payload returns the address of n's fields, which are in its header word
// if inline is set.
func (n Node) payload(inline bool) unsafe.Pointer {
	e := storeEntryOf(n.file)
	// SAFETY: A node's index is within its store's nodes.
	h := (*nodeHeader)(unsafe.Add(unsafe.Pointer(e.nodes.Load()), uintptr(n.node)*nodeHeaderSize))
	if inline {
		// SAFETY: Inline fields are in the header's word.
		return unsafe.Pointer(&h.word)
	}
	// SAFETY: A node whose fields are not inline has them at word in data.
	return unsafe.Add(unsafe.Pointer(e.data.Load()), uintptr(h.word)*4)
}

// fields returns the address of n's fields, whatever its kind.
func (n Node) fields() unsafe.Pointer {
	e := storeEntryOf(n.file)
	// SAFETY: A node's index is within its store's nodes.
	h := (*nodeHeader)(unsafe.Add(unsafe.Pointer(e.nodes.Load()), uintptr(n.node)*nodeHeaderSize))
	if kindInline[h.kind] {
		// SAFETY: Inline fields are in the header's word.
		return unsafe.Pointer(&h.word)
	}
	// SAFETY: A node whose fields are not inline has them at word in data.
	return unsafe.Add(unsafe.Pointer(e.data.Load()), uintptr(h.word)*4)
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
		return Node{n.file, uint32(r)}
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
	if v.file == owner.file {
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
func (s *nodeStore) node(r nodeRef) Node {
	if r&refFar == 0 {
		if r == 0 {
			return Node{}
		}
		return Node{s.id, uint32(r)}
	}
	return s.far[r&^refFar]
}

// setRef makes the ref f, held by s, refer to v.
func (s *nodeStore) setRef(f *nodeRef, v Node) {
	switch {
	case v.node == 0:
		*f = 0
	case v.file == s.id:
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
