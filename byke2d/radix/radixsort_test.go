package radix

import (
	"math/rand/v2"
	"slices"
	"sort"
	"testing"

	"github.com/oliverbestmann/byke/byke2d/radix/radix_c"
	"github.com/stretchr/testify/require"
)

const ValueCount = 100_000

func BenchmarkStdlib(b *testing.B) {
	bench(b, func(valuesToSort []Value) {
		sort.Slice(valuesToSort, func(i, j int) bool {
			return valuesToSort[i].Key < valuesToSort[j].Key
		})
	})
}

func BenchmarkRadixGo(b *testing.B) {
	// scratch buffer for sorting
	scratch := make([]Value, ValueCount)

	bench(b, func(valuesToSort []Value) {
		radix_c.radixsortGo(valuesToSort, scratch)
	})
}

func BenchmarkRadixC(b *testing.B) {
	var cache Cache

	bench(b, func(valuesToSort []Value) {
		Sort(&cache, valuesToSort)
	})
}

func bench(b *testing.B, doSort func(values []Value)) {
	var values []Value

	r := rand.New(rand.NewPCG(1, 2))

	for range ValueCount {
		values = append(values, Value{
			Key:   r.Float32() - 0.5,
			Index: r.Uint32(),
		})
	}

	for idx := range b.N {
		valuesToSort := slices.Clone(values)
		doSort(valuesToSort)

		if idx == 0 {
			sorted := sort.SliceIsSorted(valuesToSort, func(i, j int) bool { return valuesToSort[i].Key < valuesToSort[j].Key })
			require.True(b, sorted)
		}
	}
}
