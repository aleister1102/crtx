package main

import (
	"sort"
	"sync"
)

// Set is a helper for managing unique string sets.
type Set struct {
	items map[string]struct{}
	mu    sync.RWMutex
}

// NewSet creates a new Set.
func NewSet() *Set {
	return &Set{items: make(map[string]struct{})}
}

// Add adds an item to the set.
func (s *Set) Add(item string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item] = struct{}{}
}

// Copy creates a new Set with a copy of the items from the original set.
func (s *Set) Copy() *Set {
	s.mu.RLock()
	defer s.mu.RUnlock()
	newSet := NewSet()
	for item := range s.items {
		newSet.items[item] = struct{}{}
	}
	return newSet
}

// Length returns the number of items in the set.
func (s *Set) Length() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Sorted returns a sorted slice of items from the set.
func (s *Set) Sorted() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]string, 0, len(s.items))
	for item := range s.items {
		res = append(res, item)
	}
	sort.Strings(res)
	return res
}
