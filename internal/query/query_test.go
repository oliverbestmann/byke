package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"unsafe"
)

type Position struct {
	spoke.ComparableComponent[Position]
	X int
}

type Velocity struct {
	spoke.ComparableComponent[Velocity]
	X int
}

type Acceleration struct {
	spoke.ComparableComponent[Acceleration]
	X int
}

type SomeConfig struct {
	spoke.ComparableComponent[SomeConfig]
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
		Builder: spoke.QueryBuilder{
			Fetch: []spoke.FetchComponent{
				{
					ComponentType: spoke.ComponentTypeOf[Position](),
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
		Builder: spoke.QueryBuilder{
			Fetch: []spoke.FetchComponent{
				{
					ComponentType: spoke.ComponentTypeOf[Position](),
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

		Mutable: []*spoke.ComponentType{
			spoke.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Option[Position]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Fetch: []spoke.FetchComponent{
				{
					ComponentType: spoke.ComponentTypeOf[Position](),
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
		Builder: spoke.QueryBuilder{
			Fetch: []spoke.FetchComponent{
				{
					ComponentType: spoke.ComponentTypeOf[Position](),
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

		Mutable: []*spoke.ComponentType{
			spoke.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Has[Position]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Fetch: []spoke.FetchComponent{
				{
					ComponentType: spoke.ComponentTypeOf[Position](),
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

	parseQueryTest(t, reflect.TypeFor[Added[Position]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Filters: []spoke.Filter{
				{
					Added: spoke.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Changed[Position]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Filters: []spoke.Filter{
				{
					Changed: spoke.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[With[Position]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Filters: []spoke.Filter{
				{
					With: spoke.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Without[Position]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Filters: []spoke.Filter{
				{
					Without: spoke.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Or[With[Velocity], Without[Position]]](), ParsedQuery{
		Builder: spoke.QueryBuilder{
			Filters: []spoke.Filter{
				{
					Or: []spoke.Filter{
						{
							With: spoke.ComponentTypeOf[Velocity](),
						},
						{
							Without: spoke.ComponentTypeOf[Position](),
						},
					},
				},
			},
		},
	})
}

func TestParseQueryStruct(t *testing.T) {
	{
		type Item struct {
			Position Position
		}

		parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
			Builder: spoke.QueryBuilder{
				Fetch: []spoke.FetchComponent{
					{
						ComponentType: spoke.ComponentTypeOf[Position](),
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
			Builder: spoke.QueryBuilder{
				Fetch: []spoke.FetchComponent{
					{
						ComponentType: spoke.ComponentTypeOf[Position](),
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

			Mutable: []*spoke.ComponentType{
				spoke.ComponentTypeOf[Position](),
			},
		})
	}

	{
		type Item struct {
			// can be embedded
			spoke.EntityId

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
			Builder: spoke.QueryBuilder{
				Fetch: []spoke.FetchComponent{
					{
						ComponentType: spoke.ComponentTypeOf[Position](),
					},
					{
						ComponentType: spoke.ComponentTypeOf[SomeConfig](),
					},
					{
						ComponentType: spoke.ComponentTypeOf[Acceleration](),
						Optional:      true,
					},
				},

				Filters: []spoke.Filter{
					{
						Without: spoke.ComponentTypeOf[Velocity](),
					},
					{
						Changed: spoke.ComponentTypeOf[Position](),
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

			Mutable: []*spoke.ComponentType{
				spoke.ComponentTypeOf[Position](),
			},
		})
	}

}

func TestFromEntity(t *testing.T) {
	s := spoke.NewStorage()
	s.Spawn(0, 10, []spoke.ErasedComponent{
		&Position{X: 1},
		&Velocity{X: 2},
	})

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

		velocity := entity.Get(spoke.ComponentTypeOf[Velocity]())
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
			EntityId spoke.EntityId
			With[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithEntity{
			EntityId: spoke.EntityId(10),
		})
	}

	{
		type QueryItemWithEmbeddedEntity struct {
			spoke.EntityId
			With[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithEmbeddedEntity{
			EntityId: spoke.EntityId(10),
		})
	}

	type QueryItemWithNestedStruct struct {
		spoke.EntityId

		Motion struct {
			Position *Position
			Velocity Velocity
		}
	}

	runTestFromEntity(t, entity, QueryItemWithNestedStruct{
		EntityId: spoke.EntityId(10),
		Motion: struct {
			Position *Position
			Velocity Velocity
		}{
			Position: &Position{X: 1},
			Velocity: Velocity{X: 2},
		},
	})

	{
		runTestFromEntity(t, entity, spoke.EntityId(10))
	}
}

func runTestFromEntity[Q any](t *testing.T, entity spoke.EntityRef, expected Q) {
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
		spoke.EntityId

		With[Acceleration]
		Changed[Velocity]

		Position        *Position
		Velocity        Velocity
		Acceleration    Option[Acceleration]
		HasAcceleration Has[Acceleration]
	}

	query, err := ParseQuery(reflect.TypeFor[QueryItem]())
	require.NoError(b, err)

	s := spoke.NewStorage()
	s.Spawn(0, 10, []spoke.ErasedComponent{
		&Position{X: 1},
		&Velocity{X: 2},
		&Acceleration{X: 3},
	})

	entity, _ := s.Get(10)

	var value QueryItem

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		FromEntity(&value, query.Setters, entity)
	}
}
