package byke2d

import (
	"crypto/sha1"
	"log/slog"
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

type Linear struct{}

func (Linear) Eval(t float32) float32 {
	return t
}

type EaseInQuad struct{}

func (EaseInQuad) Eval(t float32) float32 {
	return t * t
}

type Interpolator[T any] interface {
	// Interpolate between two values
	Interpolate(a, b T, alpha float32) T
}

type FloatInterpolator struct{}

func (FloatInterpolator) Interpolate(a, b float32, alpha float32) float32 {
	return a + (b-a)*alpha
}

type Vec3fInterpolator struct{}

func (Vec3fInterpolator) Interpolate(a, b glm.Vec3f, alpha float32) glm.Vec3f {
	return glm.Vec3f{
		a[0] + (b[0]-a[0])*alpha,
		a[1] + (b[1]-a[1])*alpha,
		a[2] + (b[2]-a[2])*alpha,
	}
}

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

type Keyframe[T any] struct {
	Time  float32
	Value T
}

type KeyframeCurve[T any] struct {
	Keys         []Keyframe[T]
	Interpolator Interpolator[T]

	// easing to apply to the value. Defaults to Linear if not set.
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

type ConstantCurve[T any] struct {
	Value T
}

func (c ConstantCurve[T]) Sample(_ float32) T {
	return c.Value
}

type FunctionCurve[T any] func(t float32) T

func (c FunctionCurve[T]) Sample(t float32) T {
	return c(t)
}

type PropertyAccessor[T any] interface {
	Set(entity byke.EntityRef, value T)
	Get(entity byke.EntityRef) (T, bool)
}

type fieldAccessorImpl[T any, C byke.IsComponent[C]] func(comp *C) *T

func fieldAccessor[T any, C byke.IsComponent[C]](f func(comp *C) *T) PropertyAccessor[T] {
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

var TranslationPropertyAccessor = fieldAccessor(func(comp *Transform) *glm.Vec3f {
	return &comp.Translation
})

var RotationPropertyAccessor = fieldAccessor(func(comp *Transform) *glm.Quat {
	return &comp.Rotation
})

var ScalePropertyAccessor = fieldAccessor(func(comp *Transform) *glm.Vec3f {
	return &comp.Scale
})

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

type TypedAnimationCurve[T any] struct {
	Accessor PropertyAccessor[T]
	Curve    Curve[T]
}

func (a *TypedAnimationCurve[T]) ApplyTo(ref byke.EntityRef, t float32) {
	a.Accessor.Set(ref, a.Curve.Sample(t))
}

type AnimationCurve interface {
	ApplyTo(ref byke.EntityRef, t float32)
}

type AnimationTargetId struct {
	byke.Component[AnimationTargetId]
	Id [16]byte
}

func AnimationTargetIdOf(value string) AnimationTargetId {
	hash := sha1.New()
	hash.Write([]byte(value))
	hashSum := hash.Sum(nil)

	var targetId AnimationTargetId
	copy(targetId.Id[:], hashSum[:20])

	return targetId
}

var _ = byke.ValidateComponent[AnimatedBy]()
var _ = byke.ValidateComponent[ActiveAnimation]()
var _ = byke.ValidateComponent[AnimationTargetId]()

type AnimatedBy struct {
	byke.Component[AnimatedBy]
	// Animator is the entity that holds the AnimationPlayer and
	// the current animation that has an AnimationTarget to the current entity.
	Animator byke.EntityId
}

type ActiveAnimation struct {
	byke.Component[ActiveAnimation]
	Time      float32
	Animation AnimationClip
}

type AnimationClip struct {
	animations map[AnimationTargetId][]AnimationCurve
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
}]) {
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
