package partycle

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
)

func particleSystem(
	commands *byke.Commands,
	vt byke.VirtualTime,
	particleQuery byke.Query[struct {
		byke.EntityId
		Particle  *Particle
		Transform *bykebiten.Transform
		Color     byke.OptionMut[bykebiten.ColorTint]
	}],
) {
	for item := range particleQuery.Items() {
		p := item.Particle
		if p.Lifetime.Tick(vt.Delta).JustFinished() {
			commands.Entity(item.EntityId).Despawn()
			continue
		}

		p.LinearVelocity = p.LinearVelocity.Add(p.LinearAcceleration.Mul(vt.DeltaSecs))
		p.AngularVelocity += p.AngularVelocity * gm.Rad(vt.DeltaSecs)

		if p.LinearDampening != 0 {
			// TODO use time based exp decay
			p.LinearVelocity = p.LinearVelocity.Mul(1 - p.LinearDampening*vt.DeltaSecs)
		}

		if p.AngularDampening != 0 {
			// TODO use time based exp decay
			p.AngularDampening = p.AngularDampening * (1 - p.AngularDampening*vt.DeltaSecs)
		}

		tr := item.Transform
		tr.Rotation += p.AngularVelocity * gm.Rad(vt.DeltaSecs)
		tr.Translation = tr.Translation.Add(p.LinearVelocity.Mul(vt.DeltaSecs))
		tr.Scale = p.BaseScale.MulEach(p.ScaleCurve.ValueAt(p.Lifetime.Fraction()))

		if color, ok := item.Color.Get(); ok {
			color.Color = p.ColorCurve.ValueAt(p.Lifetime.Fraction())
		}
	}
}
