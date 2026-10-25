package shared

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/glm"
)

var _ = byke.ValidateComponent[RotatingX]()
var _ = byke.ValidateComponent[RotatingY]()
var _ = byke.ValidateComponent[RotatingZ]()

type RotatingX struct {
	byke.ComparableComponent[RotatingX]
	Speed float32
}

type RotatingY struct {
	byke.ComparableComponent[RotatingY]
	Speed float32
}

type RotatingZ struct {
	byke.ComparableComponent[RotatingZ]
	Speed float32
}

func PluginRotatable(app *byke.App) {
	// rotations do not commute, keep the order fixed for entities with multiple rotators
	app.AddSystems(byke.Update, byke.System(rotateXSystem, rotateYSystem, rotateZSystem).Chain())
}

func rotateXSystem(
	vt byke.VirtualTime,
	items byke.Query[struct {
		RotatableZ RotatingX
		Transform  *byke2d.Transform
	}],
) {
	for item := range items.Items() {
		item.Transform.Rotation = glm.RotationXQuat(glm.Rad(vt.DeltaSecs * item.RotatableZ.Speed)).Mul(item.Transform.Rotation)
	}
}

func rotateYSystem(
	vt byke.VirtualTime,
	items byke.Query[struct {
		RotatableZ RotatingY
		Transform  *byke2d.Transform
	}],
) {
	for item := range items.Items() {
		item.Transform.Rotation = glm.RotationYQuat(glm.Rad(vt.DeltaSecs * item.RotatableZ.Speed)).Mul(item.Transform.Rotation)
	}
}

func rotateZSystem(
	vt byke.VirtualTime,
	items byke.Query[struct {
		RotatableZ RotatingZ
		Transform  *byke2d.Transform
	}],
) {
	for item := range items.Items() {
		item.Transform.Rotation = glm.RotationZQuat(glm.Rad(vt.DeltaSecs * item.RotatableZ.Speed)).Mul(item.Transform.Rotation)
	}
}
