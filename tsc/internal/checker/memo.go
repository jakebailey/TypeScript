package checker

import (
	"iter"
	"maps"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/core"
	"github.com/microsoft/TypeScript/tsc/internal/debug"
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
	memoCell memoField = iota // A memo[T], keyed by its own address
	memoWriteType
	memoDeclaredType
	memoTypeNodeType
	memoBaseTypes
	memoStructuredMembers
	memoGlobal
	memoTypeParameterConstraint
	memoTypeParameterDefault
	memoTypeAlias
	memoEnumMemberValue
	memoBaseConstraint
	memoAccessorType
	memoRegularExpression
	memoContextChecked
	memoInitializerIsUndefined
	memoFlowExpressionType
	memoFlowLoopType
	memoJsxImplicitImportContainer
	memoModuleExports
	memoInstantiationExpressionType
	memoRelationOverflow
	memoResolvedMembers // followed by one field per MembersOrExportsResolutionKind
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

// memo is a cached result. Cached checker state must live in a memo so that
// every read accounts for the effects of the computation that produced it;
// see the package comment above. The value is only accessible through
// methods. The memo state of a result is kept in the store's slot map, keyed
// by the address of the memo, so a memo costs nothing beyond its value.
type memo[T any] struct {
	_         memoNoCopy
	memoValue T
}

// memoNoCopy makes go vet reject copies of a memo, which would bypass it.
type memoNoCopy struct{}

func (*memoNoCopy) Lock()   {}
func (*memoNoCopy) Unlock() {}

func (m *memo[T]) key() memoKey {
	return memoKeyOf(m, memoCell)
}

// get returns the cached result, accounting for the effects of computing it.
func (m *memo[T]) get(c *Checker) T {
	if c.memoActive() {
		c.observeMemo(m.key())
	}
	return m.memoValue
}

// set caches a result. When set outside the result's own computation (for
// example, the types of binding elements assigned while resolving the type of
// their declaration), the result depends on the computation setting it.
func (m *memo[T]) set(c *Checker, v T) {
	m.memoValue = v
	if !c.reporting {
		c.memoAttach(m.key())
	}
}

// peek returns the cached result without accounting for effects. It is only
// for a result whose computation is in progress, where reading it does not
// observe it.
func (m *memo[T]) peek() T {
	return m.memoValue
}

// exchange replaces the cached result without accounting for effects. It is
// only for speculative computations that restore the previous result after.
func (m *memo[T]) exchange(v T) T {
	old := m.memoValue
	m.memoValue = v
	return old
}

// init sets the result of a cell whose owner is being created. Nothing can
// observe the owner before its creator returns it, so the result needs no
// dependency of its own.
func (m *memo[T]) init(v T) {
	m.memoValue = v
}

// enter starts the computation of the result; see enterMemo.
func (m *memo[T]) enter(c *Checker) bool {
	return c.enterMemo(m.key())
}

// memoMap is a map of cached results, each accounted for like a memo. The
// entries are only accessible through methods. Memo state is kept per entry in
// a side map allocated on first use by a non-reporting activation.
//
// An entry should be computed between enter and exitMemo, so that reusing it
// replays the effects of computing that entry alone. An entry set outside its
// own computation depends on the whole enclosing computation instead.
type memoMap[K comparable, V any] struct {
	_           memoNoCopy
	memoEntries map[K]V
	memoSlots   map[K]*memoSlot
}

// get returns the cached result for k, accounting for the effects of
// computing it.
func (m *memoMap[K, V]) get(c *Checker, k K) (V, bool) {
	if m.memoSlots != nil && c.memoActive() {
		c.observeSlot(m.memoSlots[k])
	}
	v, ok := m.memoEntries[k]
	return v, ok
}

// peek returns the cached result for k without observing it, for results
// that are interned and never change.
func (m *memoMap[K, V]) peek(k K) (V, bool) {
	v, ok := m.memoEntries[k]
	return v, ok
}

// has reports whether a result is cached for k; see get.
func (m *memoMap[K, V]) has(c *Checker, k K) bool {
	_, ok := m.get(c, k)
	return ok
}

// isEmpty reports whether no result is cached, without observing any.
func (m *memoMap[K, V]) isEmpty() bool {
	return len(m.memoEntries) == 0
}

// values iterates the cached results without observing them, for
// inspection outside of checking.
func (m *memoMap[K, V]) values() iter.Seq[V] {
	return maps.Values(m.memoEntries)
}

// set caches the result for k; see memo.set.
func (m *memoMap[K, V]) set(c *Checker, k K, v V) {
	if m.memoEntries == nil {
		m.memoEntries = make(map[K]V)
	}
	m.memoEntries[k] = v
	if !c.reporting && len(c.memoFrames) != 0 {
		c.attachSlot(m.slot(k))
	}
}

// intern caches a result constructed from the inputs in k alone, without
// consulting any cached result, so computing it has no effects to account for.
func (m *memoMap[K, V]) intern(k K, v V) {
	if m.memoEntries == nil {
		m.memoEntries = make(map[K]V)
	}
	m.memoEntries[k] = v
}

// compute computes and caches the result for k in a computation of its own.
func (m *memoMap[K, V]) compute(c *Checker, k K, fn func() V) V {
	pushed := c.memoActive() && m.enter(c, k)
	v := fn()
	m.set(c, k, v)
	c.exitMemo(pushed)
	return v
}

// enter starts the computation of the result for k; see enterMemo.
func (m *memoMap[K, V]) enter(c *Checker, k K) bool {
	if c.reporting {
		if c.memoPendingCount != 0 && m.memoSlots != nil {
			c.replayPending(m.memoSlots[k])
		}
		return false
	}
	return c.enterSlot(m.slot(k))
}

func (m *memoMap[K, V]) slot(k K) *memoSlot {
	slot := m.memoSlots[k]
	if slot == nil {
		if m.memoSlots == nil {
			m.memoSlots = make(map[K]*memoSlot)
		}
		slot = &memoSlot{}
		m.memoSlots[k] = slot
	}
	return slot
}

// memoSet tracks computations that run at most once per key, such as checks
// whose only results are diagnostics. Each run is a computation of its own, so
// skipping a key that already ran accounts for the effects of that run.
type memoSet[K comparable] struct {
	m memoMap[K, struct{}]
}

// has reports whether k has run; see memoMap.get.
func (s *memoSet[K]) has(c *Checker, k K) bool {
	return s.m.has(c, k)
}

// once runs fn unless k has run already. k is marked before fn runs, so fn
// does not run again if it re-enters.
func (s *memoSet[K]) once(c *Checker, k K, fn func()) {
	if s.m.has(c, k) {
		return
	}
	pushed := c.memoActive() && s.m.enter(c, k)
	s.m.set(c, k, struct{}{})
	fn()
	c.exitMemo(pushed)
}

// memoOnce is a memoSet for a single key, kept in the links of the entity it
// guards.
type memoOnce struct {
	done memo[bool]
}

// once runs fn unless it has run already; see memoSet.once.
func (o *memoOnce) once(c *Checker, fn func()) {
	if o.done.get(c) {
		return
	}
	pushed := c.memoActive() && o.done.enter(c)
	o.done.set(c, true)
	fn()
	c.exitMemo(pushed)
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
			c.replayPending(c.memoSlots[key])
		}
		return false
	}
	return c.enterSlot(c.memoSlot(key))
}

// enterSlot is enterMemo for the slot of a result, in a non-reporting
// activation.
func (c *Checker) enterSlot(slot *memoSlot) bool {
	if slot.pending != nil {
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

func (c *Checker) memoSlot(key memoKey) *memoSlot {
	slot := c.memoSlots[key]
	if slot == nil {
		if c.memoSlots == nil {
			c.memoSlots = make(map[memoKey]*memoSlot)
		}
		slot = &memoSlot{}
		c.memoSlots[key] = slot
	}
	return slot
}

// memoAttach makes the result identified by key depend on the innermost
// computation in progress, unless that computation is the result's own.
func (c *Checker) memoAttach(key memoKey) {
	if len(c.memoFrames) == 0 {
		return
	}
	c.attachSlot(c.memoSlot(key))
}

// attachSlot is memoAttach for the slot of a result.
func (c *Checker) attachSlot(slot *memoSlot) {
	n := len(c.memoFrames) - 1
	if n < 0 {
		return
	}
	if c.memoFrames[n].slot == slot {
		return
	}
	f := c.memoFrameAt(n)
	if slot.pending == nil {
		slot.pending = &memoEffects{}
		c.memoPendingCount++
	}
	slot.pending.addDep(f)
}

// observeMemo accounts for reading a memoized result without computing it.
func (c *Checker) observeMemo(key memoKey) {
	c.observeSlot(c.memoSlots[key])
}

// observeSlot is observeMemo for the slot of a result, if it has one.
func (c *Checker) observeSlot(slot *memoSlot) {
	if slot == nil {
		return
	}
	if c.reporting {
		c.replayPending(slot)
		return
	}
	if slot.pending != nil {
		if parent := c.currentMemoFrame(); parent != nil {
			parent.addDep(slot.pending)
		}
	}
	if slot.depth != 0 {
		c.memoCycle(slot.depth)
	}
}

func (c *Checker) replayPending(slot *memoSlot) {
	if slot != nil && slot.pending != nil {
		f := slot.pending
		slot.pending = nil
		c.memoPendingCount--
		c.replayMemo(f)
	}
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
	// Reporting starts only from top-level requests, so any frame left at
	// this point was never exited and would absorb unrelated effects.
	debug.Assert(len(c.memoFrames) == 0, "memo frame leaked")
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
