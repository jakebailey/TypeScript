package checker

import (
	"iter"
	"maps"
	"slices"
	"unsafe"

	"github.com/microsoft/TypeScript/tsc/internal/ast"

	"github.com/microsoft/TypeScript/tsc/internal/debug"
	"github.com/microsoft/TypeScript/tsc/internal/diagnostics"
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
	memoTypePredicate
	memoDeclaredMembers
	memoCouldContainTypeVariables
	memoGenericObjectFlags
	memoUnknownLikeUnion
	memoNeverIntersection
	memoDiscriminant
	memoResolvedMembers // followed by one field per MembersOrExportsResolutionKind
	memoSymbolExports   = memoResolvedMembers + memoField(MembersOrExportsResolutionKindResolvedExports)
	memoSymbolMembers   = memoResolvedMembers + memoField(MembersOrExportsResolutionKindResolvedMembers)
	memoFieldCount      = memoResolvedMembers + 2
)

// A reentry says what it means to re-enter a computation in progress, which
// can only observe whatever the computation has published so far.
type reentry uint8

const (
	// What re-entry observes is incomplete, so the results computed while
	// the outer computation is in progress are discarded when it finishes,
	// and computed again from its result. This is the default.
	reentryTaint reentry = iota
	// The computation publishes a partial result before it finishes, and
	// re-entry observes that partial result. Results derived from it may
	// differ depending on which computation starts first.
	reentryPartial
)

// memoReentry is the re-entry policy of each kind of computation.
var memoReentry = [memoFieldCount]reentry{
	// memoCompute keeps the result that a re-entrant computation cached,
	// and many cells publish an in-progress marker for re-entry to read.
	memoCell: reentryPartial,
	// Many kinds of types make members available before they are complete.
	memoStructuredMembers: reentryPartial,
}

// memoKey identifies a memoized result: the address of the entity owning the
// cache slot (an *ast.Node, *ast.Symbol, *Type, or *Signature) shifted left by
// eight bits, plus the memoField of the slot. Packing into an integer keeps map
// operations on the fast path. Entities live as long as the program or the
// checker store, so an address cannot be reused while its keys exist.
type memoKey uint64

func (k memoKey) field() memoField {
	return memoField(k)
}

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
	pending *memoEffects  // Effects not yet replayed by a reporting activation
	reports []*nodeReport // Errors the computation reported at the current node; see setReported
	sketch  *workSketch   // Instantiations the computation needed; see "Work sketches"
	height  int32         // How much deeper than its start the computation nested instantiations
}

// memo is a cached result. Cached checker state must live in a memo so that
// every read accounts for the effects of the computation that produced it;
// see the package comment above. The value is only accessible through
// methods. The memo state of a result is kept in the store's slot map, keyed
// by the address of the memo, except for its height.
type memo[T any] struct {
	memoHeader
	memoValue T
}

// memoHeader is the part of a memo independent of its type, which also
// identifies it. It holds the height of the result (see "Instantiation
// limits"), since most reads of a result need nothing else of its memo state,
// and most results have none.
type memoHeader struct {
	_          memoNoCopy
	memoHeight int32
}

// memoNoCopy makes go vet reject copies of a memo, which would bypass it.
type memoNoCopy struct{}

func (*memoNoCopy) Lock()   {}
func (*memoNoCopy) Unlock() {}

func (h *memoHeader) key() memoKey {
	return memoKeyOf(h, memoCell)
}

// get returns the cached result, accounting for the effects of computing it.
func (m *memo[T]) get(c *Checker) T {
	if c.memoActive() || m.memoHeight != 0 && c.observingWork() {
		c.observeCell(&m.memoHeader)
	}
	return m.memoValue
}

// set caches a result. When set outside the result's own computation (for
// example, the types of binding elements assigned while resolving the type of
// their declaration), the result depends on the computation setting it.
func (m *memo[T]) set(c *Checker, v T) {
	if c.logResults() {
		m.logUndo(c)
	}
	m.memoValue = v
	c.linkWork(&m.memoHeader)
	if !c.reporting {
		c.attachCell(&m.memoHeader)
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
//
// Unlike set, it does not take on the work of the computation setting it
// (see linkWork): an identity is reached only through the result that
// created it, whose work is already accounted for, and the setting
// computation's later work, which would otherwise count, is unrelated.
func (m *memo[T]) setStable(c *Checker, v T) {
	if c.aborting {
		// Nothing computed while aborting is kept; see "Instantiation limits".
		m.logUndo(c)
	}
	m.memoValue = v
	if !c.reporting {
		c.attachCell(&m.memoHeader)
	}
}

// setReported is setStable for a result whose computation collected errors
// reported at the current node, as returned by endNodeReports. Every node
// reusing the result reports them again, as it would had it computed the
// result itself. If computing the result elsewhere may not report them (see
// endNodeReports), it is instead discarded with other results cached for the
// current node.
func (m *memo[T]) setReported(c *Checker, v T, r nodeReports) {
	if r.cyclic {
		if c.logResults() {
			m.logUndo(c)
		}
		r.list = nil
	}
	m.setStable(c, v)
	if c.aborting {
		return
	}
	if r.list != nil || c.memoReportSlots != 0 && c.memoSlots[m.key()] != nil {
		c.memoSlot(m.key()).setReports(c, r.list)
	}
}

//go:noinline
func (m *memo[T]) logUndo(c *Checker) {
	old := m.memoValue
	c.journal = append(c.journal, undoEntry{&m.memoHeader, func() { m.memoValue = old }, false})
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
func (m *memo[T]) enter(c *Checker) {
	c.enterWork(m.key(), &m.memoHeader, nil, 0)
	c.enterMemoSlot(m.key())
}

// memoCompute returns the result cached in m, computing it in a computation
// of its own if there is none. A result cached while computing it, by a
// computation that re-entered it, is kept.
func (c *Checker) memoCompute[T comparable](m *memo[T], compute func() T) T {
	var zero T
	if v := m.get(c); v != zero {
		return v
	}
	m.enter(c)
	v := compute()
	if cur := m.peek(); cur != zero {
		v = cur
	} else {
		m.set(c, v)
	}
	c.exitMemo()
	return v
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
	workKeys    []K // Keys of computations in progress without a slot yet
}

// get returns the cached result for k, accounting for the effects of
// computing it.
func (m *memoMap[K, V]) get(c *Checker, k K) (V, bool) {
	if m.memoSlots != nil && c.observing() {
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
	if c.observeSlot(m.memoSlots[k]) {
		c.memoCycle(m.computingFrame(c, k))
	}
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
	if c.logResults() {
		m.logUndo(c, k)
	}
	m.setStable(c, k, v)
	if len(c.frames) > 1 && !c.aborting && !m.isOwnWork(c, k) {
		// Like memo.set; see linkWork.
		if w := c.workSoFar(); w.sketch != nil || w.height != 0 || m.memoSlots[k] != nil {
			m.slot(k).linkWork(w)
		}
	}
}

//go:noinline
func (m *memoMap[K, V]) logUndo(c *Checker, k K) {
	old, ok := m.memoEntries[k]
	c.journal = append(c.journal, undoEntry{nil, func() {
		if ok {
			m.memoEntries[k] = old
		} else {
			delete(m.memoEntries, k)
		}
	}, false})
}

// setReported is setStable for an entry; see memo.setReported.
func (m *memoMap[K, V]) setReported(c *Checker, k K, v V, r nodeReports) {
	if r.cyclic {
		if c.logResults() {
			m.logUndo(c, k)
		}
		r.list = nil
	}
	m.setStable(c, k, v)
	if c.aborting {
		return
	}
	if r.list != nil || c.memoReportSlots != 0 && m.memoSlots[k] != nil {
		m.slot(k).setReports(c, r.list)
	}
}

// setStable is set for a result that defines the identity of what it
// caches; see memo.setStable.
func (m *memoMap[K, V]) setStable(c *Checker, k K, v V) {
	if m.memoEntries == nil {
		m.memoEntries = make(map[K]V)
	}
	m.memoEntries[k] = v
	if c.memoTop != 0 {
		c.attachSlot(m.slot(k))
	}
}

// setWithWork is set for a result whose computation needed the work w,
// rather than that of the current work frame.
func (m *memoMap[K, V]) setWithWork(c *Checker, k K, v V, w resultWork) {
	if c.logResults() {
		m.logUndo(c, k)
	}
	if m.memoEntries == nil {
		m.memoEntries = make(map[K]V)
	}
	m.memoEntries[k] = v
	if w.sketch != nil || w.height != 0 {
		m.slot(k).linkWork(w)
	}
	if c.memoTop != 0 {
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
	m.enter(c, k)
	v := fn()
	m.set(c, k, v)
	c.exitMemo()
	return v
}

// enter starts the computation of the result for k; see enterMemo.
func (m *memoMap[K, V]) enter(c *Checker, k K) {
	if c.observingWork() {
		// Entering a computed result reads it.
		if slot := m.memoSlots[k]; slot != nil {
			c.observeWork(slot.work())
		}
	}
	outer := m.computingFrame(c, k)
	if outer != 0 {
		// Re-entering an entry taints the cycle; see reentryTaint.
		c.taintFrameCycle(outer - 1)
	}
	m.workKeys = append(m.workKeys, k)
	c.enterWork(0, nil, m, len(m.workKeys)-1)
	if c.reporting {
		if c.memoPendingCount != 0 && m.memoSlots != nil {
			c.replayPending(m.memoSlots[k])
		}
		return
	}
	c.enterSlot(m.slot(k), outer)
}

// computingFrame returns 1 + the index in frames of the computation of the
// entry for k in progress, or 0.
func (m *memoMap[K, V]) computingFrame(c *Checker, k K) int {
	i := slices.Index(m.workKeys, k)
	if i < 0 {
		return 0
	}
	for j := len(c.frames) - 1; j >= 0; j-- {
		if f := &c.frames[j]; f.source == workSlotSource(m) && int(f.index) == i {
			return j + 1
		}
	}
	panic("entry in progress without a frame")
}

// isOwnWork reports whether the current work frame computes the entry for k.
func (m *memoMap[K, V]) isOwnWork(c *Checker, k K) bool {
	f := c.currentWork()
	return f.source == workSlotSource(m) && m.workKeys[f.index] == k
}

func (m *memoMap[K, V]) workSlot(i int) *memoSlot {
	return m.slot(m.workKeys[i])
}

func (m *memoMap[K, V]) popWork(i int) {
	clear(m.workKeys[i:])
	m.workKeys = m.workKeys[:i]
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
	s.m.enter(c, k)
	s.m.set(c, k, struct{}{})
	fn()
	c.exitMemo()
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
	o.done.enter(c)
	o.done.set(c, true)
	fn()
	c.exitMemo()
}

// memoActive reports whether reading a cached result must account for its
// effects: false in a reporting activation with nothing pending.
func (c *Checker) memoActive() bool {
	return !c.reporting || c.memoPendingCount|c.memoReportSlots != 0
}

// observing reports whether reading a cached result must account for its
// effects or its work.
func (c *Checker) observing() bool {
	return c.memoActive() || c.observingWork()
}

// enterMemo is called on entry to a memoized function whose result for this
// input is cached in the slot identified by key. It accounts for effects
// recorded by a previous computation of that result, and starts a frame for
// this computation, which in a non-reporting activation records its effects.
// It must be matched by exitMemo.
func (c *Checker) enterMemo(key memoKey) {
	c.enterWork(key, nil, nil, 0)
	c.enterMemoSlot(key)
}

// enterMemoSlot is the part of enterMemo for the memo state of the result.
func (c *Checker) enterMemoSlot(key memoKey) {
	if c.reporting {
		if c.memoPendingCount != 0 {
			c.replayPending(c.memoSlots[key])
		}
		return
	}
	outer := c.computingFrame(key)
	if outer == len(c.frames) {
		outer = 0
	}
	c.enterSlot(c.memoSlot(key), outer)
}

// enterSlot is enterMemo for the slot of a result, in a non-reporting
// activation. outer is 1 + the index in frames of the computation of the
// same result in progress, if this one re-enters it, or 0.
func (c *Checker) enterSlot(slot *memoSlot, outer int) {
	if slot.pending != nil {
		if parent := c.currentMemoFrame(); parent != nil {
			parent.addDep(slot.pending)
		}
	}
	c.pushMemo(slot)
	// Re-entering a computation in progress: the inner result is derived
	// from whatever in-progress marker the outer computation left behind.
	c.memoCycle(outer)
}

// pushMemo starts recording the effects of the current frame, which computes
// the result in slot.
func (c *Checker) pushMemo(slot *memoSlot) {
	n := len(c.frames) - 1
	f := &c.frames[n]
	f.memo = true
	f.prevMemo = int32(c.memoTop)
	f.result = slot
	f.entry = c.currentNode
	if c.memoTop == 0 && c.currentNodeResolution == 0 {
		// A query made outside of checking: errors located at the current
		// node are located wherever checking reuses the result, so results
		// they depend on are discarded once the query finishes (see
		// reportAtCurrentNode), and every reuse computes them again. The
		// node's resolution is entered above the computation's frame.
		f.isQuery = true
		query := c.enterCurrentNode(c.currentNode)
		c.frames[n].query = query
	}
	c.memoTop = n + 1
}

// popMemo stops recording the effects of the current frame, returning it if
// it recorded them.
func (c *Checker) popMemo() *frame {
	n := len(c.frames) - 1
	if c.frames[n].resolving {
		n--
		debug.Assert(c.frames[n].isQuery, "resolution left on the stack")
		c.exitCurrentNode(c.frames[n].query)
	}
	f := &c.frames[n]
	if !f.memo {
		return nil
	}
	c.memoTop = int(f.prevMemo)
	return f
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
	if c.memoTop == 0 {
		return
	}
	c.attachSlot(c.memoSlot(key))
}

// attachSlot is memoAttach for the slot of a result.
func (c *Checker) attachSlot(slot *memoSlot) {
	n := c.memoTop - 1
	if n < 0 {
		return
	}
	if c.frames[n].result == slot {
		return
	}
	f := c.memoFrameAt(n)
	if c.logResults() {
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
func (c *Checker) observeCell(h *memoHeader) {
	if !c.memoActive() {
		c.observeCellWork(h)
		return
	}
	if h.memoHeight != 0 {
		c.observeHeight(h.memoHeight)
	}
	if c.observeSlot(c.memoSlots[h.key()]) {
		c.memoCycle(c.computingFrame(h.key()))
	}
}

//go:noinline
func (c *Checker) attachCell(h *memoHeader) {
	c.memoAttach(h.key())
}

// observeMemo accounts for reading a memoized result without computing it.
func (c *Checker) observeMemo(key memoKey) {
	if !c.memoActive() && !c.mayHaveWork(key) {
		// Only the work of the result is accounted for, and it has none.
		return
	}
	if c.observeSlot(c.memoSlots[key]) {
		c.memoCycle(c.computingFrame(key))
	}
}

// observeSlot is observeMemo for the slot of a result, if it has one. It
// reports whether the result is read in a non-reporting activation, in
// which case reading a computation in progress is left to the caller; see
// memoCycle.
func (c *Checker) observeSlot(slot *memoSlot) bool {
	if slot == nil {
		return false
	}
	c.observeWork(slot.work())
	if slot.reports != nil {
		c.replayNodeReports(slot.reports)
	}
	if c.reporting {
		c.replayPending(slot)
		return false
	}
	if slot.pending != nil {
		if parent := c.currentMemoFrame(); parent != nil {
			parent.addDep(slot.pending)
		}
	}
	return true
}

// A nodeReport is an error reported at the current node; see
// reportAtCurrentNode. Replaying it reports the same *nodeReport, so that
// reports collected more than once can be told apart from distinct ones.
type nodeReport struct {
	report func(c *Checker, node *ast.Node)
}

type nodeReportsMark struct {
	reports int
	cyclic  bool
}

// nodeReports are the errors a computation reported at the current node.
type nodeReports struct {
	list   []*nodeReport
	cyclic bool // Whether the computation observed a cycle, so computing it elsewhere may not report them
}

// startNodeReports starts collecting the errors a computation reports at the
// current node, to be passed to endNodeReports when it finishes.
func (c *Checker) startNodeReports() nodeReportsMark {
	c.nodeReportCollectors++
	mark := nodeReportsMark{len(c.nodeReports), c.nodeReportsCyclic}
	c.nodeReportsCyclic = false
	return mark
}

// endNodeReports returns the errors reported at the current node since mark.
func (c *Checker) endNodeReports(mark nodeReportsMark) nodeReports {
	var r nodeReports
	for _, report := range c.nodeReports[mark.reports:] {
		// Reusing a result more than once reports its errors each time.
		if !slices.Contains(r.list, report) {
			r.list = append(r.list, report)
		}
	}
	// Which errors a cycle reports depends on where it is entered, so they
	// are only reported where the result is computed.
	r.cyclic = r.list != nil && c.nodeReportsCyclic
	c.nodeReportsCyclic = c.nodeReportsCyclic || mark.cyclic
	c.nodeReportCollectors--
	if c.nodeReportCollectors == 0 {
		clear(c.nodeReports)
		c.nodeReports = c.nodeReports[:0]
		c.nodeReportsCyclic = false
	}
	return r
}

func (s *memoSlot) setReports(c *Checker, reports []*nodeReport) {
	switch {
	case s.reports == nil && reports != nil:
		c.memoReportSlots++
	case s.reports != nil && reports == nil:
		c.memoReportSlots--
	}
	s.reports = reports
}

// replayNodeReports reports at the current node the errors that computing a
// reused result reported at the node it was computed for.
//
//go:noinline
func (c *Checker) replayNodeReports(reports []*nodeReport) {
	for _, r := range reports {
		c.reportNode(r)
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
func (c *Checker) exitMemo() {
	if f := c.popMemo(); f != nil && f.effects != nil {
		c.retainEffects(f.result, f.effects)
	}
	c.exitWork()
}

// exitMemoUncached is exitMemo for a computation that did not cache its
// result. Nothing can reuse the result, so its effects belong only to the
// enclosing computation, and retaining them for the result would grow them
// each time it is computed again.
func (c *Checker) exitMemoUncached() {
	defer c.exitWork()
	frame := c.popMemo()
	c.forgetWork()
	if frame == nil || frame.effects == nil {
		return
	}
	slot, f := frame.result, frame.effects
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
	if c.logEffects() {
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
	n := c.memoTop - 1
	return n >= 0 && c.frames[n].entry == c.currentNode
}

func (c *Checker) currentMemoFrame() *memoEffects {
	if c.memoTop == 0 {
		return nil
	}
	return c.memoFrameAt(c.memoTop - 1)
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
	debug.Assert(!c.reporting && c.memoTop == 0, "withReferenceMarks inside a computation")
	c.enterNodeWork()
	c.pushMemo(&memoSlot{})
	fn()
	f := c.popMemo().effects
	c.exitWork()
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
	saveMemoTop := c.memoTop
	var slots []*memoSlot
	c.reporting = false
	c.memoTop = 0
	c.flushSlots = &slots
	fn()
	c.flushSlots = nil
	c.reporting = true
	c.memoTop = saveMemoTop
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
	debug.Assert(c.memoTop == 0, "memo frame leaked")
	saveReporting := c.reporting
	c.reporting = true
	fn()
	c.reporting = saveReporting
}

// memoCycle is called when a computation observes another computation that is
// still in progress, whose frame is at index depth-1. Every
// computation started since then produced its result from the incomplete
// state of that computation, so reusing any of them must also replay the
// effects of the whole cycle (for example, its circularity error).
func (c *Checker) memoCycle(depth int) {
	if depth == 0 || !c.frames[depth-1].memo {
		// The outer computation does not record effects.
		return
	}
	outer := c.memoFrameAt(depth - 1)
	for i := depth; i < len(c.frames); i++ {
		if !c.frames[i].memo {
			continue
		}
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
// innermost effects mark, which are about to grow, so rollBackTo can
// restore them.
//
//go:noinline
func (c *Checker) logDepsUndo(f *memoEffects) {
	n := len(f.deps)
	c.journal = append(c.journal, undoEntry{effects: true, restore: func() {
		if len(f.deps) > n {
			f.truncateDeps(n)
		}
	}})
}

func (c *Checker) memoFrameAt(i int) *memoEffects {
	f := c.frames[i].effects
	if f == nil {
		f = &memoEffects{entry: c.frames[i].entry, entered: true}
		c.frames[i].effects = f
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

// Journal
//
// Some computations run provisionally, and may have to be undone as if they
// had never run: those in a cycle (see above), and those a computation that
// retries from a mark of its effects runs before retrying (see markEffects).
// While any of them is in progress, what they cache is logged in the
// journal, from which each rolls back to the position where it started.
// The journal is cleared once nothing in progress can roll back.

type undoEntry struct {
	memo    *memoHeader // The memo restored, or nil
	restore func()
	effects bool // Restores the effects of a result rather than the result
}

// logResults reports whether caching a result must be logged in the
// journal.
func (c *Checker) logResults() bool {
	return c.cycleHead != 0 || len(c.instantiationStack) != 0 || c.aborting
}

// logEffects reports whether changing the effects of a result must be
// logged in the journal.
func (c *Checker) logEffects() bool {
	return c.logResults() || c.effectsMarkDepth != 0
}

// rollBack restores everything logged in the journal since position start.
func (c *Checker) rollBack(start int) {
	start = min(start, len(c.journal))
	for i := len(c.journal) - 1; i >= start; i-- {
		c.journal[i].restore()
	}
	clear(c.journal[start:])
	c.journal = c.journal[:start]
}

// trimJournal clears the journal if nothing in progress can roll back.
func (c *Checker) trimJournal() {
	if !c.logEffects() && len(c.journal) != 0 {
		clear(c.journal)
		c.journal = c.journal[:0]
	}
}

// A keyed cell is a cached result held outside a memo, so that it costs no
// space of its own: bits of a flags word shared with other results, or a
// field of storage allocated only when needed. It is accounted for like a
// memo under a key naming its entity and field. Readers finding it cached
// call observeKeyed; it is computed between enterMemo and exitMemo for the
// key, and cached with setKeyedBits, or with setKeyed after logging how to
// discard it with logKeyedUndo.

// observeKeyed accounts for reading the keyed cell identified by key.
func (c *Checker) observeKeyed(key memoKey) {
	if c.observing() {
		c.observeMemo(key)
	}
}

// setKeyed accounts for caching the result of the keyed cell identified by
// key, as memo.set does.
func (c *Checker) setKeyed(key memoKey) {
	if !c.reporting {
		c.memoAttach(key)
	}
}

// logKeyedUndo logs how to discard the result of a keyed cell in the
// journal; the caller checks logResults so that the closure is only
// allocated when needed.
//
//go:noinline
func (c *Checker) logKeyedUndo(restore func()) {
	c.journal = append(c.journal, undoEntry{nil, restore, false})
}

// objectFlagsCell returns the flags under value of t, a keyed cell whose
// result is cached once the computed flag is set, computing them in a
// computation of their own if needed. With inProgress, the computed flag is
// set while computing, so that a computation re-entering it finds no flags.
func (c *Checker) objectFlagsCell(t *Type, field memoField, computed ObjectFlags, value ObjectFlags, inProgress bool, compute func() ObjectFlags) ObjectFlags {
	key := memoKeyOf(t, field)
	if t.objectFlags&computed != 0 {
		c.observeKeyed(key)
		return t.objectFlags & value
	}
	c.enterMemo(key)
	if inProgress {
		setKeyedBits(c, key, &t.objectFlags, computed|value, computed)
	}
	result := compute() & value
	if inProgress || t.objectFlags&computed == 0 {
		setKeyedBits(c, key, &t.objectFlags, computed|value, computed|result)
	} else {
		result = t.objectFlags & value
	}
	c.exitMemo()
	return result
}

// setKeyedBits caches the bits of *p under mask as the result of the keyed
// cell identified by key.
func setKeyedBits[T ~uint32](c *Checker, key memoKey, p *T, mask T, bits T) {
	if c.logResults() {
		old := *p & mask
		c.logKeyedUndo(func() { *p = *p&^mask | old })
	}
	*p = *p&^mask | bits&mask
	c.setKeyed(key)
}

// endCycle discards the results cached since the outermost cycle in progress
// was detected, except those of its participants.
func (c *Checker) endCycle() {
	mark := min(c.cycleMark, len(c.journal))
	undo := c.journal[mark:]
	participants := c.cycleParticipants
	c.cycleHead = 0
	c.cycleParticipants = nil
	keep := make(map[*memoHeader]struct{}, len(participants))
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
	clear(undo)
	c.journal = c.journal[:mark]
	c.trimJournal()
}

// raiseCycleHead makes the resolution at index head-1 part of the cycle in
// progress, starting one if there is none.
func (c *Checker) raiseCycleHead(head int) {
	if c.cycleHead == 0 {
		c.cycleMark = len(c.journal)
		c.cycleHead = head
	} else if head < c.cycleHead {
		c.cycleHead = head
	}
}

// A rollbackMark is a position in the computation in progress to which it
// can roll back, so that it can run again from the same state, as if what it
// computed since had never run. From markRollback to releaseRollback, the
// journal logs all that may have to be rolled back.
type rollbackMark struct {
	journal          int
	participants     int
	cycleHead        int
	effects          memoEffectsMark
	effectsMarkDepth int // Checker.effectsMarkDepth to restore on release
}

func (c *Checker) markRollback() rollbackMark {
	m := rollbackMark{
		journal:          len(c.journal),
		participants:     len(c.cycleParticipants),
		cycleHead:        c.cycleHead,
		effects:          c.markEffects(),
		effectsMarkDepth: c.effectsMarkDepth,
	}
	if m.effects.frameIndex >= 0 {
		c.effectsMarkDepth = m.effects.frameIndex + 1
	}
	return m
}

// rollBackTo discards the results cached since m, with the effects of
// computing them, restoring the cycle in progress then, and rolls back the
// effects recorded by the computation in progress.
func (c *Checker) rollBackTo(m rollbackMark) {
	c.rollBack(m.journal)
	c.cycleParticipants = c.cycleParticipants[:min(m.participants, len(c.cycleParticipants))]
	c.cycleHead = m.cycleHead
	c.discardEffectsSince(m.effects)
}

// releaseRollback ends the period during which the computation in progress
// can roll back to m.
func (c *Checker) releaseRollback(m rollbackMark) {
	c.effectsMarkDepth = m.effectsMarkDepth
	c.trimJournal()
}

// memoEffectsMark is a position in the effects recorded by a computation in
// progress, to which they can be rolled back.
type memoEffectsMark struct {
	effects    int
	deps       int
	frameIndex int // Index in Checker.frames of the computation, or -1 if none records effects
}

// markEffects returns the position of the effects recorded so far by the
// innermost computation in progress.
func (c *Checker) markEffects() memoEffectsMark {
	n := c.memoTop - 1
	m := memoEffectsMark{frameIndex: n}
	if n >= 0 {
		if f := c.frames[n].effects; f != nil {
			m.effects, m.deps = len(f.effects), len(f.deps)
		}
	}
	return m
}

func (c *Checker) discardEffectsSince(m memoEffectsMark) {
	if m.frameIndex < 0 {
		return
	}
	debug.Assert(c.memoTop-1 == m.frameIndex, "effects discarded outside their computation")
	f := c.frames[m.frameIndex].effects
	if f == nil {
		return
	}
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
// to change, so rollBackTo can restore them.
//
//go:noinline
func (c *Checker) logPendingUndo(slot *memoSlot) {
	old := slot.pending
	n := 0
	if old != nil {
		n = len(old.deps)
	}
	c.journal = append(c.journal, undoEntry{effects: true, restore: func() {
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
func (c *Checker) typeResolutionMemo(r *TypeResolution) *memoHeader {
	switch r.propertyName {
	case TypeSystemPropertyNameType:
		return &c.valueSymbolLinks.Get(r.target.(*ast.Symbol)).resolvedType.memoHeader
	case TypeSystemPropertyNameDeclaredType:
		return &c.typeAliasLinks.Get(r.target.(*ast.Symbol)).declaredType.memoHeader
	case TypeSystemPropertyNameResolvedTypeArguments:
		return &r.target.(*Type).AsTypeReference().resolvedTypeArguments.memoHeader
	case TypeSystemPropertyNameResolvedBaseTypes:
		return &r.target.(*Type).AsInterfaceType().resolvedBaseTypes.memoHeader
	case TypeSystemPropertyNameResolvedBaseConstructorType:
		return &r.target.(*Type).AsInterfaceType().resolvedBaseConstructorType.memoHeader
	case TypeSystemPropertyNameResolvedReturnType:
		return &r.target.(*Signature).resolvedReturnType.memoHeader
	case TypeSystemPropertyNameResolvedBaseConstraint:
		return &r.target.(*Type).AsConstrainedType().resolvedBaseConstraint.memoHeader
	case TypeSystemPropertyNameWriteType:
		if extra := c.valueSymbolLinks.Get(r.target.(*ast.Symbol)).extra; extra != nil {
			return &extra.writeType.memoHeader
		}
	case TypeSystemPropertyNameAliasTarget:
		return &c.aliasSymbolLinks.Get(r.target.(*ast.Symbol)).aliasTarget.memoHeader
	case TypeSystemPropertyNameComputedPropertyName:
		return &c.typeNodeLinks.Get(r.target.(*ast.Node)).resolvedType.memoHeader
	case TypeSystemPropertyNameVariances:
		return &c.varianceLinks.Get(r.target.(*ast.Symbol)).variances.memoHeader
	}
	return nil
}

// Instantiation limits
//
// Instantiation stops with an error when nested too deeply or after too
// many instantiations, as the instantiation is then very likely infinite.
// For the result not to depend on what was cached before, each memoized
// computation (including each cached comparison of two types) measures
// depth from where it started, whether or not it reports errors, and a
// reused result adds no depth.
//
// Chains that start a new memoized computation at every level are only
// stopped by a much larger limit on the depth of the whole outermost
// instantiation, which is only reached by runaway instantiations. Reaching
// it aborts the outermost instantiation, which then fails with an error. So
// that reusing a result aborts wherever computing it would, each result
// records its height, how much deeper than where its computation started
// that computation nested instantiations, including the heights of the
// results it reused. Reusing a result whose height would take the stack past
// the limit aborts, as computing it would have.
//
// Aborting finishes the computations in progress within the outermost
// instantiation, with every instantiation failing at once, and then rolls
// back what they cached and reported (see rollbackMark), as they were cut
// short at a point that depends on what was cached before. So while an
// instantiation is in progress, the journal logs every result cached, and
// errors reported by a reporting activation are held until the outermost
// instantiation finishes.
//
// The limit on the number of instantiations likewise applies to the work
// needed to compute the current result from scratch, estimated by a work
// sketch of the distinct instantiations it needs, including those of the
// results it reuses (see worksketch.go).

const (
	instantiationDepthLimit      = 100
	instantiationTotalDepthLimit = 1000
	instantiationCountLimit      = 5_000_000
)

// instantiationLimitReached reports whether a further instantiation would
// exceed a limit.
func (c *Checker) instantiationLimitReached() bool {
	return c.instantiationDepthLimitReached() || c.currentWork().sketch.exceedsLimit()
}

func (c *Checker) instantiationDepthLimitReached() bool {
	return len(c.instantiationStack)-int(c.currentWork().depthBase) == instantiationDepthLimit
}

// instantiateOutermost is instantiateTypeWithAlias with no instantiation in
// progress.
func (c *Checker) instantiateOutermost(t *Type, m *TypeMapper, alias *TypeAlias) *Type {
	mark := c.markRollback()
	nodeReports := len(c.nodeReports)
	result := c.instantiateTypeNested(t, m, alias)
	if c.aborting {
		c.aborting = false
		c.rollBackTo(mark)
		clear(c.nodeReports[nodeReports:])
		c.nodeReports = c.nodeReports[:nodeReports]
		clear(c.heldDiagnostics)
		c.heldDiagnostics = c.heldDiagnostics[:0]
		c.releaseRollback(mark)
		c.errorAtCurrentNode(diagnostics.Type_instantiation_is_excessively_deep_and_possibly_infinite)
		return c.errorType
	}
	c.releaseRollback(mark)
	if len(c.heldDiagnostics) != 0 {
		held := c.heldDiagnostics
		c.heldDiagnostics = nil
		for _, d := range held {
			if d.suggestion {
				c.suggestionDiagnostics.Add(d.diagnostic)
			} else {
				c.diagnostics.Add(d.diagnostic)
			}
		}
		clear(held)
		c.heldDiagnostics = held[:0]
	}
	return result
}

// abortInstantiation starts aborting the outermost instantiation in
// progress, as one would nest past instantiationTotalDepthLimit.
func (c *Checker) abortInstantiation() {
	f := c.currentWork()
	f.top = max(f.top, instantiationTotalDepthLimit+1)
	c.aborting = true
}

// A heldDiagnostic is an error reported by a reporting activation within
// the outermost instantiation in progress, which is only reported once the
// instantiation finishes without aborting.
type heldDiagnostic struct {
	diagnostic *ast.Diagnostic
	suggestion bool
}
