package core

import (
	"math/rand/v2"
	"testing"

	"gotest.tools/v3/assert"
)

func TestIdMapMatchesMap(t *testing.T) {
	t.Parallel()
	r := rand.New(rand.NewPCG(1, 2))
	var m IdMap[uint64, int]
	ref := map[uint64]int{}
	for range 200_000 {
		// A small key range forces collisions, long probe runs and deletes
		// that shift entries back across the end of the table.
		key := uint64(r.IntN(3000)) + 1
		if r.IntN(3) == 0 {
			key <<= 32
		}
		switch r.IntN(4) {
		case 0:
			m.Delete(key)
			delete(ref, key)
		case 1:
			*m.Ptr(key) += 1
			ref[key]++
		default:
			v := r.Int()
			m.Set(key, v)
			ref[key] = v
		}
		got, ok := m.TryGet(key)
		want, wantOk := ref[key]
		assert.Equal(t, ok, wantOk)
		assert.Equal(t, got, want)
	}
	assert.Equal(t, m.Len(), len(ref))
	n := 0
	for k, v := range m.All() {
		assert.Equal(t, v, ref[k])
		n++
	}
	assert.Equal(t, n, len(ref))
	for k, v := range ref {
		assert.Equal(t, m.Get(k), v)
	}
	m.Clear()
	assert.Equal(t, m.Len(), 0)
	assert.Assert(t, !m.Has(1))
}
