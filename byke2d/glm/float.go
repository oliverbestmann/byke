package glm

import "golang.org/x/exp/constraints"

type float interface {
	~float32 | ~float64
}

type Numeric interface {
	constraints.Integer | constraints.Float
}
