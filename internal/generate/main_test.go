package main

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCharacters(t *testing.T) {
	words, err := parseCharacters([]byte("# header\nU+5415: lǚ,lǜ # 吕\nU+20BB7: jí\nU+0041: a\nU+E000: a\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 2 || words["吕"] != "lv" || words["𠮷"] != "ji" {
		t.Fatalf("unexpected character readings: %v", words)
	}
	for _, text := range []string{"U+110000: a", "U+XYZ: a", "U+4E2D zhong", "U+4E2D: zh0ng"} {
		if _, err := parseCharacters([]byte(text)); err == nil {
			t.Errorf("accepted invalid entry %q", text)
		}
	}
}

func TestParsePhrases(t *testing.T) {
	words, err := parsePhrases([]byte("# header\n朝阳: zhāo yáng\n朝阳: cháo yáng\n女儿: nǚ ér # comment\nA股: a gu\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 2 || words["朝阳"] != "zhao yang" || words["女儿"] != "nv er" {
		t.Fatalf("unexpected phrase readings: %v", words)
	}
	for _, text := range []string{"中国 zhong guo", "中国: zhong", "中国: zh0ng guo"} {
		if _, err := parsePhrases([]byte(text)); err == nil {
			t.Errorf("accepted invalid entry %q", text)
		}
	}
}

func TestParseScript(t *testing.T) {
	words, err := parseScript([]byte("const Surnames = {\n 万俟: 'mò qí',\n '尉迟': \"yù chí\", // comment\n};\n"), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 2 || words["万俟"] != "mo qi" || words["尉迟"] != "yu chi" {
		t.Fatalf("unexpected surname readings: %v", words)
	}
	if _, err := parseScript([]byte("const Surnames = new Map();"), false); err == nil {
		t.Fatal("accepted source without recognized entries")
	}
	words, err = parseScript([]byte("const Characters = {\n 佛: 'fó fú',\n 吕: 'lǚ'\n};\n"), true)
	if err != nil || words["佛"] != "fo" || words["吕"] != "lv" {
		t.Fatalf("unexpected default character readings: %v, %v", words, err)
	}
}

func TestReading(t *testing.T) {
	for _, tt := range []struct {
		word, value, want string
	}{
		{"略", "lüè", "lve"},
		{"呣", "m̀", "m"},
		{"嗯", "ǹg", "ng"},
		{"欸", "ê̌", "e"},
	} {
		if got, err := reading(tt.word, tt.value); err != nil || got != tt.want {
			t.Errorf("reading(%q, %q) = %q, %v; want %q", tt.word, tt.value, got, err, tt.want)
		}
	}
}

func TestFetchVerifiesCachedData(t *testing.T) {
	dir := t.TempDir()
	data := []byte("trusted source")
	src := source{name: "test", url: "https://invalid.example/", sum: fmt.Sprintf("%x", sha256.Sum256(data))}
	path := filepath.Join(dir, src.name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := fetch(src, dir); err != nil || string(got) != string(data) {
		t.Fatalf("fetch cached data: %q, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("corrupted source"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fetch(src, dir); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("expected checksum failure, got %v", err)
	}
}
