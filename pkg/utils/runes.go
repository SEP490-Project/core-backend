package utils

import "unicode/utf16"

// UTF16RuneCount returns the number of UTF-16 code units in the given string.
func UTF16RuneCount(s string) int {
	return len(utf16.Encode(
		[]rune(s),
	))
}
