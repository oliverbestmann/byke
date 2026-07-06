package spoke

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

type (
	Component0  struct{ Component[Component0] }
	Component1  struct{ Component[Component1] }
	Component2  struct{ Component[Component2] }
	Component3  struct{ Component[Component3] }
	Component4  struct{ Component[Component4] }
	Component5  struct{ Component[Component5] }
	Component6  struct{ Component[Component6] }
	Component7  struct{ Component[Component7] }
	Component8  struct{ Component[Component8] }
	Component9  struct{ Component[Component9] }
	Component10 struct{ Component[Component10] }
	Component11 struct{ Component[Component11] }
	Component12 struct{ Component[Component12] }
	Component13 struct{ Component[Component13] }
	Component14 struct{ Component[Component14] }
	Component15 struct{ Component[Component15] }
	Component16 struct{ Component[Component16] }
	Component17 struct{ Component[Component17] }
	Component18 struct{ Component[Component18] }
	Component19 struct{ Component[Component19] }
	Component20 struct{ Component[Component20] }
	Component21 struct{ Component[Component21] }
	Component22 struct{ Component[Component22] }
	Component23 struct{ Component[Component23] }
	Component24 struct{ Component[Component24] }
	Component25 struct{ Component[Component25] }
	Component26 struct{ Component[Component26] }
	Component27 struct{ Component[Component27] }
	Component28 struct{ Component[Component28] }
	Component29 struct{ Component[Component29] }
	Component30 struct{ Component[Component30] }
	Component31 struct{ Component[Component31] }
	Component32 struct{ Component[Component32] }
	Component33 struct{ Component[Component33] }
	Component34 struct{ Component[Component34] }
	Component35 struct{ Component[Component35] }
	Component36 struct{ Component[Component36] }
	Component37 struct{ Component[Component37] }
	Component38 struct{ Component[Component38] }
	Component39 struct{ Component[Component39] }
	Component40 struct{ Component[Component40] }
	Component41 struct{ Component[Component41] }
	Component42 struct{ Component[Component42] }
	Component43 struct{ Component[Component43] }
	Component44 struct{ Component[Component44] }
	Component45 struct{ Component[Component45] }
	Component46 struct{ Component[Component46] }
	Component47 struct{ Component[Component47] }
	Component48 struct{ Component[Component48] }
	Component49 struct{ Component[Component49] }
)
