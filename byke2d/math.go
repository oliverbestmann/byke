package byke2d

import (
	"math/bits"

	"golang.org/x/exp/constraints"
)

func nextPowerOfTwo[T constraints.Integer](x T) T {
	if x <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(x-1))
}
