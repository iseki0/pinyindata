# pinyindata

Embedded Mandarin pronunciation data and a validated Go loader. Requires Go 1.24 or later and uses only the standard library.

The dataset contains 44,371 character defaults, 47,472 phrases, 490 single/compound surnames, and 1,745 traditional-to-simplified character mappings. Each entry has one selected reading in lowercase ASCII, without tones, using spaces between syllables and `v` for `ü`.

```go
package main

import (
    "fmt"
    "github.com/iseki0/pinyindata"
)

func main() {
    data := pinyindata.Default()
    phrase, _ := data.Phrase("重庆")
    surname, _ := data.Surname("单")
    fmt.Println(phrase, surname) // chong qing shan
}
```

`Character`, `Phrase` and `Surname` perform exact table lookups. `Simplified` returns a mapped character or the original character. Apply it to phrase/surname lookup keys as needed; character defaults preserve the original character's identity. The iterator methods expose entries without exposing mutable maps. Dictionaries are immutable and safe for concurrent reads. Segmentation, contextual pronunciation selection and search-key generation remain the consumer's responsibility.

To load files, use `pinyindata.Load(os.DirFS("dictionary"))`. That directory must contain `characters.txt`, `phrases.txt`, `surnames.txt` and `traditional.txt` with the same tab-separated format as [data](data). All four tables are validated before a snapshot is returned. An empty table is allowed; duplicate keys, invalid records and invalid UTF-8 return `ErrInvalidData`. File errors preserve their `fs` error categories. Later source-file edits do not change a loaded snapshot.

Generated data is committed and embedded. A checkout can be built and used offline without running the generator. To regenerate, run these commands at the repository root:

```sh
go generate ./...
go test -race ./...
go vet ./...
```

The generator downloads exact upstream revisions and verifies SHA-256 before parsing. It normalizes readings, selects one reading for repeated entries, canonicalizes phrase/surname keys and applies [local corrections](data/overrides.txt). To cache source downloads, run `go run ./internal/generate -cache /path/to/cache`. Generated tables are sorted and contain no timestamps. Reviewing data changes includes reviewing the corrections and upstream licenses.

The Go code is licensed under [MIT](LICENSE). Data sources, pinned revisions, transformations and full upstream MIT/Unicode notices are in [THIRD_PARTY_LICENSES](THIRD_PARTY_LICENSES).
