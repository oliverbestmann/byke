package byke

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSystemId(t *testing.T) {
	t.Run("different systems", func(t *testing.T) {
		a := asSystemConfig(a).Id
		b := asSystemConfig(b).Id
		c := asSystemConfig(c).Id

		require.NotEqual(t, a, b)
		require.NotEqual(t, a, c)
		require.NotEqual(t, b, c)
	})

	t.Run("same system", func(t *testing.T) {
		a0 := asSystemConfig(a).Id
		a1 := asSystemConfig(a).Id
		a2 := asSystemConfig(a).Id

		require.Equal(t, a0, a1)
		require.Equal(t, a0, a2)
	})

}

func TestSystemIdWithClosure(t *testing.T) {
	a := asSystemConfig(makeSystem(1)).Id
	b := asSystemConfig(makeSystem(2)).Id
	require.NotEqual(t, a, b)
}

func TestSystemIdWithGeneric(t *testing.T) {
	a0 := asSystemConfig(gen[int]).Id
	a1 := asSystemConfig(gen[int]).Id
	require.Equal(t, a0, a1)

	b := asSystemConfig(gen[float32]).Id
	require.NotEqual(t, a0, b)
}

func makeSystem(param int) func() int {
	return func() int {
		return param
	}
}
