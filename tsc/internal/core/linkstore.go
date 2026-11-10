package core

import (
	"sync"
	"sync/atomic"
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

// PagedLinkStore implements a sparse-array-like structure for storing elements keyed by dense uint64 keys.
// Elements are allocated in an arena, element references are stored in fixed-size pages of 256 entries, and
// an index of pages is maintained in a growable list.
//
// It is safe for concurrent use. Looking up an element that exists takes
// only atomic loads; creating an element, its page, or a longer index takes
// a lock, and publishes it with an atomic store.
type PagedLinkStore[V any] struct {
	pages atomic.Pointer[[]atomic.Pointer[linkPage[V]]]
	mu    sync.Mutex // Guards growing pages, creating pages and elements, and arena
	arena Arena[V]
}

type linkPage[V any] [LinkPageSize]atomic.Pointer[V]

func (s *PagedLinkStore[V]) Get(key uint64) *V {
	if link := s.TryGet(key); link != nil {
		return link
	}
	return s.create(key)
}

func (s *PagedLinkStore[V]) create(key uint64) *V {
	s.mu.Lock()
	defer s.mu.Unlock()
	pageIndex := int(key >> LinkPageShift)
	pages := s.pages.Load()
	if pages == nil || pageIndex >= len(*pages) {
		var old []atomic.Pointer[linkPage[V]]
		if pages != nil {
			old = *pages
		}
		grown := make([]atomic.Pointer[linkPage[V]], max(pageIndex+1, 2*len(old)))
		for i := range old {
			grown[i].Store(old[i].Load())
		}
		pages = &grown
		s.pages.Store(pages)
	}
	page := (*pages)[pageIndex].Load()
	if page == nil {
		page = new(linkPage[V])
		(*pages)[pageIndex].Store(page)
	}
	link := page[key&LinkPageMask].Load()
	if link == nil {
		link = s.arena.New()
		page[key&LinkPageMask].Store(link)
	}
	return link
}

func (s *PagedLinkStore[V]) Has(key uint64) bool {
	return s.TryGet(key) != nil
}

func (s *PagedLinkStore[V]) TryGet(key uint64) *V {
	if pages := s.pages.Load(); pages != nil {
		if pageIndex := key >> LinkPageShift; pageIndex < uint64(len(*pages)) {
			if page := (*pages)[pageIndex].Load(); page != nil {
				return page[key&LinkPageMask].Load()
			}
		}
	}
	return nil
}
