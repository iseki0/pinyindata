package pinyindata

import (
	"embed"
	"io/fs"
	"sync"
)

//go:embed data/characters.txt data/phrases.txt data/surnames.txt data/traditional.txt
var embedded embed.FS

var embeddedDictionary = sync.OnceValue(func() *Dictionary {
	source, err := fs.Sub(embedded, "data")
	if err != nil {
		panic(err)
	}
	dictionary, err := Load(source)
	if err != nil {
		panic(err)
	}
	return dictionary
})

// Default returns the shared embedded dictionary, loading it once on first use.
// It requires no runtime files or network access. It panics if the library was
// built with invalid embedded tables; external data errors are returned by Load.
func Default() *Dictionary {
	return embeddedDictionary()
}
