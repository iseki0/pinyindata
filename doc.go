// Package pinyindata stores and loads Mandarin pronunciation dictionaries.
//
// Default returns an immutable embedded dictionary of character defaults,
// phrase readings, surname readings and traditional-to-simplified mappings.
// Readings use lowercase ASCII, omit tones, separate syllables with spaces, and
// write ü as v. Each entry contains one selected reading.
//
// Load reads the same four UTF-8 tables from an fs.FS and validates every entry
// before returning a dictionary. Dictionaries support concurrent reads and
// iteration without exposing mutable maps. Iteration order is unspecified.
// Segmentation, pronunciation selection and search-key generation belong to
// the consuming application.
//
// The Go code is MIT-licensed. Data provenance and upstream license notices are
// in THIRD_PARTY_LICENSES. Regeneration requires network access to pinned,
// checksum-verified sources; normal builds and dictionary use work offline.
package pinyindata

//go:generate go run ./internal/generate
