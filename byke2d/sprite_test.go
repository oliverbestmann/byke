package byke2d

import (
	"math/rand/v2"
	"slices"
	"sort"
	"testing"

	"github.com/Kaidzen-62/radixsort"
	"github.com/stretchr/testify/require"
)

type Value struct {
	Key   float32
	Index uint32
}

func BenchmarkSortZ_Float32(b *testing.B) {
	var values []Value

	r := rand.New(rand.NewPCG(1, 2))

	for range 100_000 {
		values = append(values, Value{
			Key:   r.Float32(),
			Index: r.Uint32(),
		})
	}

	for idx := range b.N {
		valuesToSort := slices.Clone(values)

		sort.Slice(valuesToSort, func(i, j int) bool {
			return valuesToSort[i].Key < valuesToSort[j].Key
		})

		if idx == 0 {
			sorted := sort.SliceIsSorted(valuesToSort, func(i, j int) bool { return valuesToSort[i].Key < valuesToSort[j].Key })
			require.True(b, sorted)
		}
	}
}

func BenchmarkSortZ_Float32_Radix(b *testing.B) {
	var values []Value

	r := rand.New(rand.NewPCG(1, 2))

	for range 100_000 {
		values = append(values, Value{
			Key:   r.Float32(),
			Index: r.Uint32(),
		})
	}

	// scratch buffer for sorting
	buf := make([]Value, len(values))

	for idx := range b.N {
		valuesToSort := slices.Clone(values)

		_ = radixsort.Generic[Value, float32](valuesToSort, buf, func(a Value) float32 { return a.Key })

		if idx == 0 {
			sorted := sort.SliceIsSorted(valuesToSort, func(i, j int) bool { return valuesToSort[i].Key < valuesToSort[j].Key })
			require.True(b, sorted)
		}
	}
}
