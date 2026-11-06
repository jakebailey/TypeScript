package core

import (
	"slices"
	"sync/atomic"
)

const (
	pageShift = 10
	pageSize  = 1 << pageShift
)

// Pages is an append-only sequence of records in fixed-size pages. Records never move, so a pointer to one stays
// valid, and the page directory is replaced rather than grown in place, so a reader holding an index from an earlier
// append can read concurrently with later appends.
type Pages[T any] struct {
	pages atomic.Pointer[[]*[pageSize]T]
	count uint32
}

// Add appends a zero record and returns its index and address.
func (p *Pages[T]) Add() (uint32, *T) {
	i := p.count
	ps := p.pages.Load()
	if ps == nil || int(i>>pageShift) == len(*ps) {
		// grown is declared here so that only growing allocates the
		// header that the directory pointer refers to.
		var grown []*[pageSize]T
		if ps != nil {
			grown = slices.Clip(*ps)
		}
		grown = append(grown, new([pageSize]T))
		p.pages.Store(&grown)
		ps = &grown
	}
	p.count++
	return i, &(*ps)[i>>pageShift][i&(pageSize-1)]
}

// At returns the address of the record at index i.
func (p *Pages[T]) At(i uint32) *T {
	return &(*p.pages.Load())[i>>pageShift][i&(pageSize-1)]
}

// Len returns the number of records.
func (p *Pages[T]) Len() uint32 {
	return p.count
}
