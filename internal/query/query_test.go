package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type Position struct {
	arch.ComparableComponent[Position]
	X int
}

type Velocity struct {
	arch.ComparableComponent[Velocity]
	X int
}

type Acceleration struct {
	arch.ComparableComponent[Acceleration]
	X int
}

func parseQueryTest(t *testing.T, queryType reflect.Type, expected ParsedQuery) {
	t.Helper()

	t.Run(fmt.Sprintf("parse %s", queryType), func(t *testing.T) {
		parsed, err := ParseQuery(queryType)
		require.NoError(t, err)

		// do not include Setters in the comparison
		parsed.Setters = nil

		// TODO need to make this testable somehow?
		parsed.Query.Filters = nil

		require.EqualValues(t, expected, parsed)
	})
}

func TestBuildQuerySimple(t *testing.T) {
	parseQueryTest(t, reflect.TypeFor[Position](), ParsedQuery{
		Query: arch.Query{
			Fetch: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[*Position](), ParsedQuery{
		Query: arch.Query{
			Fetch: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},

		Mutable: []*arch.ComponentType{
			arch.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Option[Position]](), ParsedQuery{
		Query: arch.Query{},
	})

	parseQueryTest(t, reflect.TypeFor[OptionMut[Position]](), ParsedQuery{
		Query: arch.Query{},

		Mutable: []*arch.ComponentType{
			arch.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Has[Position]](), ParsedQuery{
		Query: arch.Query{
			FetchHas: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},
	})

	// parseQueryTest(t, reflect.TypeFor[Added[Position]](), ParsedQuery{
	// 	Query: arch.Query{
	// 		WithAdded: []*arch.ComponentType{
	// 			arch.ComponentTypeOf[Position](),
	// 		},
	// 	},
	// })

	// parseQueryTest(t, reflect.TypeFor[Changed[Position]](), ParsedQuery{
	// 	Query: arch.Query{
	// 		WithChanged: []*arch.ComponentType{
	// 			arch.ComponentTypeOf[Position](),
	// 		},
	// 	},
	// })

	// parseQueryTest(t, reflect.TypeFor[With[Position]](), ParsedQuery{
	// 	Query: arch.Query{
	// 		With: []*arch.ComponentType{
	// 			arch.ComponentTypeOf[Position](),
	// 		},
	// 	},
	// })

	// parseQueryTest(t, reflect.TypeFor[Without[Position]](), ParsedQuery{
	// 	Query: arch.Query{
	// 		Without: []*arch.ComponentType{
	// 			arch.ComponentTypeOf[Position](),
	// 		},
	// 	},
	// })
}

func TestParseQueryStruct(t *testing.T) {
	{
		type Item struct {
			Position Position
		}

		parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
			Query: arch.Query{
				Fetch: []*arch.ComponentType{
					arch.ComponentTypeOf[Position](),
				},
			},
		})
	}

	{
		type Item struct {
			Position *Position
		}

		parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
			Query: arch.Query{
				Fetch: []*arch.ComponentType{
					arch.ComponentTypeOf[Position](),
				},
			},

			Mutable: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		})
	}

	{
		type Item struct {
			// can be embedded
			arch.EntityId

			// embeddable filters can also be embedded
			Without[Velocity]
			Changed[Position]

			// normal fetches can be recursive
			Nested struct {
				Position     *Position
				Acceleration Has[Acceleration]
			}
		}

		//parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
		//	Query: arch.Query{
		//		Fetch: []*arch.ComponentType{
		//			arch.ComponentTypeOf[Position](),
		//		},
		//
		//		WithChanged: []*arch.ComponentType{
		//			arch.ComponentTypeOf[Position](),
		//		},
		//
		//		Without: []*arch.ComponentType{
		//			arch.ComponentTypeOf[Velocity](),
		//		},
		//
		//		FetchHas: []*arch.ComponentType{
		//			arch.ComponentTypeOf[Acceleration](),
		//		},
		//	},
		//
		//	Mutable: []*arch.ComponentType{
		//		arch.ComponentTypeOf[Position](),
		//	},
		//})
	}

}

func TestFromEntity(t *testing.T) {
	var entity = arch.EntityRef{
		EntityId: 10,
		Components: []arch.ComponentValue{
			{
				Type:  arch.ComponentTypeOf[Position](),
				Value: &Position{X: 1},
			},
			{
				Type:  arch.ComponentTypeOf[Velocity](),
				Value: &Velocity{X: 2},
			},
		},
	}

	runTestFromEntity(t, entity, Position{X: 1})
	runTestFromEntity(t, entity, &Position{X: 1})

	{
		type QueryItemWithMutable struct {
			Position *Position
			Velocity Velocity
		}

		runTestFromEntity(t, entity, QueryItemWithMutable{
			Position: &Position{X: 1},
			Velocity: Velocity{X: 2},
		})
	}

	{
		type QueryItemWithHas struct {
			Position    Position
			HasVelocity Has[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithHas{
			Position:    Position{X: 1},
			HasVelocity: Has[Velocity]{Exists: true},
		})
	}

	{
		type QueryItemWithOption struct {
			Position    Option[Position]
			HasVelocity OptionMut[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithOption{
			Position:    Option[Position]{value: &Position{X: 1}},
			HasVelocity: OptionMut[Velocity]{value: &Velocity{X: 2}},
		})
	}

	{
		type QueryItemWithWith struct {
			With[Velocity]
			Position Position
		}

		runTestFromEntity(t, entity, QueryItemWithWith{
			Position: Position{X: 1},
		})
	}

	{
		type QueryItemWithEntity struct {
			EntityId arch.EntityId
			With[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithEntity{
			EntityId: arch.EntityId(10),
		})
	}

	{
		type QueryItemWithEmbeddedEntity struct {
			arch.EntityId
			With[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithEmbeddedEntity{
			EntityId: arch.EntityId(10),
		})
	}

	type QueryItemWithNestedStruct struct {
		arch.EntityId

		Motion struct {
			Position *Position
			Velocity Velocity
		}
	}

	runTestFromEntity(t, entity, QueryItemWithNestedStruct{
		EntityId: arch.EntityId(10),
		Motion: struct {
			Position *Position
			Velocity Velocity
		}{
			Position: &Position{X: 1},
			Velocity: Velocity{X: 2},
		},
	})

	{
		runTestFromEntity(t, entity, arch.EntityId(10))
	}
}

func runTestFromEntity[Q any](t *testing.T, entity arch.EntityRef, expected Q) {
	t.Helper()

	t.Run(reflect.TypeFor[Q]().String(), func(t *testing.T) {
		parsed, err := ParseQuery(reflect.TypeFor[Q]())
		require.NoError(t, err)

		var queryTarget Q
		FromEntity(&queryTarget, parsed.Setters, entity)
		require.Equal(t, expected, queryTarget)
	})
}

func BenchmarkFromEntity(b *testing.B) {
	type QueryItem struct {
		arch.EntityId

		With[Acceleration]
		Changed[Velocity]

		Position        *Position
		Velocity        Velocity
		Acceleration    Option[Acceleration]
		HasAcceleration Has[Acceleration]
	}

	query, err := ParseQuery(reflect.TypeFor[QueryItem]())
	require.NoError(b, err)

	entity := arch.EntityRef{
		EntityId: 10,
		Components: []arch.ComponentValue{
			{
				Type:  arch.ComponentTypeOf[Position](),
				Value: &Position{X: 1},
			},
			{
				Type:  arch.ComponentTypeOf[Velocity](),
				Value: &Velocity{X: 2},
			},
			{
				Type:  arch.ComponentTypeOf[Acceleration](),
				Value: &Acceleration{X: 3},
			},
		},
	}

	var value QueryItem

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		FromEntity(&value, query.Setters, entity)
	}
}
