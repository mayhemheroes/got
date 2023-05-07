package fuzz_got_gdat

import (
    fuzz "github.com/AdaLogics/go-fuzz-headers"

    "github.com/gotvc/got/pkg/gdat"
)

func mayhemit(data []byte) int {

    if len(data) > 2 {
        num := int(data[0])
        data = data[1:]
        fuzzConsumer := fuzz.NewConsumer(data)
        
        switch num {
            
            case 0:
                gdat.Hash(data)
                return 0

            // case 1:
            //     testBytes, _ := fuzzConsumer.GetBytes()
            //     testExtraByte, _ := fuzzConsumer.GetBytes()
            //     var testSecret [32]byte
            //     fuzzConsumer.CreateSlice(&testSecret)

            //     gdat.DeriveKey(testBytes, &testSecret, testExtraByte)
            //     return 0

            case 2:
                var ref gdat.Ref
                fuzzConsumer.GenerateStruct(&ref)

                ref.UnmarshalText(data)
                return 0

            case 3:
                var ref gdat.Ref
                fuzzConsumer.GenerateStruct(&ref)
                
                ref.UnmarshalBinary(data)
                return 0
        }
    }
    return 0
}

func Fuzz(data []byte) int {
    _ = mayhemit(data)
    return 0
}