package pinyindata

import (
	"sync"
	"testing"
)

func TestDefault(t *testing.T) {
	t.Parallel()
	data := Default()
	assertEntry(t, data.Character, '重', "zhong")
	assertEntry(t, data.Character, '吕', "lv")
	assertEntry(t, data.Character, '𠮷', "ji")
	assertEntry(t, data.Phrase, "重庆", "chong qing")
	assertEntry(t, data.Phrase, "乐亭", "lao ting")
	assertEntry(t, data.Phrase, "长子县", "chang zi xian")
	assertEntry(t, data.Surname, "单", "shan")
	assertEntry(t, data.Surname, "万俟", "mo qi")
	assertEntry(t, data.Surname, "尉迟", "yu chi")
	if got := data.Simplified('慶'); got != '庆' {
		t.Fatalf("Simplified(慶) = %c", got)
	}
	counts := []int{0, 0, 0, 0}
	for character, reading := range data.Characters() {
		assertEntry(t, data.Character, character, reading)
		counts[0]++
	}
	for word, reading := range data.Phrases() {
		assertEntry(t, data.Phrase, word, reading)
		counts[1]++
	}
	for word, reading := range data.Surnames() {
		assertEntry(t, data.Surname, word, reading)
		counts[2]++
	}
	for character, simplified := range data.Traditional() {
		if got := data.Simplified(character); got != simplified {
			t.Errorf("Simplified(%c) = %c, want %c", character, got, simplified)
		}
		counts[3]++
	}
	for i, minimum := range []int{44000, 47000, 480, 1700} {
		if counts[i] < minimum {
			t.Errorf("table %d has %d entries, want at least %d", i, counts[i], minimum)
		}
	}
}

func TestDefaultConcurrent(t *testing.T) {
	t.Parallel()
	var workers sync.WaitGroup
	for range 16 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 100 {
				data := Default()
				if data != Default() {
					t.Error("Default returned different snapshots")
				}
				assertEntry(t, data.Phrase, "重庆", "chong qing")
				for word, reading := range data.Surnames() {
					assertEntry(t, data.Surname, word, reading)
					break
				}
			}
		}()
	}
	workers.Wait()
}
