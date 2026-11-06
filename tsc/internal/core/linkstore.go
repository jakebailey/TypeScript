package core

import (
	"slices"
)

// Links store

type LinkStore[K comparable, V any] struct {
	entries map[K]*V
	arena   Arena[V]
}

func (s *LinkStore[K, V]) Get(key K) *V {
	value := s.entries[key]
	if value != nil {
		return value
	}
	if s.entries == nil {
		s.entries = make(map[K]*V)
	}
	value = s.arena.New()
	s.entries[key] = value
	return value
}

func (s *LinkStore[K, V]) Has(key K) bool {
	_, ok := s.entries[key]
	return ok
}

func (s *LinkStore[K, V]) TryGet(key K) *V {
	return s.entries[key]
}

const (
	LinkPageShift = 8
	LinkPageSize  = 1 << LinkPageShift
	LinkPageMask  = LinkPageSize - 1
)

// PagedLinkStore stores values inline in fixed-size pages of 256 entries,
// keyed by dense uint64 keys. A page is allocated the first time a key in its
// range is used, and a bitmap per page records which entries exist. Pages
// never move, so a pointer returned by Get stays valid.

type PagedLinkStore[V any] struct {
	pages []*linkPage[V]
}

type linkPage[V any] struct {
	present [LinkPageSize / 64]uint64
	values  [LinkPageSize]V
}

func (s *PagedLinkStore[V]) Get(key uint64) *V {
	pageIndex := key >> LinkPageShift
	if int(pageIndex) >= len(s.pages) {
		// Grow the length of the list to pageIndex+1
		s.pages = slices.Grow(s.pages, int(pageIndex)-len(s.pages)+1)[:pageIndex+1]
	}
	page := s.pages[pageIndex]
	if page == nil {
		page = new(linkPage[V])
		s.pages[pageIndex] = page
	}
	i := key & LinkPageMask
	page.present[i>>6] |= 1 << (i & 63)
	return &page.values[i]
}

func (s *PagedLinkStore[V]) Has(key uint64) bool {
	return s.TryGet(key) != nil
}

func (s *PagedLinkStore[V]) TryGet(key uint64) *V {
	pageIndex := key >> LinkPageShift
	if int(pageIndex) < len(s.pages) {
		if page := s.pages[pageIndex]; page != nil {
			i := key & LinkPageMask
			if page.present[i>>6]&(1<<(i&63)) != 0 {
				return &page.values[i]
			}
		}
	}
	return nil
}
