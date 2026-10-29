package pinyindata

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"unicode/utf8"

	"github.com/iseki0/pinyindata/internal/han"
)

// ErrInvalidData identifies malformed dictionary records. Load errors include
// the table and line number; file access errors retain their underlying cause.
var ErrInvalidData = errors.New("pinyindata: invalid data")

// Load reads characters.txt, phrases.txt, surnames.txt and traditional.txt from
// source's root. All four files must exist; empty tables are allowed. Each
// non-comment record is a Han key, a tab, and a value. Readings have one lowercase
// a..z syllable per character separated by single spaces; ü is written as v.
// Character and traditional table keys contain exactly one character, and
// traditional values contain one Han character. Blank lines and lines beginning
// with # are ignored. Duplicate keys, invalid UTF-8 and malformed entries return
// errors wrapping ErrInvalidData. File errors retain fs error categories.
//
// The returned snapshot owns its data and is unaffected by later source edits.
func Load(source fs.FS) (*Dictionary, error) {
	if source == nil {
		return nil, fmt.Errorf("%w: nil filesystem", ErrInvalidData)
	}
	dictionary := &Dictionary{
		characters:  make(map[rune]string),
		phrases:     make(map[string]string),
		surnames:    make(map[string]string),
		traditional: make(map[rune]rune),
	}
	for _, name := range []string{"characters", "phrases", "surnames", "traditional"} {
		file := name + ".txt"
		contents, err := fs.ReadFile(source, file)
		if err != nil {
			return nil, fmt.Errorf("pinyindata: read %s: %w", file, err)
		}
		seen := make(map[string]bool)
		lineNumber := 0
		for line := range strings.SplitSeq(string(contents), "\n") {
			lineNumber++
			line = strings.TrimSuffix(line, "\r")
			if !utf8.ValidString(line) {
				return nil, invalid(file, lineNumber, "expected UTF-8 text")
			}
			if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			key, value, ok := strings.Cut(line, "\t")
			if !ok || !han.Text(key) || seen[key] {
				return nil, invalid(file, lineNumber, "expected a unique Han key and a tab-separated value")
			}
			seen[key] = true
			size := utf8.RuneCountInString(key)
			if (name == "characters" || name == "traditional") && size != 1 {
				return nil, invalid(file, lineNumber, "expected one character as key")
			}
			if name == "traditional" {
				if utf8.RuneCountInString(value) != 1 || !han.Text(value) {
					return nil, invalid(file, lineNumber, "expected one Han character as value")
				}
				dictionary.traditional[[]rune(key)[0]] = []rune(value)[0]
				continue
			}
			if !validReading(value, size) {
				return nil, invalid(file, lineNumber, "expected one lowercase ASCII syllable per character")
			}
			switch name {
			case "characters":
				dictionary.characters[[]rune(key)[0]] = value
			case "phrases":
				dictionary.phrases[key] = value
			case "surnames":
				dictionary.surnames[key] = value
			}
		}
	}
	return dictionary, nil
}

func invalid(file string, line int, reason string) error {
	return fmt.Errorf("%w: %s:%d: %s", ErrInvalidData, file, line, reason)
}

func validReading(reading string, count int) bool {
	syllables := strings.Split(reading, " ")
	if len(syllables) != count {
		return false
	}
	for _, syllable := range syllables {
		if syllable == "" {
			return false
		}
		for _, r := range syllable {
			if r < 'a' || r > 'z' {
				return false
			}
		}
	}
	return true
}
