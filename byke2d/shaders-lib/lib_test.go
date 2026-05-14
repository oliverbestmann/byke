package shaders_lib

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	require.NotEmpty(t, All())
}
