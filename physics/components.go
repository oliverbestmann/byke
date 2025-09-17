package physics

import (
	"math"

	b2 "github.com/oliverbestmann/box2d-go"
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
	Mass              float64
	RotationalInertia float64
	Center            Vec
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
	shape b2.Shape
}

func (Collider) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		ColliderDensity{Value: 1},
		ColliderElasticity{Value: 0},
		ColliderFriction{Value: 0.5},
		ShapeFilter{
			Mask:       math.MaxUint,
			Categories: 1,
		},
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

type ShapeFilter struct {
	byke.ComparableComponent[ShapeFilter]

	// Two objects with the same non-zero group value do not collide.
	// This is generally used to group objects in a composite object together to disable self collisions.
	Group int32
	// A bitmask of user definable categories that this object belongs to.
	// The category/mask combinations of both objects in a collision must agree for a collision to occur.
	Categories uint64
	// A bitmask of user definable category types that this object object collides with.
	// The category/mask combinations of both objects in a collision must agree for a collision to occur.
	Mask uint64
}

type Sensor struct {
	byke.ImmutableComponent[Sensor]
}

type ContactEventsEnabled struct {
	byke.ImmutableComponent[ContactEventsEnabled]
}

type SensorEventsEnabled struct {
	byke.ImmutableComponent[SensorEventsEnabled]
}

type Body struct {
	byke.Component[Body]
	dynamic, static, kinematic bool

	body b2.Body
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
