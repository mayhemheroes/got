package fuzz_got_chunking

import (
	fuzz "github.com/AdaLogics/go-fuzz-headers"

	"github.com/gotvc/got/src/chunking"
)

func mayhemit(data []byte) int {
	if len(data) > 2 {
		num := int(data[0])
		data = data[1:]
		fuzzConsumer := fuzz.NewConsumer(data)

		var key [32]byte
		onChunk := func([]byte) error { return nil }
		c := chunking.NewContentDefined(64, 1024, 1<<20, &key, onChunk)

		switch num {

		case 0:
			c.Write(data)
			return 0

		case 1:
			testByte, _ := fuzzConsumer.GetByte()
			c.WriteByte(testByte)
			return 0
		}
	}
	return 0
}

func Fuzz(data []byte) int {
	_ = mayhemit(data)
	return 0
}
