package checker

import (
	"math/bits"
	"slices"
)

// referenceTable interns the type references to one generic target by their type
// arguments. Each reference already holds its type arguments, so the table stores
// the references themselves plus a 32-bit fingerprint per slot, instead of a
// separately hashed 16-byte key per entry. The fingerprint rejects most
// mismatched slots without loading the reference.
type referenceTable struct {
	slots  []*Type
	hashes []uint32
	count  int
	shift  uint8 // 64 - log2(len(slots)); slots are indexed by the high bits of the hash
}

func hashTypeList(types []*Type) uint64 {
	h := uint64(len(types))
	for _, t := range types {
		h = (bits.RotateLeft64(h, 5) ^ uint64(t.id)) * 0x9e3779b97f4a7c15
	}
	return h
}

func (r *referenceTable) find(typeArguments []*Type, hash uint64) *Type {
	if len(r.slots) == 0 {
		return nil
	}
	mask := uint64(len(r.slots) - 1)
	fingerprint := uint32(hash)
	for i := hash >> r.shift; ; i = (i + 1) & mask {
		t := r.slots[i]
		if t == nil {
			return nil
		}
		if r.hashes[i] == fingerprint && slices.Equal(t.AsTypeReference().resolvedTypeArguments, typeArguments) {
			return t
		}
	}
}

func (r *referenceTable) add(t *Type) {
	r.insert(t, hashTypeList(t.AsTypeReference().resolvedTypeArguments))
}

// insert adds a reference known not to be in the table.
func (r *referenceTable) insert(t *Type, hash uint64) {
	if (r.count+1)*4 > len(r.slots)*3 {
		r.grow()
	}
	r.place(t, hash)
	r.count++
}

func (r *referenceTable) place(t *Type, hash uint64) {
	mask := uint64(len(r.slots) - 1)
	i := hash >> r.shift
	for r.slots[i] != nil {
		i = (i + 1) & mask
	}
	r.slots[i] = t
	r.hashes[i] = uint32(hash)
}

func (r *referenceTable) grow() {
	old := r.slots
	r.slots = make([]*Type, max(len(old)*2, 4))
	r.hashes = make([]uint32, len(r.slots))
	r.shift = uint8(64 - bits.TrailingZeros(uint(len(r.slots))))
	for _, t := range old {
		if t != nil {
			r.place(t, hashTypeList(t.AsTypeReference().resolvedTypeArguments))
		}
	}
}
