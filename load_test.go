package pinyindata

import (
	"bytes"
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"
)

func fixture() fstest.MapFS {
	return fstest.MapFS{
		"characters.txt":  {Data: []byte("# characters\r\n中\tzhong\r\n重\tzhong\r\n庆\tqing\r\n吕\tlv\r\n")},
		"phrases.txt":     {Data: []byte("重庆\tchong qing\n")},
		"surnames.txt":    {Data: []byte("单\tshan\n尉迟\tyu chi\n")},
		"traditional.txt": {Data: []byte("慶\t庆\n")},
	}
}

func TestLoadSnapshot(t *testing.T) {
	t.Parallel()
	source := fixture()
	dictionary, err := Load(source)
	if err != nil {
		t.Fatal(err)
	}
	assertEntry(t, dictionary.Character, '吕', "lv")
	assertEntry(t, dictionary.Phrase, "重庆", "chong qing")
	assertEntry(t, dictionary.Surname, "尉迟", "yu chi")
	if got := dictionary.Simplified('慶'); got != '庆' {
		t.Errorf("Simplified(慶) = %c", got)
	}
	if got := dictionary.Simplified('乾'); got != '乾' {
		t.Errorf("Simplified(乾) = %c", got)
	}
	if _, ok := dictionary.Phrase("重慶"); ok {
		t.Error("phrase lookup unexpectedly normalized its key")
	}
	if _, ok := dictionary.Surname("單"); ok {
		t.Error("surname lookup unexpectedly normalized its key")
	}

	index := bytes.Index(source["phrases.txt"].Data, []byte("chong"))
	source["phrases.txt"].Data[index] = 'z'
	assertEntry(t, dictionary.Phrase, "重庆", "chong qing")
	newDictionary, err := Load(source)
	if err != nil {
		t.Fatal(err)
	}
	assertEntry(t, newDictionary.Phrase, "重庆", "zhong qing")
}

func TestInvalidData(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		file, contents string
	}{
		{"characters.txt", "中\tzhong\n中\tzhong\n"},
		{"characters.txt", "中国\tzhong guo\n"},
		{"characters.txt", "A\ta\n"},
		{"characters.txt", "中\tzhong1\n"},
		{"phrases.txt", "重庆 chong qing\n"},
		{"phrases.txt", "\tchong qing\n"},
		{"phrases.txt", "重庆\tchong\n"},
		{"phrases.txt", "重庆\tchong qing shi\n"},
		{"phrases.txt", "重庆\tchong  qing\n"},
		{"phrases.txt", "重庆\tchong\tqing\n"},
		{"phrases.txt", "重庆\tChong qing\n"},
		{"phrases.txt", "重庆\tchóng qìng\n"},
		{"phrases.txt", "重庆\tchong \xff\n"},
		{"phrases.txt", "# invalid comment \xff\n"},
		{"traditional.txt", "慶\tA\n"},
		{"traditional.txt", "慶\t重庆\n"},
		{"traditional.txt", "重庆\t庆\n"},
		{"surnames.txt", "尉迟\tyu\n"},
	} {
		t.Run(tt.file+"/"+tt.contents, func(t *testing.T) {
			source := fixture()
			source[tt.file].Data = []byte(tt.contents)
			dictionary, err := Load(source)
			if dictionary != nil || !errors.Is(err, ErrInvalidData) || !strings.Contains(err.Error(), tt.file+":") {
				t.Fatalf("Load = %v, %v; want ErrInvalidData with file context", dictionary, err)
			}
		})
	}
	if dictionary, err := Load(nil); dictionary != nil || !errors.Is(err, ErrInvalidData) {
		t.Fatalf("Load(nil) = %v, %v", dictionary, err)
	}
}

func TestMissingFile(t *testing.T) {
	t.Parallel()
	source := fixture()
	delete(source, "surnames.txt")
	if dictionary, err := Load(source); dictionary != nil || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing file = %v, %v; want fs.ErrNotExist", dictionary, err)
	}
}

func TestEmptyTables(t *testing.T) {
	t.Parallel()
	source := fixture()
	for _, file := range source {
		file.Data = []byte(" \n  # empty\n")
	}
	dictionary, err := Load(source)
	if err != nil {
		t.Fatal(err)
	}
	assertEmpty(t, dictionary)
	assertEmpty(t, &Dictionary{})
}

func assertEmpty(t *testing.T, dictionary *Dictionary) {
	t.Helper()
	if value, ok := dictionary.Character('中'); value != "" || ok {
		t.Errorf("unexpected character: %q, %v", value, ok)
	}
	if value, ok := dictionary.Phrase("重庆"); value != "" || ok {
		t.Errorf("unexpected phrase: %q, %v", value, ok)
	}
	if value, ok := dictionary.Surname("单"); value != "" || ok {
		t.Errorf("unexpected surname: %q, %v", value, ok)
	}
	if got := dictionary.Simplified('慶'); got != '慶' {
		t.Errorf("unexpected simplified character: %c", got)
	}
	for range dictionary.Characters() {
		t.Error("unexpected character entry")
	}
	for range dictionary.Phrases() {
		t.Error("unexpected phrase entry")
	}
	for range dictionary.Surnames() {
		t.Error("unexpected surname entry")
	}
	for range dictionary.Traditional() {
		t.Error("unexpected traditional entry")
	}
}

func assertEntry[K comparable](t *testing.T, lookup func(K) (string, bool), key K, want string) {
	t.Helper()
	if got, ok := lookup(key); !ok || got != want {
		t.Errorf("lookup(%v) = %q, %v; want %q, true", key, got, ok, want)
	}
}

func FuzzLoad(f *testing.F) {
	for _, text := range []string{"", "重庆\tchong qing\n", "# comment\n", "重庆\tchong\n", "\xff"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		if len(text) > 4096 {
			t.Skip()
		}
		source := fixture()
		source["phrases.txt"].Data = []byte(text)
		dictionary, err := Load(source)
		if err != nil {
			if !errors.Is(err, ErrInvalidData) {
				t.Fatalf("unexpected error: %v", err)
			}
			return
		}
		for word, reading := range dictionary.Phrases() {
			assertEntry(t, dictionary.Phrase, word, reading)
		}
	})
}
