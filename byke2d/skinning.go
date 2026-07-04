package byke2d

import (
	"log/slog"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[SkinnedMesh]()

type SkinnedMesh struct {
	byke.Component[SkinnedMesh]
	InverseBind []glm.Mat4f
	Joints      []byke.EntityId
}

type skinUniforms struct {
	BufJoints *wgpu.Buffer
	staging   wgsl.StructWriter
	offsets   map[byke.EntityId]uint32
}

func (s *skinUniforms) OffsetOf(entity byke.EntityId) (uint32, bool) {
	offset, ok := s.offsets[entity]
	return offset, ok
}

func prepareSkinsUniformsSystem(
	ctx *RenderContext,
	uniforms *skinUniforms,
	jointsQuery byke.Query[GlobalTransform],
	meshes *ExtractedMeshes3d,
) {
	const maxJoints = 256

	uniforms.offsets = map[byke.EntityId]uint32{}
	uniforms.staging.Clear()

outer:
	for _, mesh := range meshes.Meshes {
		if !mesh.Skin.IsSet() {
			continue
		}

		// TODO tightly pack into storage buffer
		uniforms.staging.AlignTo(256)
		offset := uniforms.staging.Offset()

		for idx, jointId := range mesh.Skin.Joints {
			joint, ok := jointsQuery.Get(jointId)
			if !ok {
				slog.Warn("Joint not found", slog.Any("entityId", jointId))
				continue outer
			}

			mat := joint.Affine.Mul(mesh.Skin.InverseBind[idx])
			uniforms.staging.AppendMat4f(mat)
		}

		uniforms.offsets[mesh.Skin.EntityId] = offset
	}

	for range maxJoints {
		// fill with dummy values to reach the array size
		uniforms.staging.AppendMat4f(glm.Mat4f{})
	}

	// upload to gpu
	uniforms.staging.WriteTo(ctx, &uniforms.BufJoints, "Joints", wgpu.BufferUsageUniform)
}
