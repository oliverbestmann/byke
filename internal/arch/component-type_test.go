package arch

import (
	"fmt"
	"testing"
)

var Blackbox int

func BenchmarkComponentLookupSlice(b *testing.B) {
	for idx := range 20 {
		componentCount := idx + 1
		b.Run(fmt.Sprintf("Count=%d", componentCount), func(b *testing.B) {
			types := components(componentCount)
			query := types[len(types)-1]

			for b.Loop() {
				for idx, ty := range types {
					if ty == query {
						Blackbox += idx
						break
					}
				}
			}
		})
	}
}

func BenchmarkComponentLookupMap(b *testing.B) {
	for idx := range 20 {
		componentCount := idx + 1
		b.Run(fmt.Sprintf("Count=%d", componentCount), func(b *testing.B) {
			types := map[*ComponentType]int{}

			typesSlice := components(componentCount)
			for idx, ty := range typesSlice {
				types[ty] = idx
			}

			query := typesSlice[len(typesSlice)-1]

			for b.Loop() {
				idx := types[query]
				Blackbox += idx
			}
		})
	}
}

func components(n int) []*ComponentType {
	types := []*ComponentType{
		Component0{}.ComponentType(),
		Component1{}.ComponentType(),
		Component2{}.ComponentType(),
		Component3{}.ComponentType(),
		Component4{}.ComponentType(),
		Component5{}.ComponentType(),
		Component6{}.ComponentType(),
		Component7{}.ComponentType(),
		Component8{}.ComponentType(),
		Component9{}.ComponentType(),
		Component10{}.ComponentType(),
		Component11{}.ComponentType(),
		Component12{}.ComponentType(),
		Component13{}.ComponentType(),
		Component14{}.ComponentType(),
		Component15{}.ComponentType(),
		Component16{}.ComponentType(),
		Component17{}.ComponentType(),
		Component18{}.ComponentType(),
		Component19{}.ComponentType(),
		Component20{}.ComponentType(),
		Component21{}.ComponentType(),
		Component22{}.ComponentType(),
		Component23{}.ComponentType(),
		Component24{}.ComponentType(),
		Component25{}.ComponentType(),
		Component26{}.ComponentType(),
		Component27{}.ComponentType(),
		Component28{}.ComponentType(),
		Component29{}.ComponentType(),
		Component30{}.ComponentType(),
		Component31{}.ComponentType(),
		Component32{}.ComponentType(),
		Component33{}.ComponentType(),
		Component34{}.ComponentType(),
		Component35{}.ComponentType(),
		Component36{}.ComponentType(),
		Component37{}.ComponentType(),
		Component38{}.ComponentType(),
		Component39{}.ComponentType(),
		Component40{}.ComponentType(),
		Component41{}.ComponentType(),
		Component42{}.ComponentType(),
		Component43{}.ComponentType(),
		Component44{}.ComponentType(),
		Component45{}.ComponentType(),
		Component46{}.ComponentType(),
		Component47{}.ComponentType(),
		Component48{}.ComponentType(),
		Component49{}.ComponentType(),
	}

	fmt.Println(types)

	return types[:n]
}

type Component0 struct{ Component[Component0] }
type Component1 struct{ Component[Component1] }
type Component2 struct{ Component[Component2] }
type Component3 struct{ Component[Component3] }
type Component4 struct{ Component[Component4] }
type Component5 struct{ Component[Component5] }
type Component6 struct{ Component[Component6] }
type Component7 struct{ Component[Component7] }
type Component8 struct{ Component[Component8] }
type Component9 struct{ Component[Component9] }
type Component10 struct{ Component[Component10] }
type Component11 struct{ Component[Component11] }
type Component12 struct{ Component[Component12] }
type Component13 struct{ Component[Component13] }
type Component14 struct{ Component[Component14] }
type Component15 struct{ Component[Component15] }
type Component16 struct{ Component[Component16] }
type Component17 struct{ Component[Component17] }
type Component18 struct{ Component[Component18] }
type Component19 struct{ Component[Component19] }
type Component20 struct{ Component[Component20] }
type Component21 struct{ Component[Component21] }
type Component22 struct{ Component[Component22] }
type Component23 struct{ Component[Component23] }
type Component24 struct{ Component[Component24] }
type Component25 struct{ Component[Component25] }
type Component26 struct{ Component[Component26] }
type Component27 struct{ Component[Component27] }
type Component28 struct{ Component[Component28] }
type Component29 struct{ Component[Component29] }
type Component30 struct{ Component[Component30] }
type Component31 struct{ Component[Component31] }
type Component32 struct{ Component[Component32] }
type Component33 struct{ Component[Component33] }
type Component34 struct{ Component[Component34] }
type Component35 struct{ Component[Component35] }
type Component36 struct{ Component[Component36] }
type Component37 struct{ Component[Component37] }
type Component38 struct{ Component[Component38] }
type Component39 struct{ Component[Component39] }
type Component40 struct{ Component[Component40] }
type Component41 struct{ Component[Component41] }
type Component42 struct{ Component[Component42] }
type Component43 struct{ Component[Component43] }
type Component44 struct{ Component[Component44] }
type Component45 struct{ Component[Component45] }
type Component46 struct{ Component[Component46] }
type Component47 struct{ Component[Component47] }
type Component48 struct{ Component[Component48] }
type Component49 struct{ Component[Component49] }
