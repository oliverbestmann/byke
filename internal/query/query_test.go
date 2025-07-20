package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"unsafe"
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

type SomeConfig struct {
	arch.ComparableComponent[SomeConfig]
	MaxX, MaxSpeed int
}

func parseQueryTest(t *testing.T, queryType reflect.Type, expected ParsedQuery) {
	t.Helper()

	t.Run(fmt.Sprintf("parse %s", queryType), func(t *testing.T) {
		parsed, err := ParseQuery(queryType)
		require.NoError(t, err)

		require.EqualValues(t, expected, parsed)
	})
}

func TestBuildQuerySimple(t *testing.T) {
	parseQueryTest(t, reflect.TypeFor[Position](), ParsedQuery{
		Builder: arch.QueryBuilder{
			Fetch: []arch.FetchComponent{
				{
					ComponentType: arch.ComponentTypeOf[Position](),
				},
			},
		},
		Setters: []Setter{
			{
				UnsafeCopyComponentValue: true,
				UnsafeFieldOffset:        0,
				ComponentIdx:             0,
				ComponentTypeSize:        unsafe.Sizeof(Position{}),
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[*Position](), ParsedQuery{
		Builder: arch.QueryBuilder{
			Fetch: []arch.FetchComponent{
				{
					ComponentType: arch.ComponentTypeOf[Position](),
				},
			},
		},

		Setters: []Setter{
			{
				UnsafeCopyComponentAddr: true,
				UnsafeFieldOffset:       0,
				ComponentIdx:            0,
			},
		},

		Mutable: []*arch.ComponentType{
			arch.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Option[Position]](), ParsedQuery{
		Builder: arch.QueryBuilder{
			Fetch: []arch.FetchComponent{
				{
					ComponentType: arch.ComponentTypeOf[Position](),
					Optional:      true,
				},
			},
		},

		Setters: []Setter{
			{
				UnsafeCopyComponentAddr: true,
				UnsafeFieldOffset:       0,
				ComponentIdx:            0,
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[OptionMut[Position]](), ParsedQuery{
		Builder: arch.QueryBuilder{
			Fetch: []arch.FetchComponent{
				{
					ComponentType: arch.ComponentTypeOf[Position](),
					Optional:      true,
				},
			},
		},

		Setters: []Setter{
			{
				UnsafeCopyComponentAddr: true,
				UnsafeFieldOffset:       0,
				ComponentIdx:            0,
			},
		},

		Mutable: []*arch.ComponentType{
			arch.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Has[Position]](), ParsedQuery{
		Builder: arch.QueryBuilder{
			Fetch: []arch.FetchComponent{
				{
					ComponentType: arch.ComponentTypeOf[Position](),
					Optional:      true,
				},
			},
		},

		Setters: []Setter{
			{
				UnsafeCopyComponentAddr: true,
				UnsafeFieldOffset:       0,
				ComponentIdx:            0,
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
			Builder: arch.QueryBuilder{
				Fetch: []arch.FetchComponent{
					{
						ComponentType: arch.ComponentTypeOf[Position](),
					},
				},
			},

			Setters: []Setter{
				{
					UnsafeCopyComponentValue: true,
					UnsafeFieldOffset:        0,
					ComponentIdx:             0,
					ComponentTypeSize:        unsafe.Sizeof(Position{}),
				},
			},
		})
	}

	{
		type Item struct {
			Position *Position
		}

		parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
			Builder: arch.QueryBuilder{
				Fetch: []arch.FetchComponent{
					{
						ComponentType: arch.ComponentTypeOf[Position](),
					},
				},
			},

			Setters: []Setter{
				{
					UnsafeCopyComponentAddr: true,
					UnsafeFieldOffset:       0,
					ComponentIdx:            0,
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
				Config       SomeConfig
				Acceleration Has[Acceleration]
			}
		}

		parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
			Builder: arch.QueryBuilder{
				Fetch: []arch.FetchComponent{
					{
						ComponentType: arch.ComponentTypeOf[Position](),
					},
					{
						ComponentType: arch.ComponentTypeOf[SomeConfig](),
					},
					{
						ComponentType: arch.ComponentTypeOf[Acceleration](),
						Optional:      true,
					},
				},

				Filters: []arch.Filter{
					{
						Without: arch.ComponentTypeOf[Velocity](),
					},
					{
						Changed: arch.ComponentTypeOf[Position](),
					},
				},
			},

			Setters: []Setter{
				{
					UseEntityId:       true,
					UnsafeFieldOffset: 0,
				},
				{
					UnsafeCopyComponentAddr: true,
					UnsafeFieldOffset:       8,
					ComponentIdx:            0,
				},
				{
					UnsafeCopyComponentValue: true,
					UnsafeFieldOffset:        16,
					ComponentIdx:             1,
					ComponentTypeSize:        16, // SizeOf(SomeConfig{})
				},
				{
					UnsafeCopyComponentAddr: true,
					UnsafeFieldOffset:       32,
					ComponentIdx:            2,
				},
			},

			Mutable: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		})
	}

}

func TestFromEntity(t *testing.T) {
	s := arch.NewStorage()
	s.Spawn(0, 10)
	s.InsertComponent(0, 10, &Position{X: 1})
	s.InsertComponent(0, 10, &Velocity{X: 2})

	entity, _ := s.Get(10)

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

		velocity := entity.Get(arch.ComponentTypeOf[Velocity]())
		runTestFromEntity(t, entity, QueryItemWithHas{
			Position:    Position{X: 1},
			HasVelocity: Has[Velocity]{ptr: uintptr(reflect.ValueOf(velocity).UnsafePointer())},
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

	s := arch.NewStorage()
	s.Spawn(0, 10)
	s.InsertComponent(0, 10, &Position{X: 1})
	s.InsertComponent(0, 10, &Velocity{X: 2})
	s.InsertComponent(0, 10, &Acceleration{X: 3})

	entity, _ := s.Get(10)

	var value QueryItem

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		FromEntity(&value, query.Setters, entity)
	}
}
