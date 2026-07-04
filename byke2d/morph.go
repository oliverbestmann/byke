package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type MorphWeights struct {
	byke.Component[MorphWeights]
	Names   []string
	Weights []float32
}

// meshMorphWeights are copied to all children of MorphWeights
// that have a Mesh3d attribute.
type meshMorphWeights struct {
	byke.Component[meshMorphWeights]
	Weights []float32
}

// MorphAttributes describes the offset for a morph target
// from its original vertex.
type MorphAttributes struct {
	Position glm.Vec3f
	Normal   glm.Vec3f
	Tangent  glm.Vec3f
}

func syncMeshMorphWeightsSystem(
	weightsQuery byke.Query[struct {
		MorphWeights MorphWeights
		Children     byke.Children

		// if not loaded from gltf, the actual weights might be on the same
		// level as the MorphWeights, lets apply them directly
		MeshMorphWeights byke.OptionMut[meshMorphWeights]
	}],
	meshes byke.Query[struct {
		_                byke.With[Mesh3d]
		MeshMorphWeights *meshMorphWeights
	}],
) {
	for parentItem := range weightsQuery.Items() {
		if meshWeights, ok := parentItem.MeshMorphWeights.Get(); ok {
			meshWeights.Weights = parentItem.MorphWeights.Weights
		}

		for _, child := range parentItem.Children.Children() {
			meshItem, _ := meshes.Get(child)
			meshItem.MeshMorphWeights.Weights = parentItem.MorphWeights.Weights
		}
	}
}

type morphUniforms struct {
	// contains a dynamic sized struct for each mesh instance
	//  * weights: array<f32>,
	BufWeights *wgpu.Buffer

	// contains MorphDescriptor instances
	BufDescriptors *wgpu.Buffer

	// temporaries for writing data
	wWeights     wgsl.StructWriter
	wDescriptors wgsl.StructWriter

	descOffsets map[byke.EntityId]uint32
}

// DescriptorIndex returns the descriptor index for the mesh on the given entity
func (u *morphUniforms) DescriptorIndex(entityId byke.EntityId) (uint32, bool) {
	offset, ok := u.descOffsets[entityId]
	return offset, ok
}

func prepareMorphUniformsSystem(
	ctx *RenderContext,
	uniforms *morphUniforms,
	meshAllocator *MeshAllocator,
	meshes byke.Query[struct {
		EntityId         byke.EntityId
		Mesh             Mesh3d
		MeshMorphWeights *meshMorphWeights
	}],
) {
	uniforms.descOffsets = map[byke.EntityId]uint32{}

	uniforms.wWeights.Clear()
	uniforms.wDescriptors.Clear()

	var descriptorIndex uint32
	for meshItem := range meshes.Items() {
		bufs, ok := meshAllocator.Get(meshItem.Mesh.Mesh)
		if !ok {
			panic("mesh not found")
		}

		if bufs.MorphAttributes == nil {
			panic("no morph attributes buffer for mesh")
		}

		// descriptor offsets
		uniforms.descOffsets[meshItem.EntityId] = descriptorIndex

		// write descriptor
		uniforms.wDescriptors.AppendUint(uint32(meshItem.Mesh.Mesh.MorphTargetCount()))
		uniforms.wDescriptors.AppendUint(uint32(meshItem.Mesh.Mesh.VertexCount()))
		uniforms.wDescriptors.AppendUint(uniforms.wWeights.Offset() / 4)
		uniforms.wDescriptors.AppendUint(bufs.MorphAttributesIndex)

		// write weights to storage buffer
		for _, weight := range meshItem.MeshMorphWeights.Weights {
			uniforms.wWeights.AppendFloat32(weight)
		}

		descriptorIndex += 1
	}

	// upload data to gpu
	uniforms.wDescriptors.WriteTo(ctx, &uniforms.BufDescriptors, "MorphDescriptors", wgpu.BufferUsageStorage)
	uniforms.wWeights.WriteTo(ctx, &uniforms.BufWeights, "MorphWeights", wgpu.BufferUsageStorage)
}
