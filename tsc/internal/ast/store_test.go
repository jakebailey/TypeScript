package ast

import (
	"reflect"
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
	store := NewNodeStore()
	t.Cleanup(store.Release)
	return NewNodeFactory(store, NodeFactoryHooks{})
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
	assert.Equal(t, len(factory.nodeStore().symbolTables.items), 2)
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

func TestStoreIDReuseChangesGeneration(t *testing.T) {
	t.Parallel()
	released := map[uint32]uint32{}
	for range 2 * storeIDReuseDelay {
		s := newNodeStore()
		released[s.id&storeSlotMask] = s.id
		s.release()
	}
	reused := 0
	var stores []*nodeStore
	for range 4 * storeIDReuseDelay {
		s := newNodeStore()
		stores = append(stores, s)
		if old, ok := released[s.id&storeSlotMask]; ok {
			assert.Assert(t, s.id != old)
			reused++
		}
	}
	assert.Assert(t, reused > 0)
	for _, s := range stores {
		s.release()
	}
}

// A handle into a released store panics rather than reading whatever store
// has its slot now.
func TestReleasedStorePanics(t *testing.T) {
	t.Parallel()
	store := NewNodeStore()
	node := NewNodeFactory(store, NodeFactoryHooks{}).NewIdentifier("x")
	assert.Equal(t, node.Text(), "x")
	assert.Assert(t, node.IsLive())
	store.Release()
	assert.Assert(t, !node.IsLive())
	testutil.AssertPanics(t, func() { _ = node.Kind() }, "ast: use of a node or list whose node store was released")
	store.Release()
}
