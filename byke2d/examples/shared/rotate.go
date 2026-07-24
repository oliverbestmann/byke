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
	app.AddSystems(byke.Update, rotateXSystem)
	app.AddSystems(byke.Update, rotateYSystem)
	app.AddSystems(byke.Update, rotateZSystem)
}

func rotateXSystem(
	vt byke.VirtualTime,
	items byke.Query[struct {
		RotatableZ RotatingX
		Transform  *byke2d.Transform
	}],
) {
	for item := range items.Items() {
		item.Transform.Rotation = item.Transform.Rotation.Mul(
			glm.RotationXQuat(glm.Rad(vt.DeltaSecs * item.RotatableZ.Speed)),
		)
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
		item.Transform.Rotation = item.Transform.Rotation.Mul(
			glm.RotationYQuat(glm.Rad(vt.DeltaSecs * item.RotatableZ.Speed)),
		)
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
		item.Transform.Rotation = item.Transform.Rotation.Mul(
			glm.RotationZQuat(glm.Rad(vt.DeltaSecs * item.RotatableZ.Speed)),
		)
	}
}
