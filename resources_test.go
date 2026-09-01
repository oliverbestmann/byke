package byke

import (
	"testing"
)

func BenchmarkSystemParamState_Resources(b *testing.B) {
	world := NewWorld()
	world.InsertResource(VirtualTime{})
	system := func(l VirtualTime) {}
	benchSystem(b, world, system)
}
