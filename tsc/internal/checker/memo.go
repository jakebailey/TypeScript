package checker

import (
	"iter"
	"maps"
	"slices"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/ast"

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
	marks    []*ast.Symbol // Aliases marked referenced, also among effects; see withReferenceMarks
	deps     []*memoEffects
	parent   *memoEffects // Frame d was most recently added to; avoids duplicate deps
	entry    *ast.Node    // Current node when the computation started, if entered
	entered  bool         // Whether this is the frame of a computation, not just a set of deps
	replayed bool
}

func (f *memoEffects) addDep(d *memoEffects) {
	if f == d || d.parent == f {
		return
	}
	d.parent = f
	f.deps = append(f.deps, d)
}

// covers reports whether replaying f replays deps, checking only the direct
// dependencies of f, and only of a few, to stay cheap.
func (f *memoEffects) covers(deps []*memoEffects) bool {
	if len(deps) > 16 || len(f.deps) > 64 {
		return false
	}
	for _, d := range deps {
		if d != f && !slices.Contains(f.deps, d) {
			return false
		}
	}
	return true
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
		c.observeCell(unsafe.Pointer(m))
	}
	return m.memoValue
}

// set caches a result. When set outside the result's own computation (for
// example, the types of binding elements assigned while resolving the type of
// their declaration), the result depends on the computation setting it.
func (m *memo[T]) set(c *Checker, v T) {
	if c.cycleHead != 0 {
		m.logUndo(c)
	}
	m.memoValue = v
	if !c.reporting {
		c.attachCell(unsafe.Pointer(m))
	}
}

// memoUpdate is set for a result that every computation needing it sets
// again, such as the symbol a property access resolves to. Setting the value
// it already has comes from the same computation, whose effects are already
// retained, so it is only set when it changes.
func memoUpdate[T comparable](c *Checker, m *memo[T], v T) {
	if m.memoValue != v {
		m.set(c, v)
	}
}

// setStable is set for a result that defines the identity of what it
// caches, such as a declared type, or that is part of an entity being
// created, such as the type of a synthetic property. It is never discarded at
// the end of a cycle: discarding an identity would let a second, distinct
// result exist, and an entity can outlive the cycle in caches that are kept
// while having no way to compute its parts again.
func (m *memo[T]) setStable(c *Checker, v T) {
	m.memoValue = v
	if !c.reporting {
		c.attachCell(unsafe.Pointer(m))
	}
}

//go:noinline
func (m *memo[T]) logUndo(c *Checker) {
	old := m.memoValue
	c.cycleUndo = append(c.cycleUndo, cycleUndo{unsafe.Pointer(m), func() { m.memoValue = old }, false})
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
		m.observe(c, k)
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

// observe is the out of line part of get, keeping get cheap enough to inline.
//
//go:noinline
func (m *memoMap[K, V]) observe(c *Checker, k K) {
	c.observeSlot(m.memoSlots[k])
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
	if c.cycleHead != 0 {
		m.logUndo(c, k)
	}
	m.setStable(c, k, v)
}

//go:noinline
func (m *memoMap[K, V]) logUndo(c *Checker, k K) {
	old, ok := m.memoEntries[k]
	c.cycleUndo = append(c.cycleUndo, cycleUndo{nil, func() {
		if ok {
			m.memoEntries[k] = old
		} else {
			delete(m.memoEntries, k)
		}
	}, false})
}

// setStable is set for a result that defines the identity of what it
// caches; see memo.setStable.
func (m *memoMap[K, V]) setStable(c *Checker, k K, v V) {
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
	pushed := c.memoEnterActive() && m.enter(c, k)
	v := fn()
	m.set(c, k, v)
	c.exitMemo(pushed)
	return v
}

// enter starts the computation of the result for k; see enterMemo.
func (m *memoMap[K, V]) enter(c *Checker, k K) bool {
	c.enterInstantiationBudget()
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
	pushed := c.memoEnterActive() && s.m.enter(c, k)
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
	pushed := c.memoEnterActive() && o.done.enter(c)
	o.done.set(c, true)
	fn()
	c.exitMemo(pushed)
}

type memoFrame struct {
	effects *memoEffects // Allocated on first use
	slot    *memoSlot
	entry   *ast.Node // Current node when the computation started
	query   savedCurrentNode
	isQuery bool
}

// memoActive reports whether memoized entry points must track effects. When
// false (a reporting activation with nothing pending), they may skip enterMemo
// and exitMemo entirely.
func (c *Checker) memoActive() bool {
	return !c.reporting || c.memoPendingCount != 0
}

// memoEnterActive reports whether memoized entry points must call enterMemo
// and exitMemo: when tracking effects, or when an instantiation is in
// progress, whose limits each memoized computation starts afresh (see
// enterInstantiationBudget).
func (c *Checker) memoEnterActive() bool {
	return c.memoActive() || len(c.instantiationStack) != 0
}

// enterMemo is called on entry to a memoized function whose result for this
// input is cached in the slot identified by key. It accounts for effects
// recorded by a previous computation of that result, and in a non-reporting
// activation starts a frame recording effects of this computation. The result
// must be passed to exitMemo.
func (c *Checker) enterMemo(key memoKey) bool {
	c.enterInstantiationBudget()
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
	frame := memoFrame{slot: slot, entry: c.currentNode}
	if len(c.memoFrames) == 0 && c.currentNodeResolution == 0 {
		// A query made outside of checking: errors located at the current
		// node are located wherever checking reuses the result, so results
		// they depend on are discarded once the query finishes (see
		// reportAtCurrentNode), and every reuse computes them again.
		frame.query = c.enterCurrentNode(c.currentNode)
		frame.isQuery = true
	}
	c.memoFrames = append(c.memoFrames, frame)
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
	if c.cycleHead != 0 {
		c.logPendingUndo(slot)
	}
	if slot.pending == nil {
		slot.pending = &memoEffects{}
		c.memoPendingCount++
		if c.flushSlots != nil {
			*c.flushSlots = append(*c.flushSlots, slot)
		}
	}
	slot.pending.addDep(f)
}

// observeCell and attachCell are observeMemo and memoAttach for a memo at
// address p. They are kept out of line so that memo reads and writes are
// cheap enough to inline.
//
//go:noinline
func (c *Checker) observeCell(p unsafe.Pointer) {
	c.observeMemo(memoKeyOf((*byte)(p), memoCell))
}

//go:noinline
func (c *Checker) attachCell(p unsafe.Pointer) {
	c.memoAttach(memoKeyOf((*byte)(p), memoCell))
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
	c.exitInstantiationBudget()
	if !pushed {
		return
	}
	slot, f := c.popMemoFrame()
	if f == nil {
		return
	}
	c.retainEffects(slot, f)
}

// exitMemoUncached is exitMemo for a computation that did not cache its
// result. Nothing can reuse the result, so its effects belong only to the
// enclosing computation, and retaining them for the result would grow them
// each time it is computed again.
func (c *Checker) exitMemoUncached(pushed bool) {
	c.exitInstantiationBudget()
	if !pushed {
		return
	}
	slot, f := c.popMemoFrame()
	if f == nil {
		return
	}
	if parent := c.currentMemoFrame(); parent != nil {
		if len(f.effects) == 0 && f.entry == parent.entry {
			// Without effects of its own, and located like the enclosing
			// computation, the computation adds nothing but what it observed.
			for _, d := range f.deps {
				parent.addDep(d)
			}
		} else {
			parent.addDep(f)
		}
	} else if c.flushSlots != nil {
		// withoutReporting replays effects through the results computed.
		c.retainEffects(slot, f)
	}
}

func (c *Checker) popMemoFrame() (*memoSlot, *memoEffects) {
	n := len(c.memoFrames) - 1
	frame := c.memoFrames[n]
	c.memoFrames[n] = memoFrame{}
	c.memoFrames = c.memoFrames[:n]
	slot := frame.slot
	if slot.depth == n+1 {
		slot.depth = 0
	}
	if frame.isQuery {
		c.exitCurrentNode(frame.query)
	}
	return slot, frame.effects
}

// retainEffects retains the effects of a computation that just finished for
// replay by later reuse of its result, and adds them to the enclosing
// computation.
func (c *Checker) retainEffects(slot *memoSlot, f *memoEffects) {
	if slot.pending != nil && len(f.effects) == 0 && slot.pending.covers(f.deps) {
		// Computing the result again, typically to return it from a cache
		// that the computation reads, observed only what computing it
		// before did. Retaining that again would grow the effects of the
		// result each time.
		if parent := c.currentMemoFrame(); parent != nil {
			parent.addDep(slot.pending)
		}
		return
	}
	if c.cycleHead != 0 || c.effectsMarkDepth != 0 {
		c.logPendingUndo(slot)
	}
	if slot.pending != nil {
		f.addDep(slot.pending)
	} else {
		c.memoPendingCount++
		if c.flushSlots != nil {
			*c.flushSlots = append(*c.flushSlots, slot)
		}
	}
	slot.pending = f
	if parent := c.currentMemoFrame(); parent != nil {
		parent.addDep(f)
	}
}

func (c *Checker) replayMemo(f *memoEffects) {
	c.replayFrame(f, nil, false)
}

// replayFrame replays the effects of f and its dependencies. Effects located
// at the current node that a computation started with (see
// reportAtCurrentNode) are located at the node the reuse happens at instead,
// as they would be had the computation happened there, unless the
// computation started at a different node than the computation replaying it,
// which then fixes its location.
func (c *Checker) replayFrame(f *memoEffects, outerEntry *ast.Node, outerEntered bool) {
	if f.replayed {
		return
	}
	f.replayed = true
	saveCurrentNode := c.currentNode
	if f.entered {
		// A computation that started at no node can be located anywhere,
		// like one started at the node it is reused at.
		if outerEntered && f.entry != outerEntry && f.entry != nil {
			c.currentNode = f.entry
		}
		outerEntry, outerEntered = f.entry, true
	}
	for _, e := range f.effects {
		e(c)
	}
	for _, d := range f.deps {
		c.replayFrame(d, outerEntry, outerEntered)
	}
	c.currentNode = saveCurrentNode
	f.effects = nil
	f.deps = nil
}

// currentNodeIsAmbient reports whether the current node is the one the
// innermost computation in progress started with.
func (c *Checker) currentNodeIsAmbient() bool {
	n := len(c.memoFrames) - 1
	return n >= 0 && c.memoFrames[n].entry == c.currentNode
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

// recordReferenceMark records marking an alias referenced, an effect that
// emit applies itself (see withReferenceMarks).
func (c *Checker) recordReferenceMark(symbol *ast.Symbol) {
	if f := c.currentMemoFrame(); f != nil {
		f.effects = append(f.effects, func(c *Checker) { c.markAliasSymbolAsReferenced(symbol) })
		f.marks = append(f.marks, symbol)
	}
}

// withReferenceMarks runs fn as a query, then marks referenced the aliases
// that everything fn computed would mark, which emit relies on to elide
// imports. Unlike checking, it reports nothing, so diagnostics do not depend
// on whether a file was emitted before it was checked. Other effects stay
// pending for checking to replay.
func (c *Checker) withReferenceMarks(fn func()) {
	debug.Assert(!c.reporting && len(c.memoFrames) == 0, "withReferenceMarks inside a computation")
	frame := memoFrame{slot: &memoSlot{}, entry: c.currentNode}
	if c.currentNodeResolution == 0 {
		frame.query = c.enterCurrentNode(c.currentNode)
		frame.isQuery = true
	}
	c.memoFrames = append(c.memoFrames, frame)
	fn()
	_, f := c.popMemoFrame()
	if f == nil {
		return
	}
	var marks []*ast.Symbol
	visited := make(map[*memoEffects]struct{})
	var collect func(f *memoEffects)
	collect = func(f *memoEffects) {
		if _, ok := visited[f]; ok {
			return
		}
		visited[f] = struct{}{}
		marks = append(marks, f.marks...)
		for _, d := range f.deps {
			collect(d)
		}
	}
	collect(f)
	for _, symbol := range marks {
		c.withReferenceMarks(func() { c.referenceAliasSymbol(symbol) })
	}
}

// withoutReporting runs fn in a non-reporting activation from a reporting
// one, then replays the effects of everything fn computed, as if fn had run
// in the reporting activation. Unlike effects applied directly, recorded
// effects can be discarded with the results they belong to.
func (c *Checker) withoutReporting(fn func()) {
	debug.Assert(c.reporting && c.flushSlots == nil, "withoutReporting outside a reporting activation")
	saveFrames := c.memoFrames
	var slots []*memoSlot
	c.reporting = false
	c.memoFrames = nil
	c.flushSlots = &slots
	fn()
	c.flushSlots = nil
	c.reporting = true
	c.memoFrames = saveFrames
	// Replaying is in order of computation, so effects apply in the order
	// they would have in the reporting activation.
	// They also happened at the current node, so only the effects of
	// computations started there are located at it.
	for _, slot := range slots {
		if slot.pending != nil {
			f := slot.pending
			slot.pending = nil
			c.memoPendingCount--
			c.replayFrame(f, c.currentNode, true)
		}
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
		if i < c.effectsMarkDepth-1 && f != outer && outer.parent != f {
			// The frame is in progress below the mark, so rolling back to
			// the mark must also roll back this dependency.
			c.logDepsUndo(f)
		}
		f.addDep(outer)
	}
}

// logDepsUndo logs the dependencies of a computation in progress below the
// innermost effects mark, which are about to grow, so discardCycleSince can
// restore them.
//
//go:noinline
func (c *Checker) logDepsUndo(f *memoEffects) {
	n := len(f.deps)
	c.cycleUndo = append(c.cycleUndo, cycleUndo{effects: true, restore: func() {
		if len(f.deps) > n {
			f.truncateDeps(n)
		}
	}})
}

func (c *Checker) memoFrameAt(i int) *memoEffects {
	f := c.memoFrames[i].effects
	if f == nil {
		f = &memoEffects{entry: c.memoFrames[i].entry, entered: true}
		c.memoFrames[i].effects = f
	}
	return f
}

// Cycles
//
// A cached result must be the result of computing it from scratch, so that
// it does not depend on which computation happened to request it first. A
// computation that observes another computation still in progress (a
// circularity) sees a provisional state, so results cached from the moment
// the cycle is detected until the outermost computation of the cycle (its
// head) finishes may differ from computing them from scratch. Those results
// are discarded when the head finishes, and are recomputed when next
// requested. The computations that were in progress when the cycle was
// detected (its participants) are kept: each of them reaches the cycle when
// computed from scratch too, so each result is the fallback for a circularity
// whichever participant is computed first.

type cycleUndo struct {
	memo    unsafe.Pointer // The memo restored, or nil
	restore func()
	effects bool // Restores the effects of a result rather than the result
}

// logFieldUndo records how to discard a result cached in a plain field while
// a cycle is in progress. Callers check cycleHead first so that the closure
// is only allocated when needed.
func (c *Checker) logFieldUndo(reset func()) {
	c.cycleUndo = append(c.cycleUndo, cycleUndo{nil, reset, false})
}

// endCycle discards the results cached since the outermost cycle in progress
// was detected, except those of its participants.
func (c *Checker) endCycle() {
	undo := c.cycleUndo
	participants := c.cycleParticipants
	c.cycleHead = 0
	c.cycleUndo = nil
	c.cycleParticipants = nil
	keep := make(map[unsafe.Pointer]struct{}, len(participants))
	for i := range participants {
		if p := c.typeResolutionMemo(&participants[i]); p != nil {
			keep[p] = struct{}{}
		}
	}
	for _, u := range slices.Backward(undo) {
		if u.effects {
			// A result recomputed after the cycle also replays the effects
			// of the computation discarded here.
			continue
		}
		if _, ok := keep[u.memo]; ok && u.memo != nil {
			continue
		}
		u.restore()
	}
}

// discardCycleSince discards the results cached since the given positions
// of the cycle undo log and participants, restoring the cycle head in effect
// then, so a computation in a cycle can run again from the same state. The
// effects of computing the discarded results are discarded with them, and
// those recorded by the computation in progress are rolled back to the given
// mark, as if the discarded computations had never run.
func (c *Checker) discardCycleSince(undoStart int, participantsStart int, cycleHead int, mark memoEffectsMark) {
	undoStart = min(undoStart, len(c.cycleUndo))
	for i := len(c.cycleUndo) - 1; i >= undoStart; i-- {
		c.cycleUndo[i].restore()
	}
	c.cycleUndo = c.cycleUndo[:undoStart]
	c.cycleParticipants = c.cycleParticipants[:min(participantsStart, len(c.cycleParticipants))]
	c.cycleHead = cycleHead
	c.discardEffectsSince(mark)
}

// memoEffectsMark is a position in the effects recorded by a computation in
// progress, to which they can be rolled back.
type memoEffectsMark struct {
	frame      *memoEffects
	effects    int
	deps       int
	frameIndex int
}

// markEffects returns the position of the effects recorded so far by the
// innermost computation in progress, which must have a memo frame.
func (c *Checker) markEffects() memoEffectsMark {
	n := len(c.memoFrames) - 1
	f := c.memoFrameAt(n)
	return memoEffectsMark{frame: f, effects: len(f.effects), deps: len(f.deps), frameIndex: n}
}

func (c *Checker) discardEffectsSince(m memoEffectsMark) {
	debug.Assert(len(c.memoFrames)-1 == m.frameIndex && c.memoFrames[m.frameIndex].effects == m.frame, "effects discarded outside their computation")
	f := m.frame
	clear(f.effects[m.effects:])
	f.effects = f.effects[:m.effects]
	f.truncateDeps(m.deps)
}

func (f *memoEffects) truncateDeps(n int) {
	for _, d := range f.deps[n:] {
		if d.parent == f {
			d.parent = nil
		}
	}
	clear(f.deps[n:])
	f.deps = f.deps[:n]
}

// logPendingUndo logs the effects of the result in the slot, which are about
// to change, so discardCycleSince can restore them.
//
//go:noinline
func (c *Checker) logPendingUndo(slot *memoSlot) {
	old := slot.pending
	n := 0
	if old != nil {
		n = len(old.deps)
	}
	c.cycleUndo = append(c.cycleUndo, cycleUndo{effects: true, restore: func() {
		switch {
		case old == nil && slot.pending != nil:
			c.memoPendingCount--
		case old != nil && slot.pending == nil:
			c.memoPendingCount++
		}
		slot.pending = old
		if old != nil && len(old.deps) > n {
			old.truncateDeps(n)
		}
	}})
}

// typeResolutionMemo returns the address of the memo caching the result of
// a type resolution, or nil if it is not cached in a memo.
func (c *Checker) typeResolutionMemo(r *TypeResolution) unsafe.Pointer {
	switch r.propertyName {
	case TypeSystemPropertyNameType:
		return unsafe.Pointer(&c.valueSymbolLinks.Get(r.target.(*ast.Symbol)).resolvedType)
	case TypeSystemPropertyNameDeclaredType:
		return unsafe.Pointer(&c.typeAliasLinks.Get(r.target.(*ast.Symbol)).declaredType)
	case TypeSystemPropertyNameResolvedBaseTypes:
		return unsafe.Pointer(&r.target.(*Type).AsInterfaceType().resolvedBaseTypes)
	case TypeSystemPropertyNameResolvedBaseConstructorType:
		return unsafe.Pointer(&r.target.(*Type).AsInterfaceType().resolvedBaseConstructorType)
	case TypeSystemPropertyNameResolvedReturnType:
		return unsafe.Pointer(&r.target.(*Signature).resolvedReturnType)
	case TypeSystemPropertyNameResolvedBaseConstraint:
		return unsafe.Pointer(&r.target.(*Type).AsConstrainedType().resolvedBaseConstraint)
	case TypeSystemPropertyNameWriteType:
		if extra := c.valueSymbolLinks.Get(r.target.(*ast.Symbol)).extra; extra != nil {
			return unsafe.Pointer(&extra.writeType)
		}
	case TypeSystemPropertyNameAliasTarget:
		return unsafe.Pointer(&c.aliasSymbolLinks.Get(r.target.(*ast.Symbol)).aliasTarget)
	case TypeSystemPropertyNameComputedPropertyName:
		return unsafe.Pointer(&c.typeNodeLinks.Get(r.target.(*ast.Node)).resolvedType)
	case TypeSystemPropertyNameVariances:
		return unsafe.Pointer(&c.varianceLinks.Get(r.target.(*ast.Symbol)).variances)
	}
	return nil
}

// memoizeGlobal memoizes the resolution of a global, replaying effects of
// the resolution wherever the cached result is observed.
func memoizeGlobal[T any](c *Checker, fn func() T) func() T {
	cell := &struct{ resolve func() T }{resolve: core.Memoize(fn)}
	return func() T {
		if c.memoEnterActive() {
			pushed := c.enterMemo(memoKeyOf(cell, memoGlobal))
			result := cell.resolve()
			c.exitMemo(pushed)
			return result
		}
		return cell.resolve()
	}
}

// Instantiation limits
//
// Instantiation stops with an error when nested too deeply or after too
// many instantiations, as the instantiation is then very likely infinite.
// For the result not to depend on what was cached before, each memoized
// computation (including each cached comparison of two types) measures
// depth from where it started, whether or not it reports errors, and a
// reused result adds no depth. Chains that start a new memoized computation at every
// level are only stopped by a much larger limit on the whole outermost
// instantiation, which is only reached by runaway instantiations, and which
// does depend on what was cached before.
//
// TODO: The limit on the number of instantiations still counts the work of
// the current statement or expression rather than of the result, so whether
// it is reached depends on what was cached before. Counting per memoized
// computation is pure, but most runaway instantiations cross a new memoized
// computation at every level and so are then only stopped by a far larger
// limit on the whole outermost instantiation. Instead, each cached result
// could store a mergeable distinct-count sketch (e.g. HyperLogLog) of the
// computations it needed, merged into the current sketch on reuse,
// approximating the work to compute a result from scratch. Summing stored
// costs instead overcounts shared results exponentially. Sketches need
// computation IDs that do not depend on creation order, which type IDs
// currently do.

const (
	instantiationDepthLimit      = 100
	instantiationTotalDepthLimit = 1000
	instantiationCountLimit      = 5_000_000
)

// enterInstantiationBudget starts the depth limit of a memoized computation.
// It is called on entry to every memoized computation, and must be matched
// by exitInstantiationBudget at the same instantiation depth.
func (c *Checker) enterInstantiationBudget() {
	if len(c.instantiationStack) != 0 {
		c.instantiationDepthBases = append(c.instantiationDepthBases, c.instantiationDepthBase)
		c.instantiationDepthBase = len(c.instantiationStack)
	}
}

func (c *Checker) exitInstantiationBudget() {
	if len(c.instantiationStack) != 0 {
		n := len(c.instantiationDepthBases) - 1
		c.instantiationDepthBase = c.instantiationDepthBases[n]
		c.instantiationDepthBases = c.instantiationDepthBases[:n]
	}
}

// instantiationLimitReached reports whether a further instantiation would
// exceed a limit.
func (c *Checker) instantiationLimitReached() bool {
	return c.instantiationDepthLimitReached() || c.instantiationCount >= instantiationCountLimit
}

func (c *Checker) instantiationDepthLimitReached() bool {
	depth := len(c.instantiationStack)
	return depth-c.instantiationDepthBase == instantiationDepthLimit || depth == instantiationTotalDepthLimit
}
