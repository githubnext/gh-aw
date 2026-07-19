package trimleftright

import "testing"

func TestLooksSuspiciousCutset(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		cutset string
		want   bool
	}{
		// Trivially safe: too short or non-alphanumeric.
		{name: "empty", cutset: "", want: false},
		{name: "single rune", cutset: "a", want: false},
		{name: "contains whitespace", cutset: " \t", want: false},
		{name: "contains punctuation", cutset: "a!", want: false},

		// Suspicious: repeated rune (legacy coverage).
		{name: "repeated alnum", cutset: "foo", want: true},
		{name: "repeated suffix alnum", cutset: "barr", want: true},

		// Suspicious: distinct-char prefixes/suffixes — the main gap fixed.
		{name: "http prefix", cutset: "http", want: true},
		{name: "abc prefix", cutset: "abc", want: true},
		{name: "txt suffix", cutset: "txt", want: true},
		{name: "bar prefix", cutset: "bar", want: true},
		{name: "v1 version prefix", cutset: "v1", want: true},
		{name: "mixed alpha digit", cutset: "0x", want: true},

		// Safe: intentional decimal-digit class.
		{name: "all digits", cutset: "0123456789", want: false},
		{name: "digit subset", cutset: "012", want: false},

		// Safe: intentional ASCII-vowel class.
		{name: "all vowels", cutset: "aeiou", want: false},
		{name: "vowel subset", cutset: "aei", want: false},
		{name: "vowels uppercase", cutset: "AEIOU", want: false},

		// Safe: complete hex-letter alphabet (all six of a–f present).
		{name: "hex letters lower", cutset: "abcdef", want: false},
		{name: "hex letters upper", cutset: "ABCDEF", want: false},
		{name: "full hex with digits lower", cutset: "0123456789abcdef", want: false},
		{name: "full hex with digits upper", cutset: "0123456789ABCDEF", want: false},
		{name: "hex mixed case", cutset: "abcDEF", want: false},

		// Suspicious: partial hex-letter subset (not the complete alphabet).
		// "abc" is tested above via "abc prefix"; here we confirm the
		// 4-letter subset also escapes the hex exception.
		{name: "partial hex abce", cutset: "abce", want: true},
		// Boundary: all six hex letters plus a non-hex character.
		{name: "almost hex abcdefg", cutset: "abcdefg", want: true},
		// Regression: repeated hex letters look like a bug, not a character class.
		{name: "repeated hex letters", cutset: "aabbccddeeff", want: true},
		{name: "hex with duplicate letter", cutset: "aabcdef", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksSuspiciousCutset(tt.cutset)
			if got != tt.want {
				t.Fatalf("looksSuspiciousCutset(%q) = %v, want %v", tt.cutset, got, tt.want)
			}
		})
	}
}
