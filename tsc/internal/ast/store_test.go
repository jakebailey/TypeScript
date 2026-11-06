package ast

import (
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/microsoft/TypeScript/tsc/internal/testutil"
	"gotest.tools/v3/assert"
)

// Records must hold no Go pointers, so the garbage collector can skip
// them; pointer-valued fields belong in the store's side tables.
func TestNodeRecordsArePointerFree(t *testing.T) {
	t.Parallel()
	for layout := nodeLayout(1); layout < layoutCount; layout++ {
		typ := reflect.TypeOf(layoutRecord(layout))
		assert.Assert(t, typ != nil, "layout %d has no record type", layout)
		if path := findPointer(typ, typ.Name()); path != "" {
			t.Errorf("record field %s holds a pointer", path)
		}
	}
}

// Records are ordered so that their only padding rounds out their size.
func TestNodeRecordsAreUnpadded(t *testing.T) {
	t.Parallel()
	for layout := nodeLayout(1); layout < layoutCount; layout++ {
		typ := reflect.TypeOf(layoutRecord(layout))
		var used uintptr
		for field := range typ.Fields() {
			used += field.Type.Size()
		}
		if typ.Size()-used >= uintptr(typ.Align()) {
			t.Errorf("record %s has %d bytes of padding", typ.Name(), typ.Size()-used)
		}
	}
}

func findPointer(typ reflect.Type, path string) string {
	switch typ.Kind() {
	case reflect.Struct:
		for field := range typ.Fields() {
			if p := findPointer(field.Type, path+"."+field.Name); p != "" {
				return p
			}
		}
	case reflect.Array:
		return findPointer(typ.Elem(), path+"[]")
	case reflect.Pointer, reflect.String, reflect.Slice, reflect.Map, reflect.Interface, reflect.Func, reflect.Chan, reflect.UnsafePointer:
		return path + " (" + typ.String() + ")"
	}
	return ""
}

func newTestFactory(t *testing.T) *NodeFactory {
	t.Helper()
	return NewNodeFactory(nil, NodeFactoryHooks{})
}

// Nodes keep their handles as the store's tables grow and are trimmed.
func TestNodeStoreGrowth(t *testing.T) {
	t.Parallel()
	factory := newTestFactory(t)
	var statements []Node
	for i := range 20000 {
		statements = append(statements, factory.NewExpressionStatement(factory.NewIdentifier(strconv.Itoa(i))))
	}
	store := factory.nodeStore()
	for i, statement := range statements {
		expression := statement.Expression()
		assert.Equal(t, expression.Text(), strconv.Itoa(i))
		assert.Equal(t, expression.store(), store)
		expression.SetParent(statement)
		assert.Equal(t, expression.Parent(), statement)
	}
	assert.Equal(t, len(store.far), 0)
	factory.FinishStore()
	assert.Equal(t, len(store.nodes), len(statements)*2+1)
	assert.Equal(t, len(store.nodes), cap(store.nodes))
	assert.Equal(t, statements[0].Expression().Text(), "0")
	assert.Equal(t, statements[19999].Expression().Parent(), statements[19999])
}

func TestNodeIds(t *testing.T) {
	t.Parallel()
	a := newTestFactory(t)
	b := newTestFactory(t)
	nodes := []Node{a.NewToken(KindCommaToken), a.NewIdentifier("x"), b.NewToken(KindCommaToken), b.NewIdentifier("x")}
	seen := make(map[NodeId]bool)
	for _, node := range nodes {
		id := GetNodeId(node)
		assert.Assert(t, id != 0)
		assert.Assert(t, !seen[id], "id %d given twice", id)
		seen[id] = true
		assert.Equal(t, GetNodeId(node), id)
	}
}

func TestNodeStoreCrossStoreRefs(t *testing.T) {
	t.Parallel()
	a := newTestFactory(t)
	b := newTestFactory(t)
	target := a.NewIdentifier("x")
	first := b.NewExpressionStatement(target)
	second := b.NewExpressionStatement(target)
	assert.Equal(t, first.Expression(), target)
	assert.Equal(t, second.Expression(), target)
	target.SetParent(first)
	assert.Equal(t, target.Parent(), first)
	assert.Equal(t, len(b.nodeStore().far), 2)
	target.SetParent(second)
	target.SetParent(first)
	assert.Equal(t, len(a.nodeStore().far), 1, "rewriting a far ref should reuse its slot")
	first.AsExpressionStatement().SetExpression(Node{})
	assert.Assert(t, first.Expression().IsNil())
	assert.Equal(t, first.Expression(), Node{})
	list := b.NewNodeList([]Node{target, Node{}, first})
	assert.Equal(t, list.At(0), target)
	assert.Equal(t, list.At(1), Node{})
	assert.Equal(t, list.At(2), first)
}

func TestFarSlotBounds(t *testing.T) {
	t.Parallel()
	assert.Equal(t, farSlot(1<<31-1), uint32(0xffffffff))
	if strconv.IntSize == 64 {
		overflow := uint64(1 << 31)
		testutil.AssertPanics(t, func() { farSlot(int(overflow)) }, "node store far ref overflow")
	}
}

func TestNodeSideTableZeroValues(t *testing.T) {
	t.Parallel()
	factory := newTestFactory(t)
	node := factory.NewBlock(StatementList{}, false)
	assert.Assert(t, node.AsBlock().Statements().IsNil())
	assert.Assert(t, node.AsBlock().Locals() == nil)

	locals := NewSymbolTable()
	node.AsBlock().SetLocals(locals)
	assert.Assert(t, node.AsBlock().Locals() != nil)
	node.AsBlock().SetLocals(nil)
	assert.Assert(t, node.AsBlock().Locals() == nil)
	assert.Equal(t, len(factory.nodeStore().binder.containers.extra), 0)
}

func TestSparseMap(t *testing.T) {
	t.Parallel()
	var m sparseMap[uint32]
	const n = 100
	for _, i := range []uint32{3, 31, 32, 70} {
		m.set(i, i+1000, n)
	}
	m.set(5, 0, n)
	m.pack()
	assert.Assert(t, m.dense == nil)
	assert.Equal(t, len(m.blocks), 3)
	assert.Equal(t, len(m.values), 4)
	for i := range uint32(n) {
		want := uint32(0)
		if i == 3 || i == 31 || i == 32 || i == 70 {
			want = i + 1000
		}
		assert.Equal(t, m.get(i), want, "index %d", i)
	}
	m.set(32, 7, 0)
	m.set(33, 8, 0)
	m.set(99, 9, 0)
	assert.Equal(t, m.get(32), uint32(7))
	assert.Equal(t, m.get(33), uint32(8))
	assert.Equal(t, m.get(99), uint32(9))
	assert.Equal(t, m.get(70), uint32(1070))
	assert.Equal(t, len(m.extra), 2)
}

func TestSortedMap(t *testing.T) {
	t.Parallel()
	var m sortedMap[uint32]
	for _, i := range []uint32{900, 4, 70} {
		m.set(i, i+1000)
	}
	m.set(5, 0)
	m.pack()
	assert.Assert(t, m.extra == nil)
	assert.DeepEqual(t, m.keys, []uint32{4, 70, 900})
	for i := range uint32(1000) {
		want := uint32(0)
		if i == 4 || i == 70 || i == 900 {
			want = i + 1000
		}
		assert.Equal(t, m.get(i), want, "index %d", i)
	}
	m.set(70, 7)
	m.set(71, 8)
	assert.Equal(t, m.get(70), uint32(7))
	assert.Equal(t, m.get(71), uint32(8))
	assert.Equal(t, len(m.extra), 1)
}

func TestNodeTextRefs(t *testing.T) {
	t.Parallel()
	factory := newTestFactory(t)
	source := "let alpha = beta;"
	factory.SetSourceText(source)
	inSource := factory.NewIdentifier(source[4:9])
	assert.Equal(t, factory.nodeStore().texts.next, uint32(0))
	synthesized := factory.NewIdentifier("gamma")
	empty := factory.NewIdentifier("")
	var long []Node
	for i := range 2000 {
		long = append(long, factory.NewIdentifier(strconv.Itoa(i)+string(make([]byte, i))))
	}
	assert.Equal(t, inSource.Text(), "alpha")
	assert.Equal(t, synthesized.Text(), "gamma")
	assert.Equal(t, empty.Text(), "")
	for i, node := range long {
		assert.Equal(t, node.Text(), strconv.Itoa(i)+string(make([]byte, i)))
	}
}

// A node keeps its store alive, and node ids stay unique across stores.
func TestNodeKeepsStoreAlive(t *testing.T) {
	t.Parallel()
	var nodes []Node
	ids := map[NodeId]bool{}
	for i := range 100 {
		node := NewNodeFactory(nil, NodeFactoryHooks{}).NewIdentifier(strconv.Itoa(i))
		nodes = append(nodes, node)
		ids[GetNodeId(node)] = true
	}
	runtime.GC()
	for i, node := range nodes {
		assert.Equal(t, node.Text(), strconv.Itoa(i))
	}
	assert.Equal(t, len(ids), len(nodes))
	assert.Equal(t, GetNodeId(Node{}), NodeId(0))
}
