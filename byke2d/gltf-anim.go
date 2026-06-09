package byke2d

import (
	"fmt"

	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/gltf"
)

type gltfAnimation struct {
	Clip  AnimationClip
	Nodes map[gltf.Ref]AnimationTargetId
}

func gtlfConvertAnimation(handle *gltfHandle, anim gltf.Animation) gltfAnimation {
	var clip AnimationClip

	targetNodeId := func(id gltf.Ref) AnimationTargetId {
		path := fmt.Sprintf("%s-%d", anim.Name, id)
		return AnimationTargetIdOf(path)
	}

	mapping := map[gltf.Ref]AnimationTargetId{}

	for _, channel := range anim.Channels {
		targetId := targetNodeId(channel.Target.Node)
		mapping[channel.Target.Node] = targetId

		switch channel.Target.Path {
		case "translation":
			curve := vec3AnimationSampler(handle, anim, channel.Sampler)

			first := curve.Sample(0)
			handle.Nodes[channel.Target.Node].Translation = new([3]float32(first))

			clip.Add(targetId, &TypedAnimationCurve[glm.Vec3f]{
				Curve:    &curve,
				Accessor: TranslationPropertyAccessor,
			})

		case "scale":
			curve := vec3AnimationSampler(handle, anim, channel.Sampler)

			first := curve.Sample(0)
			handle.Nodes[channel.Target.Node].Scale = new([3]float32(first))

			clip.Add(targetId, &TypedAnimationCurve[glm.Vec3f]{
				Curve:    &curve,
				Accessor: ScalePropertyAccessor,
			})

		case "rotation":
			curve := quatAnimationSampler(handle, anim, channel.Sampler)

			first := curve.Sample(0).ToVec4()
			handle.Nodes[channel.Target.Node].Rotation = new([4]float32(first))

			clip.Add(targetId, &TypedAnimationCurve[glm.Quat]{
				Curve:    &curve,
				Accessor: RotationPropertyAccessor,
			})

		case "weights":
			// ignoring for now
		}
	}

	return gltfAnimation{
		Clip:  clip,
		Nodes: mapping,
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
			Value: glm.QuatOf(values[idx].XYZW()),
		})
	}

	return KeyframeCurve[glm.Quat]{
		Keys:         keyframes,
		Interpolator: &QuatInterpolator{},
		Easing:       &Linear{},
	}
}
