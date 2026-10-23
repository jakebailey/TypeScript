package checker

import (
	"math/bits"

	"github.com/microsoft/TypeScript/tsc/internal/ast"
)

// Work sketches
//
// The limit on the number of instantiations bounds the work needed to
// compute a result from scratch, which does not depend on what was cached
// before. Each memoized computation is a work frame with a sketch of the
// distinct instantiations its result needs: those it performs itself, and
// those of every result it reads while an instantiation of its own is in
// progress, whether that result is computed then or reused. Reused results
// contribute the sketch stored when they were computed. Counting distinct
// instantiations rather than summing the work of each result keeps results
// shared by many others from being counted many times over.
//
// The sketch is a HyperLogLog estimate of the number of distinct
// instantiations, identified by structural hashes (see typeHash), so that
// the estimate does not depend on the order in which types were created.

const (
	workSketchIndexBits = 6
	workSketchRegisters = 1 << workSketchIndexBits
	workSketchMaxRank   = 40
	// The estimate is 0.709 * registers^2 / sum(2^-register). The sum is
	// kept as an integer scaled by 2^workSketchMaxRank, so that it is exact
	// whatever the order of additions.
	workSketchLimit = 709 * workSketchRegisters * workSketchRegisters * (1 << workSketchMaxRank) / (1000 * instantiationCountLimit)
)

type workSketch struct {
	registers [workSketchRegisters]uint8
	inverse   uint64 // Sum of 2^(workSketchMaxRank-register)
}

func newWorkSketch() *workSketch {
	return &workSketch{inverse: workSketchRegisters << workSketchMaxRank}
}

func (s *workSketch) raise(i uint64, rank uint8) {
	if old := s.registers[i]; rank > old {
		s.inverse -= 1 << (workSketchMaxRank - old)
		s.inverse += 1 << (workSketchMaxRank - rank)
		s.registers[i] = rank
	}
}

func (s *workSketch) add(h uint64) {
	rank := uint8(bits.LeadingZeros64(h<<workSketchIndexBits|1<<(workSketchIndexBits-1)) + 1)
	s.raise(h>>(64-workSketchIndexBits), min(rank, workSketchMaxRank))
}

func (s *workSketch) merge(other *workSketch) {
	for i, rank := range other.registers {
		s.raise(uint64(i), rank)
	}
}

// exceedsLimit reports whether the estimate reaches instantiationCountLimit.
func (s *workSketch) exceedsLimit() bool {
	return s != nil && s.inverse <= workSketchLimit
}

// A frame is a computation in progress: a memoized computation, a cached
// comparison, or a checked node. The innermost is the last in Checker.frames,
// above a root frame for work outside any of them.
type frame struct {
	sketch *workSketch // Allocated on first use
	// The sketch is stored for the memo identified by key, or in the slot
	// source creates for index, if either is given. The height is stored in
	// cell instead, if key identifies a memo.
	key    memoKey
	cell   *memoHeader
	source workSlotSource
	index  int32
	base   int32 // Length of the instantiation stack on entry
	// Length of the instantiation stack from which the depth limit counts:
	// base for a computation, or that of the enclosing frame for a node.
	depthBase int32
	// Deepest instantiation stack the frame needed, counting what computing
	// the results it reused would need; see "Instantiation limits".
	top int32
	// Whether reading a result accounts for its work even when no
	// instantiation of the frame is in progress: in a computation, whose
	// work is cached with its result, and in a node within one.
	heights bool
	serial  uint32 // Identifies the frame among all entered
	links   int32  // Length of workLinks on entry
	slots   int32  // Length of workLinkSlots on entry

	// A non-reporting activation records the effects of a memoized
	// computation, for replay by reuse of its result; see memo.go.
	memo     bool
	isQuery  bool
	prevMemo int32        // Checker.memoTop on entry
	result   *memoSlot    // Slot of the result computed
	effects  *memoEffects // Allocated on first use
	entry    *ast.Node    // Current node when the computation started
	query    savedCurrentNode
}

// resultWork is the work computing a result from scratch needs: the
// instantiations it performs, and how much deeper than where it starts
// they nest.
type resultWork struct {
	sketch *workSketch
	height int32
}

// A workSlotSource creates the slot of a computation in progress on demand,
// since few computations do work worth storing.
type workSlotSource interface {
	workSlot(i int) *memoSlot
	popWork(i int)
}

func (c *Checker) initWork() {
	c.frames = []frame{{}}
}

func (c *Checker) currentWork() *frame {
	return &c.frames[len(c.frames)-1]
}

// enterWork starts a work frame with an empty sketch for a computation,
// which also starts the depth limit of instantiations (see "Instantiation
// limits" in memo.go).
func (c *Checker) enterWork(key memoKey, cell *memoHeader, source workSlotSource, index int) {
	if key != 0 && c.observingWork() {
		// Entering a computed result reads it.
		if cell != nil {
			c.observeCellWork(cell)
		} else {
			c.observeWork(c.storedWork(key))
		}
	}
	base := int32(len(c.instantiationStack))
	c.pushWork(key, cell, source, index, base, true)
}

// enterNodeWork starts a work frame for checking a node, which continues
// the depth limit of the enclosing frame.
func (c *Checker) enterNodeWork() {
	parent := c.currentWork()
	c.pushWork(0, nil, nil, 0, parent.depthBase, parent.heights || c.inWork())
}

func (c *Checker) pushWork(key memoKey, cell *memoHeader, source workSlotSource, index int, depthBase int32, heights bool) {
	c.workSerials++
	base := int32(len(c.instantiationStack))
	c.frames = append(c.frames, frame{
		key:       key,
		cell:      cell,
		source:    source,
		index:     int32(index),
		base:      base,
		depthBase: depthBase,
		top:       base,
		heights:   heights,
		serial:    c.workSerials,
		links:     int32(len(c.workLinks)),
		slots:     int32(len(c.workLinkSlots)),
	})
}

// exitWork finishes the current frame, returning its work.
func (c *Checker) exitWork() resultWork {
	n := len(c.frames) - 1
	f := &c.frames[n]
	w := resultWork{f.sketch, f.top - f.base}
	if c.aborting {
		// Nothing computed while aborting is kept.
		w = resultWork{}
		c.dropWork(f)
	} else if w.sketch != nil || w.height != 0 || f.source != nil || int(f.links) != len(c.workLinks) || int(f.slots) != len(c.workLinkSlots) {
		c.finishWork(f, w)
	}
	top := f.top
	c.frames[n] = frame{}
	c.frames = c.frames[:n]
	parent := c.currentWork()
	parent.top = max(parent.top, top)
	if w.sketch != nil {
		c.observeSketch(w.sketch)
	}
	return w
}

// finishWork stores the work of a frame for the results it computed.
func (c *Checker) finishWork(f *frame, w resultWork) {
	if w.sketch != nil || w.height != 0 {
		var slot *memoSlot
		if f.source != nil {
			slot = f.source.workSlot(int(f.index))
		} else if f.cell != nil {
			f.cell.memoHeight = max(f.cell.memoHeight, w.height)
			if w.sketch != nil {
				c.memoSlot(f.key).storeWork(resultWork{sketch: w.sketch})
			}
		} else if f.key != 0 {
			slot = c.memoSlot(f.key)
			c.markWork(f.key)
		}
		if slot != nil {
			slot.storeWork(w)
		}
		// Results set outside a computation of their own need the work of
		// the computation that set them.
		for _, cell := range c.workLinks[f.links:] {
			cell.memoHeight = w.height
			if w.sketch != nil {
				c.memoSlot(cell.key()).sketch = w.sketch
			}
		}
		for _, slot := range c.workLinkSlots[f.slots:] {
			slot.linkWork(w)
		}
	}
	c.dropWork(f)
}

// dropWork releases the state of a frame without storing its work.
func (c *Checker) dropWork(f *frame) {
	if f.source != nil {
		f.source.popWork(int(f.index))
	}
	clear(c.workLinks[f.links:])
	c.workLinks = c.workLinks[:f.links]
	clear(c.workLinkSlots[f.slots:])
	c.workLinkSlots = c.workLinkSlots[:f.slots]
}

// storeWork stores the work of computing the slot's result, which earlier
// computations of it in a cycle may also have needed.
func (s *memoSlot) storeWork(w resultWork) {
	if w.sketch != nil {
		if s.sketch != nil {
			w.sketch.merge(s.sketch)
		}
		s.sketch = w.sketch
	}
	s.height = max(s.height, w.height)
}

// linkWork stores the work of the computation that set the slot's result.
func (s *memoSlot) linkWork(w resultWork) {
	if w.sketch != nil {
		s.sketch = w.sketch
	}
	s.height = w.height
}

func (s *memoSlot) work() resultWork {
	return resultWork{s.sketch, s.height}
}

// forgetWork makes the current work frame store its sketch for no result,
// for a computation that did not cache its result.
func (c *Checker) forgetWork() {
	f := c.currentWork()
	if f.source != nil {
		f.source.popWork(int(f.index))
	}
	f.key, f.cell, f.source = 0, nil, nil
}

// inWork reports whether an instantiation of the current work frame is in
// progress, so that results it reads are part of its work.
func (c *Checker) inWork() bool {
	return len(c.instantiationStack) > int(c.currentWork().base)
}

// observingWork reports whether reading a result accounts for its work.
func (c *Checker) observingWork() bool {
	f := c.currentWork()
	return f.heights || len(c.instantiationStack) > int(f.base)
}

// observeWork accounts for the work of a result read by the current frame.
func (c *Checker) observeWork(w resultWork) {
	if w.height != 0 {
		c.observeHeight(w.height)
	}
	c.observeSketch(w.sketch)
}

// observeHeight accounts for reusing a result whose computation nests
// instantiations height deeper than where it starts.
func (c *Checker) observeHeight(height int32) {
	depth := int32(len(c.instantiationStack))
	f := c.currentWork()
	f.top = max(f.top, depth+height)
	if depth != 0 && depth+height > instantiationTotalDepthLimit {
		c.abortInstantiation()
	}
}

// observeSketch accounts for the instantiations of a result read by the
// current frame.
func (c *Checker) observeSketch(sketch *workSketch) {
	if sketch == nil || !c.inWork() {
		return
	}
	f := c.currentWork()
	if sketch == f.sketch {
		return
	}
	if f.sketch == nil {
		f.sketch = newWorkSketch()
	}
	f.sketch.merge(sketch)
}

func (c *Checker) addWork(h uint64) {
	f := c.currentWork()
	if f.sketch == nil {
		f.sketch = newWorkSketch()
	}
	f.sketch.add(h)
}

// linkWork records that the memo was set by the current work frame.
func (c *Checker) linkWork(cell *memoHeader) {
	if len(c.frames) > 1 && cell != c.currentWork().cell {
		c.workLinks = append(c.workLinks, cell)
	}
}

// observeCellWork is observeWork for the result of a memo. Only results with
// a height can have a sketch, as they performed instantiations.
func (c *Checker) observeCellWork(cell *memoHeader) {
	if cell.memoHeight == 0 {
		return
	}
	c.observeHeight(cell.memoHeight)
	if c.inWork() {
		if slot := c.memoSlots[cell.key()]; slot != nil {
			c.observeSketch(slot.sketch)
		}
	}
}

func (c *Checker) storedWork(key memoKey) resultWork {
	if !c.mayHaveWork(key) {
		return resultWork{}
	}
	slot := c.memoSlots[key]
	if slot == nil {
		return resultWork{}
	}
	return slot.work()
}

// The work filter is a set of the memo keys whose slots may store work, so
// that reading a result without effects to account for can skip looking up
// its slot. Most reads within computations only account for work, and most
// results need none.

const workFilterBits = 22

func workFilterIndex(key memoKey) uint64 {
	return uint64(key) * 0x9E3779B97F4A7C15 >> (64 - workFilterBits)
}

func (c *Checker) markWork(key memoKey) {
	if c.workFilter == nil {
		c.workFilter = make([]uint64, 1<<workFilterBits/64)
	}
	i := workFilterIndex(key)
	c.workFilter[i/64] |= 1 << (i % 64)
}

func (c *Checker) mayHaveWork(key memoKey) bool {
	if c.workFilter == nil {
		return false
	}
	i := workFilterIndex(key)
	return c.workFilter[i/64]&(1<<(i%64)) != 0
}
