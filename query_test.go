package byke

import "testing"

func BenchmarkSystemParamState_Query(b *testing.B) {
	system := func(l Query[struct{ Pos Position }]) {}
	benchSystem(b, NewWorld(), system)
}

func BenchmarkSystemParamState_Single(b *testing.B) {
	system := func(l Single[struct{ Pos Position }]) {}
	benchSystem(b, NewWorld(), system)
}

func benchSystem(b *testing.B, world *World, system any) {
	b.ReportAllocs()
	b.ResetTimer()

	cached := AsCachedSystem(system)
	for b.Loop() {
		world.RunSystem(cached)
	}
}

func benchSystemWithInValue(b *testing.B, world *World, system any, inValue any) {
	b.ReportAllocs()
	b.ResetTimer()

	cached := AsCachedSystem(system)
	for b.Loop() {
		world.RunSystemWithInValue(cached, inValue)
	}
}
