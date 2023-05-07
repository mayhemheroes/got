package fuzz_got_branches

import (
    // "context"
    fuzz "github.com/AdaLogics/go-fuzz-headers"

    "github.com/gotvc/got/pkg/branches"
)

func mayhemit(data []byte) int {

    if len(data) > 2 {
        num := int(data[0])
        data = data[1:]
        fuzzConsumer := fuzz.NewConsumer(data)
        
        switch num {
            
            case 0:
                publicBool, _ := fuzzConsumer.GetBool()

                branches.NewMetadata(publicBool)
                return 0

            case 1:
                var testAnnotation branches.Annotation
                fuzzConsumer.GenerateStruct(&testAnnotation)

                testAnnotation.UnmarshalJSON(data)
                return 0

            case 2:
                var annotationArr []branches.Annotation
                entries, _ := fuzzConsumer.GetInt()

                for i := 0; i < entries; i++ {

                    var temp branches.Annotation
                    fuzzConsumer.GenerateStruct(&temp)

                    annotationArr = append(annotationArr, temp)
                }

                branches.SortAnnotations(annotationArr)
                return 0

            case 3:
                var annotationArr []branches.Annotation
                entries, _ := fuzzConsumer.GetInt()
                testKey, _ := fuzzConsumer.GetString()

                for i := 0; i < entries; i++ {

                    var temp branches.Annotation
                    fuzzConsumer.GenerateStruct(&temp)

                    annotationArr = append(annotationArr, temp)
                }

                branches.GetAnnotation(annotationArr, testKey)
                return 0

            case 4:
                var testBranch branches.Branch
                fuzzConsumer.GenerateStruct(&testBranch)

                branches.NewGotFS(&testBranch)
                return 0

            case 5:
                var testBranch branches.Branch
                fuzzConsumer.GenerateStruct(&testBranch)

                branches.NewGotVC(&testBranch)
                return 0

            case 6:
                var layerArr []branches.Layer
                entries, _ := fuzzConsumer.GetInt()

                for i := 0; i < entries; i++ {

                    var temp branches.Layer
                    fuzzConsumer.GenerateStruct(&temp)

                    layerArr = append(layerArr, temp)
                }

                branches.NewMultiSpace(layerArr)
                return 0

            case 7:
                testName, _ := fuzzConsumer.GetString()

                branches.CheckName(testName)
                return 0

            // default:
            //     delay = 0
            //     ctx := context.TODO()
            //     var testVolume branches.Volume
            //     fuzzConsumer.GenerateStruct(&testVolume)

            //     branches.CleanupVolume(ctx, testVolume)
            //     return 0
        }
    }
    return 0
}

func Fuzz(data []byte) int {
    _ = mayhemit(data)
    return 0
}