package byke

import (
	"testing"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/stretchr/testify/require"
)

func TestInSystemParamStateValue(t *testing.T) {
	world := NewWorld()

	var systemWasCalledWith glm.Vec3f
	system := func(inValue In[glm.Vec3f]) { systemWasCalledWith = inValue.Value }
	world.RunSystemWithInValue(system, glm.V3f(1, 2, 3))
	require.Equal(t, glm.V3f(1, 2, 3), systemWasCalledWith)
}

func TestInSystemParamStatePointerValue(t *testing.T) {
	world := NewWorld()

	var systemWasCalledWith *glm.Vec3f
	system := func(inValue In[*glm.Vec3f]) { systemWasCalledWith = inValue.Value }
	world.RunSystemWithInValue(system, new(glm.V3f(1, 2, 3)))
	require.Equal(t, glm.V3f(1, 2, 3), *systemWasCalledWith)
}
