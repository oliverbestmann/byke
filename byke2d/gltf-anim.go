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

		first := curve.Sample(0)
		handle.Nodes[channel.Target.Node].Translation = new([3]float32(first))

		return &TypedAnimationCurve[glm.Vec3f]{
			Curve:    &curve,
			Accessor: TranslationPropertyAccessor,
		}

	case "scale":
		curve := vec3AnimationSampler(handle, anim, channel.Sampler)

		first := curve.Sample(0)
		handle.Nodes[channel.Target.Node].Scale = new([3]float32(first))

		return &TypedAnimationCurve[glm.Vec3f]{
			Curve:    &curve,
			Accessor: ScalePropertyAccessor,
		}

	case "rotation":
		curve := quatAnimationSampler(handle, anim, channel.Sampler)

		first := curve.Sample(0).ToVec4()
		handle.Nodes[channel.Target.Node].Rotation = new([4]float32(first))

		return &TypedAnimationCurve[glm.Quat]{
			Curve:    &curve,
			Accessor: RotationPropertyAccessor,
		}

	case "weights":
		// ignoring for now
		return nil

	default:
		panic(fmt.Errorf("unknown animationChannel path %q", channel.Target.Path))
	}
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
