package query

import (
	"fmt"
	"github.com/oliverbestmann/byke/internal/arch"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

type Position struct {
	arch.Component[Position]
	X, Y int
}

type Velocity struct {
	arch.Component[Velocity]
	X, Y int
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
		Query: arch.Query{
			FetchOptional: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[OptionMut[Position]](), ParsedQuery{
		Query: arch.Query{
			FetchOptional: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},

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

	parseQueryTest(t, reflect.TypeFor[Added[Position]](), ParsedQuery{
		Query: arch.Query{
			WithAdded: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Changed[Position]](), ParsedQuery{
		Query: arch.Query{
			WithChanged: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[With[Position]](), ParsedQuery{
		Query: arch.Query{
			With: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		},
	})

	parseQueryTest(t, reflect.TypeFor[Without[Position]](), ParsedQuery{
		Query: arch.Query{
			Without: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
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
			Without[Velocity]
			Changed[Position]

			Nested struct {
				Position *Position
			}
		}

		parseQueryTest(t, reflect.TypeFor[Item](), ParsedQuery{
			Query: arch.Query{
				Fetch: []*arch.ComponentType{
					arch.ComponentTypeOf[Position](),
				},

				WithChanged: []*arch.ComponentType{
					arch.ComponentTypeOf[Position](),
				},

				Without: []*arch.ComponentType{
					arch.ComponentTypeOf[Velocity](),
				},
			},

			Mutable: []*arch.ComponentType{
				arch.ComponentTypeOf[Position](),
			},
		})
	}

}
