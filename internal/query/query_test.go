package query

import (
	"fmt"
	spoke2 "github.com/oliverbestmann/byke/spoke"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
	"unsafe"
)

type Position struct {
	spoke2.ComparableComponent[Position]
	X int
}

type Velocity struct {
	spoke2.ComparableComponent[Velocity]
	X int
}

type Acceleration struct {
	spoke2.ComparableComponent[Acceleration]
	X int
}

type SomeConfig struct {
	spoke2.ComparableComponent[SomeConfig]
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
		Builder: spoke2.QueryBuilder{
			Fetch: []spoke2.FetchComponent{
				{
					ComponentType: spoke2.ComponentTypeOf[Position](),
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
		Builder: spoke2.QueryBuilder{
			Fetch: []spoke2.FetchComponent{
				{
					ComponentType: spoke2.ComponentTypeOf[Position](),
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

		Mutable: []*spoke2.ComponentType{
			spoke2.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Option[Position]](), ParsedQuery{
		Builder: spoke2.QueryBuilder{
			Fetch: []spoke2.FetchComponent{
				{
					ComponentType: spoke2.ComponentTypeOf[Position](),
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
		Builder: spoke2.QueryBuilder{
			Fetch: []spoke2.FetchComponent{
				{
					ComponentType: spoke2.ComponentTypeOf[Position](),
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

		Mutable: []*spoke2.ComponentType{
			spoke2.ComponentTypeOf[Position](),
		},
	})

	parseQueryTest(t, reflect.TypeFor[Has[Position]](), ParsedQuery{
		Builder: spoke2.QueryBuilder{
			Fetch: []spoke2.FetchComponent{
				{
					ComponentType: spoke2.ComponentTypeOf[Position](),
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
		Builder: spoke2.QueryBuilder{
			Filters: []spoke2.Filter{
				{
					Added: spoke2.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Changed[Position]](), ParsedQuery{
		Builder: spoke2.QueryBuilder{
			Filters: []spoke2.Filter{
				{
					Changed: spoke2.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[With[Position]](), ParsedQuery{
		Builder: spoke2.QueryBuilder{
			Filters: []spoke2.Filter{
				{
					With: spoke2.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Without[Position]](), ParsedQuery{
		Builder: spoke2.QueryBuilder{
			Filters: []spoke2.Filter{
				{
					Without: spoke2.ComponentTypeOf[Position](),
				},
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Or[With[Velocity], Without[Position]]](), ParsedQuery{
		Builder: spoke2.QueryBuilder{
			Filters: []spoke2.Filter{
				{
					Or: []spoke2.Filter{
						{
							With: spoke2.ComponentTypeOf[Velocity](),
						},
						{
							Without: spoke2.ComponentTypeOf[Position](),
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
			Builder: spoke2.QueryBuilder{
				Fetch: []spoke2.FetchComponent{
					{
						ComponentType: spoke2.ComponentTypeOf[Position](),
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
			Builder: spoke2.QueryBuilder{
				Fetch: []spoke2.FetchComponent{
					{
						ComponentType: spoke2.ComponentTypeOf[Position](),
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

			Mutable: []*spoke2.ComponentType{
				spoke2.ComponentTypeOf[Position](),
			},
		})
	}

	{
		type Item struct {
			// can be embedded
			spoke2.EntityId

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
			Builder: spoke2.QueryBuilder{
				Fetch: []spoke2.FetchComponent{
					{
						ComponentType: spoke2.ComponentTypeOf[Position](),
					},
					{
						ComponentType: spoke2.ComponentTypeOf[SomeConfig](),
					},
					{
						ComponentType: spoke2.ComponentTypeOf[Acceleration](),
						Optional:      true,
					},
				},

				Filters: []spoke2.Filter{
					{
						Without: spoke2.ComponentTypeOf[Velocity](),
					},
					{
						Changed: spoke2.ComponentTypeOf[Position](),
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

			Mutable: []*spoke2.ComponentType{
				spoke2.ComponentTypeOf[Position](),
			},
		})
	}

}

func TestFromEntity(t *testing.T) {
	s := spoke2.NewStorage()
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

		velocity := entity.Get(spoke2.ComponentTypeOf[Velocity]())
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
			EntityId spoke2.EntityId
			With[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithEntity{
			EntityId: spoke2.EntityId(10),
		})
	}

	{
		type QueryItemWithEmbeddedEntity struct {
			spoke2.EntityId
			With[Velocity]
		}

		runTestFromEntity(t, entity, QueryItemWithEmbeddedEntity{
			EntityId: spoke2.EntityId(10),
		})
	}

	type QueryItemWithNestedStruct struct {
		spoke2.EntityId

		Motion struct {
			Position *Position
			Velocity Velocity
		}
	}

	runTestFromEntity(t, entity, QueryItemWithNestedStruct{
		EntityId: spoke2.EntityId(10),
		Motion: struct {
			Position *Position
			Velocity Velocity
		}{
			Position: &Position{X: 1},
			Velocity: Velocity{X: 2},
		},
	})

	{
		runTestFromEntity(t, entity, spoke2.EntityId(10))
	}
}

func runTestFromEntity[Q any](t *testing.T, entity spoke2.EntityRef, expected Q) {
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
		spoke2.EntityId

		With[Acceleration]
		Changed[Velocity]

		Position        *Position
		Velocity        Velocity
		Acceleration    Option[Acceleration]
		HasAcceleration Has[Acceleration]
	}

	query, err := ParseQuery(reflect.TypeFor[QueryItem]())
	require.NoError(b, err)

	s := spoke2.NewStorage()
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
