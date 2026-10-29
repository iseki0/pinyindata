// Command generate downloads pinned, checksum-verified dictionary sources and
// writes deterministic search dictionaries. It requires only the standard library.
package main

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/iseki0/pinyindata/internal/han"
)

const (
	characterRevision = "9d05c9e29cb6a7e5c79f2f5279f5c1014246a19a"
	phraseRevision    = "7aac0bf2af6d448815d69399e56eca1302fd2326"
	proRevision       = "0785234b4dc0fa3ea9f244e3d77703b2df6a9db4"
	characterURL      = "https://raw.githubusercontent.com/mozillazg/pinyin-data/" + characterRevision + "/"
	phraseURL         = "https://raw.githubusercontent.com/mozillazg/phrase-pinyin-data/" + phraseRevision + "/"
	proURL            = "https://raw.githubusercontent.com/zh-lx/pinyin-pro/" + proRevision + "/"
)

type source struct {
	name string
	url  string
	sum  string
}

var sources = []source{
	{"characters", characterURL + "pinyin.txt", "90c11bd680ccfe7f2617a6e25abdb1fb87b5540df3e6f011ca50aaf3c7263a89"},
	{"phrases", phraseURL + "pinyin.txt", "5eff64ffd4de072b7e7f96baec560c3366887f93e762cc35ef0a8f62a7259b73"},
	{"common", proURL + "packages/pinyin-pro/lib/data/dict1-pinyin.ts", "f269c8c1e37f1efea7a424925b7cba5c0b1c648ca50503696f60f9d333b02218"},
	{"dict2", proURL + "packages/pinyin-pro/lib/data/dict2.ts", "7b6d05893002fa3b9ef6381624f6ec51433dc1cb59c96f182a0c2fc92e333381"},
	{"dict3", proURL + "packages/pinyin-pro/lib/data/dict3.ts", "22c7db3345dde5a861817e84fff3926a072dc4b9b251488b02c0c3af0cc7766d"},
	{"dict4", proURL + "packages/pinyin-pro/lib/data/dict4.ts", "260d34cb7e7fd9311c614ad873d339b8e56814f8d8af760a7f1c6ebbbc1c35b5"},
	{"dict5", proURL + "packages/pinyin-pro/lib/data/dict5.ts", "0143daf04a4ad2252d33f4163d2f5f4d6046ab621d20d938c9255f0380d5e424"},
	{"surnames", proURL + "packages/pinyin-pro/lib/data/surname.ts", "daff420d81a35e8abcc5b5295d265142a3c4380e89ccef843669a397669b7733"},
	{"traditional", proURL + "packages/data/json/traditional.json", "48db7ca9309bb013e9bd8fdef30f0c29165f8a9ffdee530f812a9a13ef8ab560"},
}

func main() {
	cache := flag.String("cache", "", "optional directory for checksum-verified source downloads")
	flag.Parse()
	if err := generate(*cache); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(cache string) error {
	raw := make(map[string][]byte, len(sources))
	for _, src := range sources {
		data, err := fetch(src, cache)
		if err != nil {
			return err
		}
		raw[src.name] = data
	}
	characters, err := parseCharacters(raw["characters"])
	if err != nil {
		return err
	}
	common, err := parseScript(raw["common"], true)
	if err != nil {
		return err
	}
	for word, reading := range common {
		characters[word] = reading
	}
	phrases, err := parsePhrases(raw["phrases"])
	if err != nil {
		return err
	}
	for _, name := range []string{"dict2", "dict3", "dict4", "dict5"} {
		words, err := parseScript(raw[name], false)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		for word, reading := range words {
			phrases[word] = reading
		}
	}
	surnames, err := parseScript(raw["surnames"], false)
	if err != nil {
		return err
	}
	traditional := make(map[string]string)
	if err := json.Unmarshal(raw["traditional"], &traditional); err != nil {
		return err
	}
	for word, value := range traditional {
		if utf8.RuneCountInString(word) != 1 || utf8.RuneCountInString(value) != 1 || !han.Text(word) || !han.Text(value) {
			return fmt.Errorf("invalid traditional mapping %q: %q", word, value)
		}
	}

	// Only the phrase lookup is simplified. Character fallback always uses the
	// original character, preserving distinctions such as 乾(qian)/干(gan).
	canonicalize := func(words map[string]string) (map[string]string, error) {
		result := make(map[string]string, len(words))
		for _, word := range sortedKeys(words) {
			key := strings.Map(func(r rune) rune {
				if value, ok := traditional[string(r)]; ok {
					return []rune(value)[0]
				}
				return r
			}, word)
			if value, ok := words[key]; ok {
				// Prefer an explicit simplified entry over a traditional
				// variant, for example 应征 over upstream's 应徵.
				result[key] = value
				continue
			}
			if value, ok := result[key]; ok && value != words[word] {
				return nil, fmt.Errorf("conflicting normalized readings for %q", word)
			}
			result[key] = words[word]
		}
		return result, nil
	}
	phrases, err = canonicalize(phrases)
	if err != nil {
		return err
	}
	surnames, err = canonicalize(surnames)
	if err != nil {
		return err
	}
	corrections, err := os.ReadFile("data/overrides.txt")
	if err != nil {
		return err
	}
	local, err := parsePhrases(corrections)
	if err != nil {
		return err
	}
	for word, reading := range local {
		if utf8.RuneCountInString(word) == 1 {
			characters[word] = reading
		} else {
			phrases[word] = reading
		}
	}
	for _, output := range []struct {
		name  string
		words map[string]string
	}{
		{"characters", characters}, {"phrases", phrases},
		{"surnames", surnames}, {"traditional", traditional},
	} {
		if err := writeWords(output.name, output.words); err != nil {
			return err
		}
		fmt.Printf("%s: %d entries\n", output.name, len(output.words))
	}
	return nil
}

func fetch(src source, cache string) ([]byte, error) {
	var data []byte
	var err error
	if cache != "" {
		data, err = os.ReadFile(filepath.Join(cache, src.name))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
	}
	if data == nil {
		client := &http.Client{Timeout: 45 * time.Second}
		response, err := client.Get(src.url)
		if err != nil {
			return nil, err
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("download %s: HTTP %d", src.name, response.StatusCode)
		}
		data, err = io.ReadAll(io.LimitReader(response.Body, 16<<20))
		if err != nil {
			return nil, err
		}
	}
	if actual := fmt.Sprintf("%x", sha256.Sum256(data)); actual != src.sum {
		return nil, fmt.Errorf("checksum mismatch for %s: %s", src.name, actual)
	}
	if cache != "" {
		if err := os.MkdirAll(cache, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(cache, src.name), data, 0o644); err != nil {
			return nil, err
		}
	}
	return data, nil
}

func parseCharacters(data []byte) (map[string]string, error) {
	words := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := withoutComment(scanner.Text())
		if line == "" {
			continue
		}
		code, readings, ok := strings.Cut(line, ":")
		if !ok || !strings.HasPrefix(code, "U+") {
			return nil, fmt.Errorf("invalid character entry: %s", line)
		}
		point, err := strconv.ParseInt(strings.TrimPrefix(code, "U+"), 16, 32)
		if err != nil || !utf8.ValidRune(rune(point)) {
			return nil, fmt.Errorf("invalid code point: %s", code)
		}
		word := string(rune(point))
		if !han.Text(word) {
			continue
		}
		first, _, _ := strings.Cut(readings, ",")
		value, err := reading(word, first)
		if err != nil {
			return nil, err
		}
		words[word] = value
	}
	return words, scanner.Err()
}

func parsePhrases(data []byte) (map[string]string, error) {
	words := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := withoutComment(scanner.Text())
		if line == "" {
			continue
		}
		word, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid phrase entry: %s", line)
		}
		word = strings.TrimSpace(word)
		if !han.Text(word) {
			continue
		}
		value, err := reading(word, value)
		if err != nil {
			return nil, err
		}
		// Upstream sometimes repeats a word for multiple readings. Keep its
		// first reading rather than multiplying search keys.
		if _, exists := words[word]; !exists {
			words[word] = value
		}
	}
	return words, scanner.Err()
}

var scriptEntry = regexp.MustCompile(`(?m)^\s*([^:\r\n]+):\s*['"]([^'"]+)['"],?`)

func parseScript(data []byte, firstOnly bool) (map[string]string, error) {
	words := make(map[string]string)
	for _, match := range scriptEntry.FindAllSubmatch(data, -1) {
		word := strings.Trim(strings.TrimSpace(string(match[1])), "'\"")
		if !han.Text(word) {
			continue
		}
		raw := string(match[2])
		if firstOnly {
			if utf8.RuneCountInString(word) != 1 {
				return nil, fmt.Errorf("expected one character in %q", word)
			}
			raw, _, _ = strings.Cut(raw, " ")
		}
		value, err := reading(word, raw)
		if err != nil {
			return nil, err
		}
		words[word] = value
	}
	if len(words) == 0 {
		return nil, fmt.Errorf("no dictionary entries found in TypeScript source")
	}
	return words, nil
}

func withoutComment(line string) string {
	line, _, _ = strings.Cut(line, "#")
	return strings.TrimSpace(line)
}

var toneMarks = strings.NewReplacer(
	"ā", "a", "á", "a", "ǎ", "a", "à", "a",
	"ē", "e", "é", "e", "ě", "e", "è", "e", "ê", "e", "ế", "e", "ề", "e",
	"ī", "i", "í", "i", "ǐ", "i", "ì", "i",
	"ō", "o", "ó", "o", "ǒ", "o", "ò", "o",
	"ū", "u", "ú", "u", "ǔ", "u", "ù", "u",
	"ü", "v", "ǖ", "v", "ǘ", "v", "ǚ", "v", "ǜ", "v",
	"ń", "n", "ň", "n", "ǹ", "n", "ḿ", "m",
)

func reading(word, value string) (string, error) {
	value = toneMarks.Replace(value)
	value = strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Mn, r) {
			return -1
		}
		return r
	}, value)
	syllables := strings.Fields(value)
	if len(syllables) != utf8.RuneCountInString(word) {
		return "", fmt.Errorf("syllable count mismatch for %q: %q", word, value)
	}
	for _, syllable := range syllables {
		for _, r := range syllable {
			if r < 'a' || r > 'z' {
				return "", fmt.Errorf("invalid reading for %q: %q", word, value)
			}
		}
	}
	return strings.Join(syllables, " "), nil
}

func sortedKeys(words map[string]string) []string {
	keys := make([]string, 0, len(words))
	for key := range words {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func writeWords(name string, words map[string]string) error {
	var output strings.Builder
	output.WriteString("# Code generated by go generate; DO NOT EDIT. See ../THIRD_PARTY_LICENSES.\n")
	for _, key := range sortedKeys(words) {
		fmt.Fprintf(&output, "%s\t%s\n", key, words[key])
	}
	return os.WriteFile(filepath.Join("data", name+".txt"), []byte(output.String()), 0o644)
}
