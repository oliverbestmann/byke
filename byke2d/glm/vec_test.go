package glm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVecJSON(t *testing.T) {
	data, err := json.Marshal(V3f(1, 2.5, -3))
	require.NoError(t, err)
	require.JSONEq(t, `[1, 2.5, -3]`, string(data))

	var v Vec3f
	require.NoError(t, json.Unmarshal(data, &v))
	require.Equal(t, V3f(1, 2.5, -3), v)

	require.Error(t, json.Unmarshal([]byte(`{"X": 1}`), &v))
}
