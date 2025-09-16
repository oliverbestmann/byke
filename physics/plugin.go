package physics

import (
	"math"

	b2 "github.com/oliverbestmann/box2d-go"
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
}

type b2World struct {
	b2.World
}

func init() {
}

func Plugin(app *byke.App) {
	world := b2.CreateWorld(b2.DefaultWorldDef())

	app.InsertResource(b2World{world})
	app.InsertResource(Gravity{Value: gm.Vec{Y: -10}})
	app.InsertResource(CollisionSlop{Value: 0.1})
	app.InsertResource(Dampening{Value: 1.0})
	app.InsertResource(Stepping{
		NumberOfSubsteps: 4,
	})

	app.InsertResource(entityIndex{
		Shapes: map[byke.EntityId]b2.Shape{},
		Bodies: map[byke.EntityId]b2.Body{},
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

	/*
		handler := world.NewWildcardCollisionHandler(0)

		handler.BeginFunc = func(arb *cp.Arbiter, space *cp.Space, userData interface{}) bool {
			a, b := arb.Bodies()

			entityA := a.UserData.(byke.EntityId)
			entityB := b.UserData.(byke.EntityId)
			if entityB >= entityA {
				return true
			}

			contactSet := arb.ContactPointSet()

			app.World().RunSystemWithInValue(handleCollisionStarted, CollisionStarted{
				A: entityA,
				B: entityB,
				// Arbiter:  arb,
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
				A: entityA,
				B: entityB,
				// Arbiter:  arb,
				Normal:   gm.Vec(contactSet.Normal),
				Position: gm.Vec(contactSet.Points[0].PointA),
			})

			return
		}
	*/
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
			Other: ev.B,
			// Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}

	if ok2 {
		commands.Entity(ev.B).Trigger(OnCollisionStarted{
			Other: ev.A,
			// Arbiter:  ev.Arbiter,
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
			Other: ev.B,
			// Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}

	if ok2 {
		commands.Entity(ev.B).Trigger(OnCollisionEnded{
			Other: ev.A,
			// Arbiter:  ev.Arbiter,
			Normal:   ev.Normal,
			Position: ev.Position,
		})
	}
}

func makeBodySystem(
	world b2World,
	index *entityIndex,
	bodiesQuery byke.Query[struct {
		_ byke.Added[Body]

		byke.EntityId
		Body     *Body
		Collider *Collider
	}],
) {
	for item := range bodiesQuery.Items() {
		var userData uintptr = uintptr(item.EntityId)

		bodyDef := b2.DefaultBodyDef()

		switch {
		case item.Body.static:
			bodyDef.Type1 = b2.StaticBody
		case item.Body.kinematic:
			bodyDef.Type1 = b2.KinematicBody
		default:
			bodyDef.Type1 = b2.DynamicBody
		}

		// add user data so we can identify the body later
		bodyDef.UserData = userData
		body := world.CreateBody(bodyDef)

		shapeDef := b2.DefaultShapeDef()
		shapeDef.UserData = userData
		shape := item.Collider.Shape.MakeShape(body, shapeDef)

		item.Body.body = body
		item.Collider.shape = shape

		if body, ok := index.Bodies[item.EntityId]; ok {
			body.DestroyBody()
		}

		if shape, ok := index.Shapes[item.EntityId]; ok {
			shape.DestroyShape(1)
		}

		// keep a reverse mapping so we can cleanup on entity despawn
		index.Bodies[item.EntityId] = body
		index.Shapes[item.EntityId] = shape
	}
}

func preStepSyncResourcesSystem(
	world b2World,
	gravity Gravity,
	collisionSlop CollisionSlop,
	dampening Dampening,
) {
	if !vecSimilar(world.GetGravity(), b2VecOf(gravity.Value)) {
		world.SetGravity(b2VecOf(gravity.Value))
	}

	// TODO
	// world.SetDamping(dampening.Value)
	// world.SetCollisionSlop(collisionSlop.Value)
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
		shape := item.Collider.shape

		if shape.GetDensity() != float32(item.ColliderDensity.Value) {
			shape.SetDensity(float32(item.ColliderDensity.Value), 1)
		}

		if shape.GetRestitution() != float32(item.ColliderElasticity.Value) {
			shape.SetRestitution(float32(item.ColliderElasticity.Value))
		}

		if shape.GetFriction() != float32(item.ColliderFriction.Value) {
			shape.SetFriction(float32(item.ColliderFriction.Value))
		}

		// TODO
		// if (shape.IsSensor() != 0) != item.IsSensor.Exists() {
		// 	shape.IsSensor(item.IsSensor.Exists())
		// }

		var filter b2.Filter
		filter.GroupIndex = item.ShapeFilter.Group
		filter.CategoryBits = item.ShapeFilter.Categories
		filter.MaskBits = item.ShapeFilter.Mask

		if shape.GetFilter() != filter {
			shape.SetFilter(filter)
		}
	}

	for entityId := range removedSensors.Read() {
		_, ok := shapesQuery.Get(entityId)
		if !ok {
			continue
		}

		// TODO
		//shape.Collider.shape.SetSensor(false)
	}
}

func preStepSyncBodiesSystem(
	bodiesQuery byke.Query[struct {
		Body      *Body
		Velocity  Velocity
		Transform bykebiten.GlobalTransform
		Mass      byke.Option[Mass]
		Forces    ExternalForces
	}],
) {
	for item := range bodiesQuery.Items() {
		body := item.Body.body

		if body.GetLinearVelocity() != b2VecOf(item.Velocity.Linear) {
			body.SetLinearVelocity(b2VecOf(item.Velocity.Linear))
		}

		if body.GetAngularVelocity() != float32(item.Velocity.Angular) {
			body.SetAngularVelocity(float32(item.Velocity.Angular))
		}

		if !vecSimilar(body.GetPosition(), b2VecOf(item.Transform.Translation)) ||
			body.GetRotation().Angle() != float32(item.Transform.Rotation) {

			pos := b2VecOf(item.Transform.Translation)
			sin, cos := item.Transform.Rotation.SinCos()
			rot := b2.Rot{C: float32(cos), S: float32(sin)}
			body.SetTransform(pos, rot)
		}

		if item.Forces.Torque != 0 {
			body.ApplyTorque(float32(item.Forces.Torque), 1)
		}

		if f := b2VecOf(item.Forces.Linear); f != (b2.Vec2{}) {
			body.ApplyForceToCenter(f, 1)
		}

		if mass_, ok := item.Mass.Get(); ok {
			var mass Mass = mass_

			md := b2.MassData{
				Mass:              float32(mass.Mass),
				RotationalInertia: float32(mass.RotationalInertia),
				Center:            b2VecOf(mass.Center),
			}

			if body.GetMassData() != md {
				body.SetMassData(md)
			}
		}
	}
}

func updateSpaceSystem(t byke.FixedTime, world b2World, steps Stepping) {
	world.Step(float32(t.DeltaSecs), int32(steps.NumberOfSubsteps))
}

func postStepSyncSystem(
	world b2World,
	bodiesQuery byke.Query[struct {
		Body      *Body
		Velocity  *Velocity
		Transform *bykebiten.Transform
	}],
) {
	events := world.GetBodyEvents()
	for _, event := range events.MoveEvents {
		entityId := byke.EntityId(event.UserData)

		body, ok := bodiesQuery.Get(entityId)
		if !ok {
			continue
		}

		b := body.Body.body
		body.Velocity.Linear = toVec(b.GetLinearVelocity())
		body.Velocity.Angular = gm.Rad(b.GetAngularVelocity())

		body.Transform.Translation = toVec(b.GetPosition())
		body.Transform.Rotation = gm.Rad(b.GetRotation().Angle())
	}
}

type entityIndex struct {
	Shapes map[byke.EntityId]b2.Shape
	Bodies map[byke.EntityId]b2.Body
}

func postStepRemoveSystem(
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

		body.DestroyBody()
	}

	for entityId := range removedColliders.Read() {
		shape, ok := index.Shapes[entityId]
		if !ok {
			continue
		}

		delete(index.Shapes, entityId)
		shape.DestroyShape(1)
	}
}

func vecSimilar(a, b b2.Vec2) bool {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return math.Sqrt(float64(dx*dx+dy*dy)) < 1e-7
}

func toVec(v b2.Vec2) gm.Vec {
	return gm.Vec{
		X: float64(v.X),
		Y: float64(v.Y),
	}
}
