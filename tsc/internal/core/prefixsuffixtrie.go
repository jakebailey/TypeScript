package core

import "iter"

type PrefixSuffixTrie[T any] struct {
	root prefixSuffixTrieNode[T]
}

type prefixSuffixTrieNode[T any] struct {
	children map[byte]*prefixSuffixTrieNode[T]
	suffix   *prefixSuffixTrieNode[T]
	value    T
	hasValue bool
}

func (t *PrefixSuffixTrie[T]) Set(prefix string, suffix string, update func(value T, exists bool) T) {
	prefixNode := &t.root
	for i := range len(prefix) {
		prefixNode = prefixNode.child(prefix[i])
	}

	suffixNode := prefixNode.suffix
	if suffixNode == nil {
		suffixNode = &prefixSuffixTrieNode[T]{}
		prefixNode.suffix = suffixNode
	}
	for i := len(suffix) - 1; i >= 0; i-- {
		suffixNode = suffixNode.child(suffix[i])
	}
	suffixNode.value = update(suffixNode.value, suffixNode.hasValue)
	suffixNode.hasValue = true
}

func (t *PrefixSuffixTrie[T]) IterateAllMatches(input string) iter.Seq[T] {
	return func(yield func(T) bool) {
		prefixNode := &t.root
		if prefixNode.suffix != nil && !iterateSuffixMatches(prefixNode.suffix, input, 0, yield) {
			return
		}
		for i := range len(input) {
			prefixNode = prefixNode.children[input[i]]
			if prefixNode == nil {
				return
			}
			if prefixNode.suffix != nil && !iterateSuffixMatches(prefixNode.suffix, input, i+1, yield) {
				return
			}
		}
	}
}

func (t *PrefixSuffixTrie[T]) HasAnyMatch(input string) bool {
	for range t.IterateAllMatches(input) {
		return true
	}
	return false
}

func (n *prefixSuffixTrieNode[T]) child(ch byte) *prefixSuffixTrieNode[T] {
	if n.children == nil {
		n.children = make(map[byte]*prefixSuffixTrieNode[T])
	}
	child := n.children[ch]
	if child == nil {
		child = &prefixSuffixTrieNode[T]{}
		n.children[ch] = child
	}
	return child
}

func iterateSuffixMatches[T any](suffixNode *prefixSuffixTrieNode[T], input string, start int, yield func(T) bool) bool {
	if suffixNode.hasValue && !yield(suffixNode.value) {
		return false
	}
	for i := len(input) - 1; i >= start; i-- {
		suffixNode = suffixNode.children[input[i]]
		if suffixNode == nil {
			return true
		}
		if suffixNode.hasValue && !yield(suffixNode.value) {
			return false
		}
	}
	return true
}
