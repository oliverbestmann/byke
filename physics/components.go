package physics

import (
	"github.com/jakecoffman/cp/v2"
	"github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
)

type Velocity struct {
	byke.ComparableComponent[Velocity]
	Linear  Vec
	Angular Rad
}

type Mass struct {
	byke.ComparableComponent[Mass]
	Value float64
}

type Moment struct {
	byke.ComparableComponent[Moment]
	Value float64
}

type ExternalForces struct {
	byke.ComparableComponent[ExternalForces]
	Linear Vec
	Torque float64
}

type Collider struct {
	byke.Component[Collider]
	Shape ToShape

	// the actual collider cp.Shape
	shape *cp.Shape
}

func (Collider) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		ColliderDensity{Value: 1},
		ColliderElasticity{Value: 0},
		ColliderFriction{Value: 0.5},
	}
}

type ColliderDensity struct {
	byke.ComparableComponent[ColliderDensity]
	Value float64
}

type ColliderElasticity struct {
	byke.ComparableComponent[ColliderElasticity]
	Value float64
}

type ColliderFriction struct {
	byke.ComparableComponent[ColliderFriction]
	Value float64
}

type Sensor struct {
	byke.ImmutableComponent[Sensor]
}

type Body struct {
	byke.Component[Body]
	dynamic, static, kinematic bool

	body *cp.Body
}

func (Body) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		Velocity{},
		ExternalForces{},
	}
}

var RigidBodyDynamic = Body{dynamic: true}
var RigidBodyStatic = Body{static: true}
var RigidBodyKinematic = Body{kinematic: true}
