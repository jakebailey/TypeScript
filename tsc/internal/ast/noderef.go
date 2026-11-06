package ast

import (
	"slices"
	"sync"
	"sync/atomic"
)

// A NodeRef names a node by the number of its store in a FileTable and its
// index there. It is half the size of a Node, for structures that hold many
// nodes, and means nothing without its table: only the table's owner can
// turn it back into a Node. The zero NodeRef is the absent node.
type NodeRef struct {
	file uint32
	node uint32
}

func (r NodeRef) IsNil() bool { return r.node == 0 }

// File returns the number of r's store in its table.
func (r NodeRef) File() uint32 { return r.file }

// Index returns r's index in its store.
func (r NodeRef) Index() uint32 { return r.node }

const (
	fileTablePageShift = 8
	fileTablePageSize  = 1 << fileTablePageShift
)

type fileTablePage [fileTablePageSize]atomic.Pointer[NodeStore]

// A FileTable numbers stores for refs: first the files of a program, in
// order, then any other store whose nodes are turned into refs (synthetic
// nodes, lazily parsed JSDoc), as they are met. It may be shared by
// goroutines.
type FileTable struct {
	files []*NodeStore

	mu    sync.Mutex
	index map[*NodeStore]uint32
	// pages holds the stores numbered after files, published so that
	// readers need no lock.
	pages atomic.Pointer[[]*fileTablePage]
	count uint32

	binderOnce sync.Once
	binder     *BinderSymbols
}

// A fileSlot is a store's number in the table that last numbered it.
type fileSlot struct {
	table *FileTable
	file  uint32
}

// NewFileTable returns a table numbering the stores of files first.
func NewFileTable(files []*SourceFile) *FileTable {
	t := &FileTable{
		files: make([]*NodeStore, len(files)),
		index: make(map[*NodeStore]uint32, len(files)),
	}
	for i, file := range files {
		s := file.store()
		t.files[i] = s
		t.index[s] = uint32(i)
		s.slot.Store(&fileSlot{t, uint32(i)})
	}
	t.count = uint32(len(files))
	return t
}

// Ref returns n's ref in t, numbering its store if t has not yet.
func (t *FileTable) Ref(n Node) NodeRef {
	if n.s == nil {
		return NodeRef{}
	}
	if slot := n.s.slot.Load(); slot != nil && slot.table == t {
		return NodeRef{slot.file, n.node}
	}
	return NodeRef{t.number(n.s), n.node}
}

// StoreNumber returns s's number in t, numbering it if t has not yet.
func (t *FileTable) StoreNumber(s *NodeStore) uint32 {
	if slot := s.slot.Load(); slot != nil && slot.table == t {
		return slot.file
	}
	return t.number(s)
}

func (t *FileTable) number(s *NodeStore) uint32 {
	t.mu.Lock()
	defer t.mu.Unlock()
	file, ok := t.index[s]
	if !ok {
		file = t.count
		k := file - uint32(len(t.files))
		var pages []*fileTablePage
		if p := t.pages.Load(); p != nil {
			pages = *p
		}
		if int(k>>fileTablePageShift) == len(pages) {
			pages = append(slices.Clip(pages), new(fileTablePage))
			t.pages.Store(&pages)
		}
		pages[k>>fileTablePageShift][k&(fileTablePageSize-1)].Store(s)
		t.index[s] = file
		t.count++
	}
	// A store in several tables keeps the slot of the one that met it
	// last.
	s.slot.Store(&fileSlot{t, file})
	return file
}

// Store returns the store numbered file in t.
func (t *FileTable) Store(file uint32) *NodeStore {
	if file < uint32(len(t.files)) {
		return t.files[file]
	}
	k := file - uint32(len(t.files))
	return (*t.pages.Load())[k>>fileTablePageShift][k&(fileTablePageSize-1)].Load()
}

// Node returns the node r names.
func (t *FileTable) Node(r NodeRef) Node {
	if r.node == 0 {
		return Node{}
	}
	return Node{t.Store(r.file), r.node}
}

// A RefCache turns nodes into refs of a table, remembering the last store
// it numbered, since nodes mostly come file by file. It must not be shared
// by goroutines.
type RefCache struct {
	table *FileTable
	last  *NodeStore
	file  uint32
}

func NewRefCache(t *FileTable) RefCache {
	return RefCache{table: t}
}

// Table returns the table c makes refs of.
func (c *RefCache) Table() *FileTable { return c.table }

// Ref returns n's ref.
func (c *RefCache) Ref(n Node) NodeRef {
	if n.s == c.last && n.s != nil {
		return NodeRef{c.file, n.node}
	}
	return c.refSlow(n)
}

func (c *RefCache) refSlow(n Node) NodeRef {
	r := c.table.Ref(n)
	if n.s != nil {
		c.last = n.s
		c.file = r.file
	}
	return r
}
