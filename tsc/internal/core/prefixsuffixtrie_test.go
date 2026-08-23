package core

import (
	"slices"
	"testing"
)

func TestPrefixSuffixTrie(t *testing.T) {
	t.Parallel()

	var trie PrefixSuffixTrie[[]string]
	add := func(prefix string, suffix string, value string) {
		trie.Set(prefix, suffix, func(values []string, _ bool) []string {
			return append(values, value)
		})
	}
	add("", "", "all")
	add("", "suffix", "suffix")
	add("prefix", "", "prefix")
	add("prefix", "suffix", "both")
	add("prefix", "suffix", "both-again")
	add("same", "same", "no-overlap")
	add("🔥", "✅", "unicode")
	add("*", "?", "literal-metacharacters")

	tests := []struct {
		input string
		want  []string
	}{
		{input: "", want: []string{"all"}},
		{input: "unrelated", want: []string{"all"}},
		{input: "a suffix", want: []string{"all", "suffix"}},
		{input: "prefix value", want: []string{"all", "prefix"}},
		{input: "prefixsuffix", want: []string{"all", "both", "both-again", "prefix", "suffix"}},
		{input: "same", want: []string{"all"}},
		{input: "samesame", want: []string{"all", "no-overlap"}},
		{input: "🔥middle✅", want: []string{"all", "unicode"}},
		{input: "*middle?", want: []string{"all", "literal-metacharacters"}},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			var got []string
			for values := range trie.IterateAllMatches(test.input) {
				got = append(got, values...)
			}
			slices.Sort(got)
			slices.Sort(test.want)
			if !slices.Equal(got, test.want) {
				t.Errorf("IterateAllMatches(%q) = %q, want %q", test.input, got, test.want)
			}
			if got := trie.HasAnyMatch(test.input); got != (len(test.want) != 0) {
				t.Errorf("HasAnyMatch(%q) = %v, want %v", test.input, got, len(test.want) != 0)
			}
		})
	}
}

func TestPrefixSuffixTrieZeroValue(t *testing.T) {
	t.Parallel()

	var trie PrefixSuffixTrie[int]
	trie.Set("a", "z", func(value int, exists bool) int {
		if exists {
			t.Fatal("first update unexpectedly found a value")
		}
		return value
	})
	trie.Set("a", "z", func(value int, exists bool) int {
		if !exists {
			t.Fatal("second update did not find the zero value")
		}
		return value + 1
	})

	got := slices.Collect(trie.IterateAllMatches("abz"))
	if !slices.Equal(got, []int{1}) {
		t.Fatalf("IterateAllMatches returned %v, want [1]", got)
	}
}
