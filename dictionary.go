package pinyindata

import (
	"iter"
	"maps"
)

// Dictionary stores an immutable snapshot of four pronunciation tables.
// The zero value is an empty dictionary. Use Default or Load to obtain populated
// dictionaries. Methods and iterators support concurrent use.
type Dictionary struct {
	characters  map[rune]string
	phrases     map[string]string
	surnames    map[string]string
	traditional map[rune]rune
}

// Character returns the selected reading of the original character. It does not
// simplify the character or apply phrase or surname rules.
func (d *Dictionary) Character(character rune) (string, bool) {
	reading, ok := d.characters[character]
	return reading, ok
}

// Phrase returns the exact table entry for a phrase. The embedded phrase keys
// use simplified characters; callers can apply Simplified before lookup.
func (d *Dictionary) Phrase(phrase string) (string, bool) {
	reading, ok := d.phrases[phrase]
	return reading, ok
}

// Surname returns the exact table entry for a single or compound surname.
// Embedded surname keys use simplified characters.
func (d *Dictionary) Surname(surname string) (string, bool) {
	reading, ok := d.surnames[surname]
	return reading, ok
}

// Simplified returns the table's simplified form, or the original character if
// no mapping exists. This mapping is intended for phrase and surname lookup;
// character defaults remain available under the original character.
func (d *Dictionary) Simplified(character rune) rune {
	if simplified, ok := d.traditional[character]; ok {
		return simplified
	}
	return character
}

// Characters iterates over the character defaults in unspecified order.
func (d *Dictionary) Characters() iter.Seq2[rune, string] {
	return maps.All(d.characters)
}

// Phrases iterates over the phrase table in unspecified order.
func (d *Dictionary) Phrases() iter.Seq2[string, string] {
	return maps.All(d.phrases)
}

// Surnames iterates over the surname table in unspecified order.
func (d *Dictionary) Surnames() iter.Seq2[string, string] {
	return maps.All(d.surnames)
}

// Traditional iterates over traditional-to-simplified mappings in unspecified
// order. Modifying iteration variables does not change the dictionary.
func (d *Dictionary) Traditional() iter.Seq2[rune, rune] {
	return maps.All(d.traditional)
}
