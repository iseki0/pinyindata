package pinyindata_test

import (
	"fmt"

	"github.com/iseki0/pinyindata"
)

func ExampleDefault() {
	dictionary := pinyindata.Default()
	phrase, _ := dictionary.Phrase("重庆")
	surname, _ := dictionary.Surname("单")
	fmt.Println(phrase, surname)
	// Output: chong qing shan
}
