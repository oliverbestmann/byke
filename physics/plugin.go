package physics

import (
	"github.com/jakecoffman/cp/v2"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/internal/query"
)

type Gravity struct {
	Value gm.Vec
}

type Dampening struct {
	Value float64
}

type CollisionSlop struct {
	Value float64
}

type Stepping struct {
	NumberOfSubsteps uint
	SolverIterations uint
}

type cpSpace struct {
	*cp.Space
}

func Plugin(app *byke.App) {
	space := cp.NewSpace()

	app.InsertResource(cpSpace{space})
	app.InsertResource(Gravity{Value: gm.Vec{Y: -10}})
	app.InsertResource(CollisionSlop{Value: 0.1})
	app.InsertResource(Dampening{Value: 1.0})
	app.InsertResource(Stepping{
		NumberOfSubsteps: 3,
		SolverIterations: 6,
	})

	app.InsertResource(entityIndex{
		Shapes: map[byke.EntityId]*cp.Shape{},
		Bodies: map[byke.EntityId]*cp.Body{},
	})

	app.AddEvent(byke.EventType[CollisionStarted]())
	app.AddEvent(byke.EventType[CollisionEnded]())

	app.AddSystems(byke.FixedUpdate, byke.System(
		makeBodySystem,
		byke.System(preStepSyncBodiesSystem, preStepSyncShapesSystem, preStepSyncResourcesSystem),
		updateSpaceSystem,
		postStepRemoveSystem,
		postStepSyncSystem,
	).Chain())

	app.AddSystems(byke.PostRender, debugSystem)

	handler := space.NewWildcardCollisionHandler(0)

	handler.BeginFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
		a, b := arb.Bodies()

		entityA := a.UserData.(byke.EntityId)
		entityB := b.UserData.(byke.EntityId)
		if entityB >= entityA {
			return true
		}

		contactSet := arb.ContactPointSet()

		app.World().RunSystemWithInValue(handleCollisionStarted, CollisionStarted{
			A:        entityA,
			B:        entityB,
			Arbiter:  arb,
			Normal:   gm.Vec(contactSet.Normal),
			Position: gm.Vec(contactSet.Points[0].PointA),
		})

		return true
	}

	handler.SeparateFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) {
		a, b := arb.Bodies()

		entityA := a.UserData.(byke.EntityId)
		entityB := b.UserData.(byke.EntityId)
		if entityB >= entityA {
			return
		}

		contactSet := arb.ContactPointSet()

		app.World().RunSystemWithInValue(handleCollisionEnded, CollisionEnded{
			A:        entityA,
			B:        entityB,
			Arbiter:  arb,
			Normal:   gm.Vec(contactSet.Normal),
			Position: gm.Vec(contactSet.Points[0].PointA),
		})

		return
	}
}

func handleCollisionStarted(
	commands *byke.Commands,
	params byke.In[CollisionStarted],
	events *byke.EventWriter[CollisionStarted],
	hasMarkerQuery byke.Query[query.With[CollisionEventsEnabled]],
) {
	ev := params.Value

	_, ok1 := hasMarkerQuery.Get(ev.A)
	_, ok2 := hasMarkerQuery.Get(ev.B)

	if !ok1 && !ok2 {
		return
	}

	events.Write(ev)

	if ok1 {
		commands.Entity(ev.A).Trigger(OnCollisionStarted{
			Other:    ev.B,
			Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}

	if ok2 {
		commands.Entity(ev.B).Trigger(OnCollisionStarted{
			Other:    ev.A,
			Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}
}

func handleCollisionEnded(
	commands *byke.Commands,
	params byke.In[CollisionEnded],
	events *byke.EventWriter[CollisionEnded],
	hasMarkerQuery byke.Query[query.With[CollisionEventsEnabled]],
) {
	ev := params.Value

	_, ok1 := hasMarkerQuery.Get(ev.A)
	_, ok2 := hasMarkerQuery.Get(ev.B)

	if !ok1 && !ok2 {
		return
	}

	events.Write(ev)

	if ok1 {
		commands.Entity(ev.A).Trigger(OnCollisionEnded{
			Other:    ev.B,
			Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}

	if ok2 {
		commands.Entity(ev.B).Trigger(OnCollisionEnded{
			Other:    ev.A,
			Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}
}

func makeBodySystem(
	space cpSpace,
	index *entityIndex,
	bodiesQuery byke.Query[struct {
		_ byke.Added[Body]

		byke.EntityId
		Body     *Body
		Collider *Collider
	}],
) {
	for item := range bodiesQuery.Items() {
		var userData any = item.EntityId

		var body *cp.Body

		switch {
		case item.Body.static:
			body = cp.NewStaticBody()
		case item.Body.kinematic:
			body = cp.NewKinematicBody()
		default:
			body = cp.NewBody(0, 0)
		}

		// add user data so we can identify the body later
		body.UserData = userData

		shape := item.Collider.Shape.MakeShape(body)
		shape.UserData = userData
		body.AddShape(shape)

		space.AddShape(shape)
		space.AddBody(body)

		item.Body.body = body
		item.Collider.shape = shape

		if body, ok := index.Bodies[item.EntityId]; ok {
			space.RemoveBody(body)
		}

		if shape, ok := index.Shapes[item.EntityId]; ok {
			space.RemoveShape(shape)
		}

		// keep a reverse mapping so we can cleanup on entity despawn
		index.Bodies[item.EntityId] = body
		index.Shapes[item.EntityId] = shape
	}
}

func preStepSyncResourcesSystem(
	space cpSpace,
	gravity Gravity,
	collisionSlop CollisionSlop,
	dampening Dampening,
) {
	if !vecSimilar(space.Gravity(), cp.Vector(gravity.Value)) {
		space.SetGravity(cp.Vector(gravity.Value))
	}

	space.SetDamping(dampening.Value)
	space.SetCollisionSlop(collisionSlop.Value)
}

func preStepSyncShapesSystem(
	shapesQuery byke.Query[struct {
		_ byke.OrStruct[struct {
			_ byke.Changed[ColliderFriction]
			_ byke.Changed[ColliderElasticity]
			_ byke.Changed[ColliderDensity]
			_ byke.Changed[ShapeFilter]
			_ byke.Added[Sensor]
		}]

		Collider           *Collider
		ColliderFriction   ColliderFriction
		ColliderElasticity ColliderElasticity
		ColliderDensity    ColliderDensity
		ShapeFilter        ShapeFilter
		IsSensor           byke.Has[Sensor]
	}],

	removedSensors byke.RemovedComponents[Sensor],
) {
	for item := range shapesQuery.Items() {
		cpShape := item.Collider.shape

		if cpShape.Density() != item.ColliderDensity.Value {
			cpShape.SetDensity(item.ColliderDensity.Value)
		}

		if cpShape.Elasticity() != item.ColliderElasticity.Value {
			cpShape.SetElasticity(item.ColliderElasticity.Value)
		}

		if cpShape.Friction() != item.ColliderFriction.Value {
			cpShape.SetFriction(item.ColliderFriction.Value)
		}

		if cpShape.Sensor() != item.IsSensor.Exists() {
			cpShape.SetSensor(item.IsSensor.Exists())
		}

		cpShape.Filter.Group = item.ShapeFilter.Group
		cpShape.Filter.Categories = item.ShapeFilter.Categories
		cpShape.Filter.Mask = item.ShapeFilter.Mask
	}

	for entityId := range removedSensors.Read() {
		cpShape, ok := shapesQuery.Get(entityId)
		if !ok {
			continue
		}

		cpShape.Collider.shape.SetSensor(false)
	}
}

func preStepSyncBodiesSystem(
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

func updateSpaceSystem(t byke.FixedTime, space cpSpace, steps Stepping) {
	space.Iterations = steps.SolverIterations

	for range steps.NumberOfSubsteps {
		space.Step(t.DeltaSecs / float64(steps.NumberOfSubsteps))
	}
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

type entityIndex struct {
	Shapes map[byke.EntityId]*cp.Shape
	Bodies map[byke.EntityId]*cp.Body
}

func postStepRemoveSystem(
	space cpSpace,
	index *entityIndex,
	removedBodies byke.RemovedComponents[Body],
	removedColliders byke.RemovedComponents[Collider],
) {
	for entityId := range removedBodies.Read() {
		body, ok := index.Bodies[entityId]
		if !ok {
			continue
		}

		delete(index.Bodies, entityId)

		space.RemoveBody(body)
	}

	for entityId := range removedColliders.Read() {
		shape, ok := index.Shapes[entityId]
		if !ok {
			continue
		}

		delete(index.Shapes, entityId)

		space.RemoveShape(shape)
	}
}

func vecSimilar(a, b cp.Vector) bool {
	return a.Near(b, 1e-9)
}
