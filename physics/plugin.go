package physics

import (
	"github.com/jakecoffman/cp/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
)

type cpSpace struct {
	*cp.Space
}

func Plugin(app *byke.App) {
	space := cp.NewSpace()
	space.SetGravity(cp.Vector{})
	app.InsertResource(cpSpace{space})
	app.AddSystems(byke.FixedUpdate, byke.System(makeBodySystem, preStepSyncSystem, updateSpaceSystem, postStepSyncSystem).Chain())
	app.AddSystems(byke.PostRender, debugSystem)
}

func makeBodySystem(
	space cpSpace,
	bodiesQuery byke.Query[struct {
		_    byke.Added[Body]
		Body *Body

		Collider           *Collider
		ColliderDensity    ColliderDensity
		ColliderElasticity ColliderElasticity
		ColliderFriction   ColliderFriction
		IsSensor           byke.Has[Sensor]
	}],
) {
	for item := range bodiesQuery.Items() {
		var body *cp.Body

		switch {
		case item.Body.static:
			body = cp.NewStaticBody()
		case item.Body.kinematic:
			body = cp.NewKinematicBody()
		default:
			body = cp.NewBody(0, 0)
		}

		shape := item.Collider.Shape.MakeShape(body)
		shape.SetDensity(item.ColliderDensity.Value)
		shape.SetElasticity(item.ColliderElasticity.Value)
		shape.SetFriction(item.ColliderFriction.Value)
		shape.SetSensor(item.IsSensor.Exists())
		body.AddShape(shape)

		space.AddShape(shape)
		space.AddBody(body)

		item.Body.body = body
		item.Collider.shape = shape
	}
}

func preStepSyncSystem(
	bodiesQuery byke.Query[struct {
		Body      *Body
		Velocity  Velocity
		Transform bykebiten.GlobalTransform
		Mass      byke.Option[Mass]
		Moment    byke.Option[Moment]
		Forces    ExternalForces
	}],
) {
	for item := range bodiesQuery.Items() {
		body := item.Body.body

		if body.Velocity() != cp.Vector(item.Velocity.Linear) {
			body.SetVelocityVector(cp.Vector(item.Velocity.Linear))
		}

		if body.AngularVelocity() != float64(item.Velocity.Angular) {
			body.SetAngularVelocity(float64(item.Velocity.Angular))
		}

		if !vecSimilar(body.Position(), cp.Vector(item.Transform.Translation)) {
			body.SetPosition(cp.Vector(item.Transform.Translation))
		}

		if body.Angle() != float64(item.Transform.Rotation) {
			body.SetAngle(float64(item.Transform.Rotation))
		}

		if item.Forces.Torque != 0 {
			body.SetTorque(item.Forces.Torque)
		}

		if f := cp.Vector(item.Forces.Linear); f != (cp.Vector{}) {
			body.SetForce(f)
		}

		var mass Mass = item.Mass.OrZero()
		if mass.Value > 0 && body.Mass() != mass.Value {
			body.SetMass(mass.Value)
		}

		var moment Moment = item.Moment.OrZero()
		if moment.Value > 0 && body.Moment() != moment.Value {
			body.SetMoment(moment.Value)
		}
	}
}

func updateSpaceSystem(t byke.FixedTime, space cpSpace) {
	space.Step(t.DeltaSecs)
}

func vecSimilar(a, b cp.Vector) bool {
	return a.Near(b, 1e-9)
}
func postStepSyncSystem(
	bodiesQuery byke.Query[struct {
		Body      *Body
		Velocity  *Velocity
		Transform *bykebiten.Transform
	}],
) {
	for item := range bodiesQuery.Items() {
		b := item.Body.body

		item.Velocity.Linear = gm.Vec(b.Velocity())
		item.Velocity.Angular = gm.Rad(b.AngularVelocity())

		item.Transform.Translation = gm.Vec(b.Position())
		item.Transform.Rotation = gm.Rad(b.Angle())
	}
}
