package byke2d

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSlabAllocator_Alloc(t *testing.T) {
	m := newSlabAllocator(1024)

	a, ok := m.Alloc(512)
	require.True(t, ok)

	b, ok := m.Alloc(512)
	require.True(t, ok)

	require.NotEqual(t, a, b)

	_, ok = m.Alloc(64)
	require.False(t, ok)
}

func TestSlabAllocator_Free(t *testing.T) {
	m := newSlabAllocator(1024)

	a, _ := m.Alloc(512)
	b, _ := m.Alloc(256)
	c, _ := m.Alloc(256)

	m.Free(b)
	m.Free(c)
	m.Free(a)

	_, ok := m.Alloc(1024)
	require.True(t, ok)
}
