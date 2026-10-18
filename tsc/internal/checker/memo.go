package checker

import (
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
)

// Checker side effects and memoization
//
// Checking a file produces outputs other than types: diagnostics, reference
// marks used by unused-identifier checks, deferred nodes, and so on. Those
// outputs are produced while computing memoized results (the type of a symbol,
// the resolved symbol of an identifier, ...). If such a result is first computed
// for an API query rather than while checking, a later check reuses the cached
// result without recomputing it, so the outputs would historically either leak
// (reported for code that checking never visits) or be lost (never reported for
// code that checking does visit). Both make diagnostics depend on query history.
//
// To make checking independent of query history, an activation is either
// reporting (driven by checking) or not (driven by queries). A non-reporting
// activation does not apply side effects. Instead it records them against the
// innermost memoized computation in progress, along with the memoized results
// that computation reused. When a reporting activation later reuses a memoized
// result, it replays the recorded effects of that result and everything it
// transitively reused, exactly as if it had computed the result itself.
// Effects produced outside any memoized computation belong solely to the query
// and are discarded.

type memoField uint8

const (
	memoResolvedSymbol memoField = iota
	memoValueType
	memoWriteType
	memoDeclaredType
	memoAliasTarget
	memoReturnType
	memoResolvedSignature
	memoExpressionType
	memoTypeNodeType
	memoBaseTypes
	memoBaseConstructorType
	memoStructuredMembers
	memoGlobal
	memoTypeParameterConstraint
	memoTypeParameterDefault
	memoTypeAlias
	memoEnumMemberValue
	memoBaseConstraint
	memoAccessorType
	memoComputedPropertyName
	memoRegularExpression
	memoContextChecked
	memoInitializerIsUndefined
	memoFlowExpressionType
	memoFlowLoopType
	memoJsxImplicitImportContainer
	memoModuleExports
	memoInstantiationExpressionType
	memoRelationOverflow
	memoIterationTypes                                                               // followed by one field per IterationUseCacheFlags combination
	memoResolvedMembers = memoIterationTypes + memoField(IterationUseCacheFlags) + 1 // followed by one field per MembersOrExportsResolutionKind
)

// memoKey identifies a memoized result: the address of the entity owning the
// cache slot (an *ast.Node, *ast.Symbol, *Type, or *Signature) shifted left by
// eight bits, plus the memoField of the slot. Packing into an integer keeps map
// operations on the fast path. Entities live as long as the program or the
// checker store, so an address cannot be reused while its keys exist.
type memoKey uint64

func memoKeyOf[T any](entity *T, field memoField) memoKey {
	return memoKey(uint64(uintptr(unsafe.Pointer(entity)))<<8 | uint64(field))
}

type memoEffects struct {
	effects  []func(c *Checker)
	deps     []*memoEffects
	parent   *memoEffects // Frame d was most recently added to; avoids duplicate deps
	replayed bool
}

func (f *memoEffects) addDep(d *memoEffects) {
	if f == d || d.parent == f {
		return
	}
	d.parent = f
	f.deps = append(f.deps, d)
}

// memoSlot is the memo state of one memoized result.
type memoSlot struct {
	pending *memoEffects // Effects not yet replayed by a reporting activation
	depth   int          // 1-based index in memoFrames of the outermost computation in progress, or 0
}

type memoFrame struct {
	effects *memoEffects // Allocated on first use
	slot    *memoSlot
}

// memoActive reports whether memoized entry points must track effects. When
// false (a reporting activation with nothing pending), they may skip enterMemo
// and exitMemo entirely.
func (c *Checker) memoActive() bool {
	return !c.reporting || c.memoPendingCount != 0
}

// enterMemo is called on entry to a memoized function whose result for this
// input is cached in the slot identified by key. It accounts for effects
// recorded by a previous computation of that result, and in a non-reporting
// activation starts a frame recording effects of this computation. The result
// must be passed to exitMemo.
func (c *Checker) enterMemo(key memoKey) bool {
	if c.reporting {
		if c.memoPendingCount != 0 {
			if slot := c.memoSlots[key]; slot != nil && slot.pending != nil {
				f := slot.pending
				slot.pending = nil
				c.memoPendingCount--
				c.replayMemo(f)
			}
		}
		return false
	}
	slot := c.memoSlots[key]
	if slot == nil {
		if c.memoSlots == nil {
			c.memoSlots = make(map[memoKey]*memoSlot)
		}
		slot = &memoSlot{}
		c.memoSlots[key] = slot
	} else if slot.pending != nil {
		if parent := c.currentMemoFrame(); parent != nil {
			parent.addDep(slot.pending)
		}
	}
	c.memoFrames = append(c.memoFrames, memoFrame{slot: slot})
	if slot.depth != 0 {
		// Re-entering a computation in progress: the inner result is derived
		// from whatever in-progress marker the outer computation left behind.
		c.memoCycle(slot.depth)
	} else {
		slot.depth = len(c.memoFrames)
	}
	return true
}

// exitMemo finishes a computation started by enterMemo, retaining its effects
// for replay by later reuse of the result. The effects also become part of the
// enclosing computation, since it observed the result.
func (c *Checker) exitMemo(pushed bool) {
	if !pushed {
		return
	}
	n := len(c.memoFrames) - 1
	frame := c.memoFrames[n]
	c.memoFrames[n] = memoFrame{}
	c.memoFrames = c.memoFrames[:n]
	slot := frame.slot
	if slot.depth == n+1 {
		slot.depth = 0
	}
	f := frame.effects
	if f == nil {
		return
	}
	if slot.pending != nil {
		f.addDep(slot.pending)
	} else {
		c.memoPendingCount++
	}
	slot.pending = f
	if parent := c.currentMemoFrame(); parent != nil {
		parent.addDep(f)
	}
}

func (c *Checker) replayMemo(f *memoEffects) {
	if f.replayed {
		return
	}
	f.replayed = true
	for _, e := range f.effects {
		e(c)
	}
	for _, d := range f.deps {
		c.replayMemo(d)
	}
	f.effects = nil
	f.deps = nil
}

func (c *Checker) currentMemoFrame() *memoEffects {
	n := len(c.memoFrames) - 1
	if n < 0 {
		return nil
	}
	return c.memoFrameAt(n)
}

// recordEffect records a side effect produced by a non-reporting activation
// against the innermost memoized computation, if any. Callers must only call
// this when !c.reporting, and must not apply the effect themselves.
func (c *Checker) recordEffect(effect func(c *Checker)) {
	if f := c.currentMemoFrame(); f != nil {
		f.effects = append(f.effects, effect)
	}
}

// withReporting runs fn in a reporting activation.
func (c *Checker) withReporting(fn func()) {
	saveReporting := c.reporting
	saveFrames := c.memoFrames
	c.reporting = true
	c.memoFrames = nil
	fn()
	c.reporting = saveReporting
	c.memoFrames = saveFrames
}

// memoCycle is called when a computation observes another computation that is
// still in progress, whose enclosing memo frame is at index depth-1. Every
// computation started since then produced its result from the incomplete
// state of that computation, so reusing any of them must also replay the
// effects of the whole cycle (for example, its circularity error).
func (c *Checker) memoCycle(depth int) {
	if depth == 0 {
		return
	}
	outer := c.memoFrameAt(depth - 1)
	for i := depth; i < len(c.memoFrames); i++ {
		f := c.memoFrameAt(i)
		f.addDep(outer)
	}
}

func (c *Checker) memoFrameAt(i int) *memoEffects {
	f := c.memoFrames[i].effects
	if f == nil {
		f = &memoEffects{}
		c.memoFrames[i].effects = f
	}
	return f
}

// memoizeGlobal memoizes the resolution of a global, replaying effects of
// the resolution wherever the cached result is observed.
func memoizeGlobal[T any](c *Checker, fn func() T) func() T {
	cell := &struct{ resolve func() T }{resolve: core.Memoize(fn)}
	return func() T {
		if c.memoActive() {
			pushed := c.enterMemo(memoKeyOf(cell, memoGlobal))
			result := cell.resolve()
			c.exitMemo(pushed)
			return result
		}
		return cell.resolve()
	}
}
