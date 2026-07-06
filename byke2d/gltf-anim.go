package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/gltf"
)

func (sc *spawnContext) animationCurveOf(anim gltf.Animation, channel gltf.AnimationChannel) AnimationCurve {
	handle := &sc.Handle

	switch channel.Target.Path {
	case "translation":
		curve := vec3AnimationSampler(handle, anim, channel.Sampler)

		return &TypedAnimationCurve[glm.Vec3f]{
			Curve:    new(curve),
			Accessor: TranslationPropertyAccessor,
		}

	case "scale":
		curve := vec3AnimationSampler(handle, anim, channel.Sampler)

		return &TypedAnimationCurve[glm.Vec3f]{
			Curve:    new(curve),
			Accessor: ScalePropertyAccessor,
		}

	case "rotation":
		curve := quatAnimationSampler(handle, anim, channel.Sampler)

		return &TypedAnimationCurve[glm.Quat]{
			Curve:    new(curve),
			Accessor: RotationPropertyAccessor,
		}

	case "weights":
		meshId := sc.Handle.Nodes[channel.Target.Node].Mesh.Get()
		weightsCount := len(sc.Handle.Meshes[meshId].Weights)

		curve := weightsCurve(handle, anim, channel.Sampler, weightsCount)

		return &TypedAnimationCurve[[]float32]{
			Curve:    new(curve),
			Accessor: weightsAccessor,
		}

	default:
		panic(fmt.Errorf("unknown animationChannel path %q", channel.Target.Path))
	}
}

var weightsAccessor = FieldAccessor[[]float32, MorphWeights](
	func(comp *MorphWeights) *[]float32 { return &comp.Weights },
)

func weightsCurve(handle *gltfHandle, anim gltf.Animation, sid gltf.Ref, weightsCount int) KeyframeCurve[[]float32] {
	sampler := anim.Samplers[sid]
	timestamps := handle.Resolve(sampler.Input).([]float32)
	values := handle.Resolve(sampler.Output).([]float32)

	// convert to keyframes
	var keyframes []Keyframe[[]float32]
	for idx, timestamp := range timestamps {
		offset := idx * weightsCount
		keyframes = append(keyframes, Keyframe[[]float32]{
			Time:  timestamp,
			Value: values[offset : offset+weightsCount],
		})
	}

	return KeyframeCurve[[]float32]{
		Keys:         keyframes,
		Interpolator: &weightsInterpolator{},
		Easing:       &Linear{},
	}
}

type weightsInterpolator struct{}

func (w weightsInterpolator) Interpolate(a, b []float32, alpha float32) []float32 {
	var res []float32

	for idx := range a {
		val := (FloatInterpolator{}).Interpolate(a[idx], b[idx], alpha)
		res = append(res, val)
	}

	return res
}

func vec3AnimationSampler(handle *gltfHandle, anim gltf.Animation, sid gltf.Ref) KeyframeCurve[glm.Vec3f] {
	sampler := anim.Samplers[sid]
	timestamps := handle.Resolve(sampler.Input).([]float32)
	values := handle.Resolve(sampler.Output).([]glm.Vec3f)

	// convert to keyframes
	var keyframes []Keyframe[glm.Vec3f]
	for idx, timestamp := range timestamps {
		keyframes = append(keyframes, Keyframe[glm.Vec3f]{
			Time:  timestamp,
			Value: values[idx],
		})
	}

	return KeyframeCurve[glm.Vec3f]{
		Keys:         keyframes,
		Interpolator: &Vec3fInterpolator{},
		Easing:       &Linear{},
	}
}

func quatAnimationSampler(handle *gltfHandle, anim gltf.Animation, sid gltf.Ref) KeyframeCurve[glm.Quat] {
	sampler := anim.Samplers[sid]
	timestamps := handle.Resolve(sampler.Input).([]float32)
	values := handle.Resolve(sampler.Output).([]glm.Vec4f)

	// convert to keyframes
	var keyframes []Keyframe[glm.Quat]
	for idx, timestamp := range timestamps {
		keyframes = append(keyframes, Keyframe[glm.Quat]{
			Time:  timestamp,
			Value: glm.QuatOf(values[idx].XYZW()).Inverse(),
		})
	}

	return KeyframeCurve[glm.Quat]{
		Keys:         keyframes,
		Interpolator: &QuatInterpolator{},
		Easing:       &Linear{},
	}
}
