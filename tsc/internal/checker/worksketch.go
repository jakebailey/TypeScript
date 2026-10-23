package checker

import "math/bits"

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

// A workFrame is a memoized computation, comparison, or checked node in
// progress. The innermost is the last in Checker.workFrames, above a root
// frame for work outside any of them.
type workFrame struct {
	sketch *workSketch // Allocated on first use
	// The sketch is stored for the memo identified by key, in slot, or in the
	// slot source creates for index, if any is given.
	key    memoKey
	slot   *memoSlot
	source workSlotSource
	index  int32
	base   int32  // Length of the instantiation stack on entry
	serial uint32 // Identifies the frame among all entered
	links  int32  // Length of workLinks on entry
	slots  int32  // Length of workLinkSlots on entry
}

// A workSlotSource creates the slot of a computation in progress on demand,
// since few computations do work worth storing.
type workSlotSource interface {
	workSlot(i int) *memoSlot
	popWork(i int)
}

func (c *Checker) initWork() {
	c.workFrames = []workFrame{{}}
}

func (c *Checker) currentWork() *workFrame {
	return &c.workFrames[len(c.workFrames)-1]
}

// enterWork starts a work frame with an empty sketch.
func (c *Checker) enterWork(key memoKey, slot *memoSlot, source workSlotSource, index int) {
	if key == 0 && slot == nil {
		c.pushWork(0, nil, source, index)
		return
	}
	if c.inWork() {
		// Entering a computed result reads it.
		c.observeWork(c.storedWork(key, slot))
	}
	c.pushWork(key, slot, source, index)
}

func (c *Checker) pushWork(key memoKey, slot *memoSlot, source workSlotSource, index int) {
	c.workSerials++
	c.workFrames = append(c.workFrames, workFrame{
		key:    key,
		slot:   slot,
		source: source,
		index:  int32(index),
		base:   int32(len(c.instantiationStack)),
		serial: c.workSerials,
		links:  int32(len(c.workLinks)),
		slots:  int32(len(c.workLinkSlots)),
	})
}

// exitWork finishes the current work frame, returning its sketch, or nil if
// it did no instantiations.
func (c *Checker) exitWork() *workSketch {
	n := len(c.workFrames) - 1
	f := &c.workFrames[n]
	sketch := f.sketch
	if sketch != nil || f.source != nil || int(f.links) != len(c.workLinks) || int(f.slots) != len(c.workLinkSlots) {
		c.finishWork(f)
	}
	c.workFrames[n] = workFrame{}
	c.workFrames = c.workFrames[:n]
	if sketch != nil {
		c.observeWork(sketch)
	}
	return sketch
}

// finishWork stores the sketch of a work frame for the results it computed.
func (c *Checker) finishWork(f *workFrame) {
	if sketch := f.sketch; sketch != nil {
		slot := f.slot
		if f.source != nil {
			slot = f.source.workSlot(int(f.index))
		} else if slot == nil && f.key != 0 {
			slot = c.memoSlot(f.key)
		}
		if slot != nil {
			if slot.sketch != nil {
				sketch.merge(slot.sketch)
			}
			slot.sketch = sketch
		}
		// Results set outside a computation of their own need the work of
		// the computation that set them.
		for _, key := range c.workLinks[f.links:] {
			c.memoSlot(key).sketch = sketch
		}
		for _, slot := range c.workLinkSlots[f.slots:] {
			slot.sketch = sketch
		}
	}
	if f.source != nil {
		f.source.popWork(int(f.index))
	}
	c.workLinks = c.workLinks[:f.links]
	clear(c.workLinkSlots[f.slots:])
	c.workLinkSlots = c.workLinkSlots[:f.slots]
}

// forgetWork makes the current work frame store its sketch for no result,
// for a computation that did not cache its result.
func (c *Checker) forgetWork() {
	f := c.currentWork()
	if f.source != nil {
		f.source.popWork(int(f.index))
	}
	f.key, f.slot, f.source = 0, nil, nil
}

// inWork reports whether an instantiation of the current work frame is in
// progress, so that results it reads are part of its work.
func (c *Checker) inWork() bool {
	return len(c.instantiationStack) > int(c.currentWork().base)
}

// observeWork accounts for the work of a result read by the current work
// frame.
func (c *Checker) observeWork(sketch *workSketch) {
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

// linkWork records that the memo identified by key was set by the current
// work frame.
func (c *Checker) linkWork(key memoKey) {
	if len(c.workFrames) > 1 && key != c.currentWork().key {
		c.workLinks = append(c.workLinks, key)
	}
}

func (c *Checker) storedWork(key memoKey, slot *memoSlot) *workSketch {
	if slot == nil && key != 0 {
		slot = c.memoSlots[key]
	}
	if slot == nil {
		return nil
	}
	return slot.sketch
}
