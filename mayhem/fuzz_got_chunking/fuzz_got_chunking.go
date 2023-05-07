package fuzz_got_chunking

import (
    fuzz "github.com/AdaLogics/go-fuzz-headers"

    "github.com/gotvc/got/pkg/chunking"
)

func mayhemit(data []byte) int {

    if len(data) > 2 {
        num := int(data[0])
        data = data[1:]
        fuzzConsumer := fuzz.NewConsumer(data)
        
        switch num {
            
            case 0:
                var testContent chunking.ContentDefined
                fuzzConsumer.GenerateStruct(&testContent)

                testContent.Write(data)
                return 0

            case 1:
                var testContent chunking.ContentDefined
                fuzzConsumer.GenerateStruct(&testContent)
                testByte, _ := fuzzConsumer.GetByte()

                testContent.WriteByte(testByte)
                return 0
        }
    }
    return 0
}

func Fuzz(data []byte) int {
    _ = mayhemit(data)
    return 0
}