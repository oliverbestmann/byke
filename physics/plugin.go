package physics

import (
	"math"

	b2 "github.com/oliverbestmann/box2d-go"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten"
	"github.com/oliverbestmann/byke/gm"
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

	app.AddEvent(byke.EventType[ContactStarted]())
	app.AddEvent(byke.EventType[ContactEnded]())
	app.AddEvent(byke.EventType[SensorStarted]())
	app.AddEvent(byke.EventType[SensorEnded]())

	app.AddSystems(byke.FixedUpdate, byke.System(
		makeBodySystem,
		byke.System(preStepSyncBodiesSystem, preStepSyncShapesSystem, preStepSyncResourcesSystem),
		updateSpaceSystem,
		postStepRemoveSystem,
		postStepSyncSystem,
		emitCollisionEventsSystem,
		emitSensorEventsSystem,
	).Chain())

	app.AddSystems(byke.PostRender, debugSystem)
}

func makeBodySystem(
	world b2World,
	index *entityIndex,
	bodiesQuery byke.Query[struct {
	_ byke.Added[Body]

	byke.EntityId
	Body     *Body
	Collider *Collider
	IsSensor byke.Has[Sensor]
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
		shapeDef.IsSensor = b2bool(item.IsSensor.Exists())
		shape := item.Collider.Shape.MakeShape(body, shapeDef)

		item.Body.body = body
		item.Collider.shape = shape

		if shape, ok := index.Shapes[item.EntityId]; ok {
			if shape.IsValid() != 0 {
				shape.DestroyShape(1)
			}
		}

		if body, ok := index.Bodies[item.EntityId]; ok {
			if body.IsValid() != 0 {
				body.DestroyBody()
			}
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
		_ byke.Added[ContactEventsEnabled]
		_ byke.Added[SensorEventsEnabled]
	}]

	Collider           *Collider
	ColliderFriction   ColliderFriction
	ColliderElasticity ColliderElasticity
	ColliderDensity    ColliderDensity
	ShapeFilter        ShapeFilter

	IsSensor             byke.Has[Sensor]
	ContactEventsEnabled byke.Has[ContactEventsEnabled]
	SensorEventsEnabled  byke.Has[SensorEventsEnabled]
}],

	removedContactEventsEnabled byke.RemovedComponents[ContactEventsEnabled],
	removedSensorEventsEnabled byke.RemovedComponents[SensorEventsEnabled],
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

		if (shape.IsSensor() != 0) != item.IsSensor.Exists() {
			// TODO recreate shape as (non) sensor
		}

		if (shape.AreContactEventsEnabled() != 0) != item.ContactEventsEnabled.Exists() {
			shape.EnableContactEvents(b2bool(item.ContactEventsEnabled.Exists()))
		}

		if (shape.AreSensorEventsEnabled() != 0) != item.SensorEventsEnabled.Exists() {
			shape.EnableSensorEvents(b2bool(item.SensorEventsEnabled.Exists()))
		}

		var filter b2.Filter
		filter.GroupIndex = item.ShapeFilter.Group
		filter.CategoryBits = item.ShapeFilter.Categories
		filter.MaskBits = item.ShapeFilter.Mask

		if shape.GetFilter() != filter {
			shape.SetFilter(filter)
		}
	}

	for entityId := range removedSensorEventsEnabled.Read() {
		shape, ok := shapesQuery.Get(entityId)
		if !ok || shape.Collider.shape.IsSensor() == 0 {
			continue
		}

		// TODO recreate shape as non-sensor
		shape.Collider.shape.EnableSensorEvents(0)
	}

	for entityId := range removedContactEventsEnabled.Read() {
		shape, ok := shapesQuery.Get(entityId)
		if !ok || shape.Collider.shape.AreContactEventsEnabled() == 0 {
			continue
		}

		shape.Collider.shape.EnableContactEvents(0)
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

func emitCollisionEventsSystem(
	commands *byke.Commands,
	world b2World,
	writerStarted *byke.EventWriter[ContactStarted],
	writerEnded *byke.EventWriter[ContactEnded],
	hasMarkerQuery byke.Query[byke.With[ContactEventsEnabled]],
) {
	events := world.GetContactEvents()

	for _, event := range events.BeginEvents {
		shapeA := b2.Shape{Id: event.ShapeIdA}
		shapeB := b2.Shape{Id: event.ShapeIdB}
		if shapeA.IsValid() == 0 || shapeB.IsValid() == 0 {
			continue
		}

		idA := byke.EntityId(shapeA.GetUserData())
		idB := byke.EntityId(shapeB.GetUserData())

		_, ok1 := hasMarkerQuery.Get(idA)
		_, ok2 := hasMarkerQuery.Get(idB)

		if !ok1 && !ok2 {
			continue
		}

		ev := ContactStarted{
			A:        idA,
			B:        idB,
			Position: toVec(event.Manifold.Points[0].Point),
			Normal:   toVec(event.Manifold.Normal),
		}

		writerStarted.Write(ev)

		if ok1 {
			commands.Entity(ev.A).Trigger(OnContactStarted{
				Other:    ev.B,
				Normal:   ev.Normal,
				Position: ev.Position,
			})
		}

		if ok2 {
			commands.Entity(ev.B).Trigger(OnContactStarted{
				Other:    ev.A,
				Normal:   ev.Normal,
				Position: ev.Position,
			})
		}
	}

	for _, event := range events.EndEvents {
		shapeA := b2.Shape{Id: event.ShapeIdA}
		shapeB := b2.Shape{Id: event.ShapeIdB}
		if shapeA.IsValid() == 0 || shapeB.IsValid() == 0 {
			continue
		}

		idA := byke.EntityId(shapeA.GetUserData())
		idB := byke.EntityId(shapeB.GetUserData())

		_, ok1 := hasMarkerQuery.Get(idA)
		_, ok2 := hasMarkerQuery.Get(idB)

		if !ok1 && !ok2 {
			continue
		}

		ev := ContactEnded{
			A: idA,
			B: idB,
		}

		writerEnded.Write(ev)

		if ok1 {
			commands.Entity(ev.A).Trigger(OnContactEnded{
				Other: ev.B,
			})
		}

		if ok2 {
			commands.Entity(ev.B).Trigger(OnContactEnded{
				Other: ev.A,
			})
		}
	}
}

func emitSensorEventsSystem(
	commands *byke.Commands,
	world b2World,
	writerStarted *byke.EventWriter[SensorStarted],
	writerEnded *byke.EventWriter[SensorEnded],
	hasMarkerQuery byke.Query[byke.With[SensorEventsEnabled]],
) {
	events := world.GetSensorEvents()

	for _, event := range events.BeginEvents {
		shapeA := b2.Shape{Id: event.SensorShapeId}
		shapeB := b2.Shape{Id: event.VisitorShapeId}
		if shapeA.IsValid() == 0 || shapeB.IsValid() == 0 {
			continue
		}

		idA := byke.EntityId(shapeA.GetUserData())
		idB := byke.EntityId(shapeB.GetUserData())

		_, ok1 := hasMarkerQuery.Get(idA)
		_, ok2 := hasMarkerQuery.Get(idB)

		if !ok1 && !ok2 {
			continue
		}

		bBBox := shapeB.GetAABB()
		bCenter := toVec(bBBox.LowerBound).Add(toVec(bBBox.UpperBound)).Mul(0.5)
		ev := SensorStarted{
			A:        idA,
			B:        idB,
			Position: toVec(shapeA.GetClosestPoint(b2VecOf(bCenter))),
		}

		writerStarted.Write(ev)

		if ok1 {
			commands.Entity(ev.A).Trigger(OnSensorStarted{
				Other:    ev.B,
				Position: ev.Position,
			})
		}

		if ok2 {
			commands.Entity(ev.B).Trigger(OnSensorStarted{
				Other:    ev.A,
				Position: ev.Position,
			})
		}
	}

	for _, event := range events.EndEvents {
		shapeA := b2.Shape{Id: event.SensorShapeId}
		shapeB := b2.Shape{Id: event.VisitorShapeId}
		if shapeA.IsValid() == 0 || shapeB.IsValid() == 0 {
			continue
		}

		idA := byke.EntityId(shapeA.GetUserData())
		idB := byke.EntityId(shapeB.GetUserData())

		_, ok1 := hasMarkerQuery.Get(idA)
		_, ok2 := hasMarkerQuery.Get(idB)

		if !ok1 && !ok2 {
			continue
		}

		ev := SensorEnded{
			A: idA,
			B: idB,
		}

		writerEnded.Write(ev)

		if ok1 {
			commands.Entity(ev.A).Trigger(OnSensorEnded{
				Other: ev.B,
			})
		}

		if ok2 {
			commands.Entity(ev.B).Trigger(OnSensorEnded{
				Other: ev.A,
			})
		}
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
	for entityId := range removedColliders.Read() {
		shape, ok := index.Shapes[entityId]
		if !ok {
			continue
		}

		delete(index.Shapes, entityId)
		shape.DestroyShape(1)
	}

	for entityId := range removedBodies.Read() {
		body, ok := index.Bodies[entityId]
		if !ok {
			continue
		}

		delete(index.Bodies, entityId)

		body.DestroyBody()
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

func b2bool(v bool) uint8 {
	if v {
		return 1
	} else {
		return 0
	}
}
