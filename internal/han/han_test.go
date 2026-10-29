package han

import "testing"

func TestRepertoire(t *testing.T) {
	for _, character := range []rune{'中', '〇', '𠮷', 0x16FF2, 0x16FF6, 0x2B73A, 0x2B73F, 0x2CEA2, 0x2CEAD, 0x2EBF0, 0x2EE5D, 0x323B0, 0x33479} {
		if !Contains(character) {
			t.Errorf("missing U+%X", character)
		}
	}
	for _, character := range []rune{'A', '1', '🙂', 0x16FF7, 0x2B81E, 0x2CEAE, 0x2EBE1, 0x2EBEF, 0x2EE5E, 0x3347A} {
		if Contains(character) {
			t.Errorf("accepted non-Han U+%X", character)
		}
	}
	if !Text("重庆𮯴") || Text("") || Text("重A庆") || Text("\xff") {
		t.Error("incorrect Han string classification")
	}
}
