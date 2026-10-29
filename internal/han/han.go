// Package han recognizes the Unicode 17 Han repertoire on supported Go versions.
package han

import "unicode"

// Contains includes the Han additions made after Unicode 15.0, the version
// bundled with Go 1.24. Source: Unicode 17.0 Scripts.txt (Han), covered by the
// Unicode notice in THIRD_PARTY_LICENSES. This keeps loading and generation
// consistent across Go versions without dropping recent ideographs.
func Contains(character rune) bool {
	return unicode.Is(unicode.Han, character) ||
		character >= 0x16FF2 && character <= 0x16FF6 ||
		character >= 0x2B73A && character <= 0x2B73F ||
		character >= 0x2CEA2 && character <= 0x2CEAD ||
		character >= 0x2EBF0 && character <= 0x2EE5D ||
		character >= 0x323B0 && character <= 0x33479
}

// Text reports whether text is nonempty and contains only Han characters.
func Text(text string) bool {
	if text == "" {
		return false
	}
	for _, character := range text {
		if !Contains(character) {
			return false
		}
	}
	return true
}
