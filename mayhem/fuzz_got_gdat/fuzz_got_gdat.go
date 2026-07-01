// gdat.Hash was removed upstream; the DeriveKey + Ref/DEK (Un)marshal surface
// is what remains, so this harness fuzzes that instead.
package fuzz_got_gdat

import (
	fuzz "github.com/AdaLogics/go-fuzz-headers"

	"github.com/gotvc/got/src/gdat"
)

func mayhemit(data []byte) int {
	if len(data) > 2 {
		num := int(data[0])
		data = data[1:]
		fuzzConsumer := fuzz.NewConsumer(data)

		switch num {

		case 0:
			var secret [32]byte
			fuzzConsumer.GetBytes()
			out := make([]byte, 32)
			gdat.DeriveKey(out, &secret, data)
			return 0

		case 1:
			var ref gdat.Ref
			_ = ref.UnmarshalText(data)
			return 0

		case 2:
			var ref gdat.Ref
			_ = ref.UnmarshalBinary(data)
			return 0

		case 3:
			var dek gdat.DEK
			_ = dek.UnmarshalText(data)
			return 0
		}
	}
	return 0
}

func Fuzz(data []byte) int {
	_ = mayhemit(data)
	return 0
}
