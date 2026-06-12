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

func (s SkinnedMesh) IsValid() bool {
	return len(s.Joints) > 0
}

type skinUniforms struct {
	staging wgsl.StructWriter
	buffer  *wgpu.Buffer
	offsets map[byke.EntityId]uint32
}

func (s *skinUniforms) OffsetOf(entity byke.EntityId) (uint32, bool) {
	offset, ok := s.offsets[entity]
	return offset, ok
}

var SkinningBindGroupLayout = SequentialLayoutWithLabel("Skinning",
	BindingLayoutBuffer(wgpu.BufferBindingTypeUniform, true),
)

func prepareJointsForSkinSystem(
	ctx *RenderContext,
	uniforms *skinUniforms,
	jointsQuery byke.Query[GlobalTransform],
	meshesQuery byke.Query[struct {
		EntitId byke.EntityId
		Skin    SkinnedMesh
	}],
) {
	uniforms.offsets = map[byke.EntityId]uint32{}
	uniforms.staging.Clear()

outer:
	for mesh := range meshesQuery.Items() {
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

		uniforms.offsets[mesh.EntitId] = offset

		// TODO add padding to align with next uniform border
	}

	for range 16 {
		// fill with dummy values to reach minimum size
		uniforms.staging.AppendMat4f(glm.Mat4f{})
	}

	uniforms.staging.WriteTo(ctx, &uniforms.buffer, wgpu.BufferUsageUniform)
}

type SkinBindGroup struct {
	// has dynamic offset configured for the start of the joints array
	BindGroup *wgpu.BindGroup
	buffer    *wgpu.Buffer
}

func prepareSkinViewBindGroupSystem(
	ctx *RenderContext,
	bindGroups *SkinBindGroup,
	uniforms *skinUniforms,
	pipelines *PipelineCache,
) {
	if bindGroups.buffer == uniforms.buffer {
		return
	}

	bindGroups.BindGroup.Release()

	bindGroups.BindGroup = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   "Skinning",
		Layout:  pipelines.BindGroupLayout(SkinningBindGroupLayout),
		Entries: Sequential(BindingBuffer(uniforms.buffer)),
	})

	bindGroups.buffer = uniforms.buffer
}
