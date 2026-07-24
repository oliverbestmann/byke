package byke2d

import (
	"crypto/sha1"
	"log/slog"
	"maps"
	"sort"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/spoke"
)

type Curve[T any] interface {
	// Sample samples the curve at the given time value t
	Sample(t float32) T
}

type Easing interface {
	// Eval evaluates the easing functon at the given value t.
	Eval(t float32) float32
}

// Linear is an easing function that returns the input value unchanged.
type Linear struct{}

func (Linear) Eval(t float32) float32 {
	return t
}

// EaseInQuad is an easing function that accelerates from zero velocity.
type EaseInQuad struct{}

func (EaseInQuad) Eval(t float32) float32 {
	return t * t
}

type Interpolator[T any] interface {
	// Interpolate between two values
	Interpolate(a, b T, alpha float32) T
}

// FloatInterpolator implements linear interpolation for float32 values.
type FloatInterpolator struct{}

func (FloatInterpolator) Interpolate(a, b float32, alpha float32) float32 {
	return a + (b-a)*alpha
}

// Vec3fInterpolator implements linear interpolation for Vec3f values.
type Vec3fInterpolator struct{}

func (Vec3fInterpolator) Interpolate(a, b glm.Vec3f, alpha float32) glm.Vec3f {
	return glm.Vec3f{
		a[0] + (b[0]-a[0])*alpha,
		a[1] + (b[1]-a[1])*alpha,
		a[2] + (b[2]-a[2])*alpha,
	}
}

// QuatInterpolator implements spherical linear interpolation for quaternion values.
type QuatInterpolator struct{}

func (QuatInterpolator) Interpolate(a, b glm.Quat, alpha float32) glm.Quat {
	aVec := a.ToVec4()
	bVec := b.ToVec4()

	if aVec.Dot(bVec) < 0 {
		bVec = bVec.Scale(-1)
	}

	aVec = aVec.Scale(1 - alpha)
	bVec = bVec.Scale(alpha)

	return glm.QuatOf(aVec.Add(bVec).Normalize().XYZW())
}

// Keyframe represents a value at a specific time in an animation.
type Keyframe[T any] struct {
	// Time is the time position of this keyframe.
	Time float32

	// Value is the animated value at this keyframe.
	Value T
}

// KeyframeCurve is a curve that interpolates values between keyframes.
type KeyframeCurve[T any] struct {
	// Keys is the sorted list of keyframes.
	Keys []Keyframe[T]

	// Interpolator defines how to interpolate between values.
	Interpolator Interpolator[T]

	// Easing is the easing function to apply to the interpolation. Defaults to Linear if not set.
	Easing Easing
}

func (c *KeyframeCurve[T]) Sample(t float32) T {
	if len(c.Keys) == 0 {
		panic("no keyframe defined")
	}

	if len(c.Keys) == 1 {
		return c.Keys[0].Value
	}

	// find keyframe using binary search
	i := sort.Search(len(c.Keys), func(i int) bool {
		return c.Keys[i].Time >= t
	})

	if i == 0 {
		return c.Keys[0].Value
	}

	if i == len(c.Keys) {
		return c.Keys[len(c.Keys)-1].Value
	}

	// interpolate between two keyframes using configured easing function
	a := c.Keys[i-1]
	b := c.Keys[i]

	alpha := (t - a.Time) / (b.Time - a.Time)

	if c.Easing != nil {
		alpha = c.Easing.Eval(alpha)
	}

	return c.Interpolator.Interpolate(a.Value, b.Value, alpha)
}

// ConstantCurve is a curve that always returns the same value.
type ConstantCurve[T any] struct {
	// Value is the constant value returned by this curve.
	Value T
}

func (c ConstantCurve[T]) Sample(_ float32) T {
	return c.Value
}

// FunctionCurve is a curve implemented by a function.
type FunctionCurve[T any] func(t float32) T

func (c FunctionCurve[T]) Sample(t float32) T {
	return c(t)
}

// PropertyAccessor defines how to get and set a property on an entity.
type PropertyAccessor[T any] interface {
	// Set assigns a value to the property on the given entity.
	Set(entity byke.EntityRef, value T)
	// Get retrieves the value of the property from the given entity.
	// It returns the value and a boolean indicating if the property exists.
	Get(entity byke.EntityRef) (T, bool)
}

type fieldAccessorImpl[T any, C byke.IsComponent[C]] func(comp *C) *T

// FieldAccessor creates a PropertyAccessor that accesses a field within a component.
func FieldAccessor[T any, C byke.IsComponent[C]](f func(comp *C) *T) PropertyAccessor[T] {
	return fieldAccessorImpl[T, C](f)
}

func (pa fieldAccessorImpl[T, C]) Set(entity byke.EntityRef, value T) {
	compValue := entity.Get(spoke.ComponentTypeOf[C]())
	if compValue == nil {
		return
	}

	ref := pa(any(compValue).(*C))
	*ref = value
}

func (pa fieldAccessorImpl[T, C]) Get(entity byke.EntityRef) (T, bool) {
	compValue := entity.Get(spoke.ComponentTypeOf[C]())
	if compValue == nil {
		var tZero T
		return tZero, false
	}

	ref := pa(any(compValue).(*C))
	return *ref, true
}

// TranslationPropertyAccessor accesses the Translation field of a Transform component.
var TranslationPropertyAccessor = FieldAccessor(func(comp *Transform) *glm.Vec3f {
	return &comp.Translation
})

// RotationPropertyAccessor accesses the Rotation field of a Transform component.
var RotationPropertyAccessor = FieldAccessor(func(comp *Transform) *glm.Quat {
	return &comp.Rotation
})

// ScalePropertyAccessor accesses the Scale field of a Transform component.
var ScalePropertyAccessor = FieldAccessor(func(comp *Transform) *glm.Vec3f {
	return &comp.Scale
})

// ComponentPropertyAccessor implements PropertyAccessor by treating the entire component as the value.
type ComponentPropertyAccessor[T byke.IsComponent[T]] struct{}

func (ComponentPropertyAccessor[T]) Set(entity byke.EntityRef, value T) {
	tr := entity.Get(spoke.ComponentTypeOf[T]())
	if tr == nil {
		return
	}

	*any(tr).(*T) = value
}

func (ComponentPropertyAccessor[T]) Get(entity byke.EntityRef) (T, bool) {
	tr := entity.Get(spoke.ComponentTypeOf[T]())
	if tr == nil {
		var tZero T
		return tZero, false
	}

	return *any(tr).(*T), true
}

// TypedAnimationCurve animates a property on an entity using a curve.
type TypedAnimationCurve[T any] struct {
	// Accessor defines how to get and set the property on the entity.
	Accessor PropertyAccessor[T]

	// Curve defines the animation curve to apply.
	Curve Curve[T]
}

func (a *TypedAnimationCurve[T]) ApplyTo(ref byke.EntityRef, t float32) {
	a.Accessor.Set(ref, a.Curve.Sample(t))
}

// AnimationCurve is a curve that can be applied to an entity at a given time.
type AnimationCurve interface {
	// ApplyTo applies the animation curve to the given entity at time t.
	ApplyTo(ref byke.EntityRef, t float32)
}

// AnimationTargetId identifies an entity as a target for animations.
type AnimationTargetId struct {
	byke.Component[AnimationTargetId]

	// Id is the hash identifier for this animation target.
	Id [16]byte
}

// AnimationTargetIdOf creates an AnimationTargetId from a string value.
func AnimationTargetIdOf(value string) AnimationTargetId {
	hash := sha1.New()
	hash.Write([]byte(value))
	hashSum := hash.Sum(nil)

	var targetId AnimationTargetId
	copy(targetId.Id[:], hashSum[:20])

	return targetId
}

var (
	_ = byke.ValidateComponent[AnimatedBy]()
	_ = byke.ValidateComponent[ActiveAnimation]()
	_ = byke.ValidateComponent[AnimationTargetId]()
)

// AnimatedBy marks an entity as being animated by another entity.
type AnimatedBy struct {
	byke.Component[AnimatedBy]

	// Animator is the entity that holds the AnimationPlayer and the current animation that has an AnimationTarget to the current entity.
	Animator byke.EntityId
}

// ActiveAnimation holds the current state of a playing animation clip.
type ActiveAnimation struct {
	byke.Component[ActiveAnimation]

	// Time is the current playback time of the animation in seconds.
	Time float32

	// Animation is the animation clip being played.
	Animation AnimationClip
}

// AnimationClip is a collection of animation curves targeting different entities.
type AnimationClip struct {
	animations map[AnimationTargetId][]AnimationCurve
}

func (clip *AnimationClip) MergeWith(other AnimationClip) AnimationClip {
	cloned := maps.Clone(clip.animations)
	if cloned == nil {
		cloned = map[AnimationTargetId][]AnimationCurve{}
	}

	maps.Insert(cloned, maps.All(other.animations))
	return AnimationClip{animations: cloned}
}

func (clip *AnimationClip) IsEmpty() bool {
	return len(clip.animations) == 0
}

func (clip *AnimationClip) Add(target AnimationTargetId, curve AnimationCurve) {
	if clip.animations == nil {
		clip.animations = map[AnimationTargetId][]AnimationCurve{}
	}

	clip.animations[target] = append(clip.animations[target], curve)
}

func (clip *AnimationClip) Curves(target AnimationTargetId) []AnimationCurve {
	return clip.animations[target]
}

func pluginAnimations(app *byke.App) {
	app.AddSystems(byke.Update, byke.
		System(addAnimatedBySystem, animationAdvanceTimeSystem, applyAnimationValuesSystem).
		Chain())
}

func addAnimatedBySystem(
	commands *byke.Commands,
	addedTargetsQuery byke.Query[struct {
		_        byke.Added[AnimationTargetId]
		_        byke.Without[AnimatedBy]
		EntityId byke.EntityId
		Parent   byke.ChildOf
	}],
	parentQuery byke.Query[struct {
		Parent byke.Option[byke.ChildOf]
		Clip   byke.Has[ActiveAnimation]
	}],
) {
	var recurse func(node, parentId byke.EntityId)

	recurse = func(node, parentId byke.EntityId) {
		entity, ok := parentQuery.Get(parentId)
		if !ok {
			return
		}

		if !entity.Clip.Exists() {
			parent, ok := entity.Parent.Get()
			if ok {
				slog.Warn("Found a root but no animation for target")
				recurse(node, parent.Parent)
			}

			return
		}

		// found the node with the animation
		commands.
			Entity(node).
			Insert(AnimatedBy{Animator: parentId})
	}

	for target := range addedTargetsQuery.Items() {
		recurse(target.EntityId, target.EntityId)
	}
}

func animationAdvanceTimeSystem(vt byke.VirtualTime, animations byke.Query[struct {
	Animation *ActiveAnimation
}],
) {
	for animation := range animations.Items() {
		animation.Animation.Time += vt.DeltaSecs * 0.5
	}
}

func applyAnimationValuesSystem(
	animationsQuery byke.Query[struct {
		EntityId  byke.EntityId
		Animation *ActiveAnimation
	}],
	animatedQuery byke.Query[struct {
		Entity            byke.EntityRef
		AnimatedBy        AnimatedBy
		AnimationTargetId AnimationTargetId
	}],
) {
	animations := map[byke.EntityId]*ActiveAnimation{}
	for animation := range animationsQuery.Items() {
		animations[animation.EntityId] = animation.Animation
	}

	for animated := range animatedQuery.Items() {
		animation, ok := animations[animated.AnimatedBy.Animator]
		if !ok {
			continue
		}

		curves := animation.Animation.Curves(animated.AnimationTargetId)
		for _, curve := range curves {
			curve.ApplyTo(animated.Entity, animation.Time)
		}
	}
}
