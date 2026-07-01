// Package fuzz_got_branches fuzzes the successor of the old (now-removed)
// "branches" package: gotns's Root/MarkState parsing, the exported surface
// that plays the equivalent role (parsing untrusted namespace/mark bytes).
package fuzz_got_branches

import (
	fuzz "github.com/AdaLogics/go-fuzz-headers"

	"github.com/gotvc/got/src/gotns"
)

func mayhemit(data []byte) int {
	if len(data) > 2 {
		num := int(data[0])
		data = data[1:]
		fuzzConsumer := fuzz.NewConsumer(data)
		_ = fuzzConsumer

		switch num {

		case 0:
			gotns.ParseRoot(data)
			return 0

		case 1:
			var ms gotns.MarkState
			ms.Unmarshal(data)
			return 0
		}
	}
	return 0
}

func Fuzz(data []byte) int {
	_ = mayhemit(data)
	return 0
}
