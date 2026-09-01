package byke

import (
	"testing"
)

type SomeValues struct {
	MapValue map[string]string
	AString  string
	value    uint32
}

type ResA struct{ SomeValues }
type ResB struct{ SomeValues }
type ResC struct{ SomeValues }
type ResD struct{ SomeValues }
type ResE struct{ SomeValues }
type ResF struct{ SomeValues }

var F any

type MyQuery = Query[struct {
	Name     Name
	Velocity Velocity
	Position Position
}]

func mySystem(a *ResA, b *ResB, c *ResC, d *ResD, e *ResE, q MyQuery) {
}

func BenchmarkSystemRunSystemReflect(b *testing.B) {
	b.ReportAllocs()

	world := NewWorld()
	world.InsertResource(ResA{})
	world.InsertResource(ResB{})
	world.InsertResource(ResC{})
	world.InsertResource(ResD{})
	world.InsertResource(ResE{})
	world.InsertResource(ResF{})

	benchSystem(b, world, mySystem)
}
