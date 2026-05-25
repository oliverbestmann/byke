package byke2d

import (
	"errors"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/gltf"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type Scene struct {
	Handle *gltf.Handle
	Index  int
}

func SceneRoot(world *byke.World, h *gltf.Handle, sceneId gltf.Ref) byke.ErasedComponent {
	ctx := byke.RequireResourceOf[RenderContext](world)

	var bundle []byke.ErasedComponent

	for _, node := range h.Scene(sceneId) {
		components := gltfConvert(ctx, h, node)
		if len(components) == 0 {
			continue
		}

		bundle = append(bundle, byke.SpawnChild(components...))
	}

	return byke.BundleOf(
		NewTransform(),
		InheritVisibility,
		byke.BundleOf(bundle...),
	)
}

func gltfConvert(ctx *RenderContext, h *gltf.Handle, node gltf.Node) []byke.ErasedComponent {
	m := node.Mesh
	if m.IsNil() {
		return nil
	}

	transform := gltfConvertTransform(node)

	// we'll return an intermediary node
	var bundle []byke.ErasedComponent
	bundle = append(bundle, InheritVisibility)
	bundle = append(bundle, transform)

	// spawn each mesh as a direct child
	for _, prim := range h.Meshes[m.Get()].Primitives {
		var material ColorMaterial

		if ma := prim.Material; ma.IsSet {
			material = gltfConvertMaterial(ctx, h, prim.Material.Get())
		}

		mesh := gltfConvertMesh(h, prim)
		bundle = append(bundle, byke.SpawnChild(
			Mesh2d{Mesh: mesh},
			material,
		))
	}

	// recurse into child nodes and spawn them as children
	for _, child := range h.ChildNodes(node) {
		components := gltfConvert(nil, h, child)
		if len(components) == 0 {
			continue
		}

		bundleChild := byke.SpawnChild(components...)
		bundle = append(bundle, bundleChild)
	}

	return bundle
}

func gltfConvertMaterial(ctx *RenderContext, h *gltf.Handle, ma gltf.Ref) ColorMaterial {
	var cm ColorMaterial

	mat := h.Materials[ma]
	cm.Tint = ColorOf(mat.BaseColor())

	if mr := mat.MetallicRoughness; mr != nil {
		if tex := mr.BaseColorTexture; tex != nil {
			bufView := h.Images[h.Textures[tex.Index].Source].BufferView

			texture, err := DecodeTextureFromMemory(ctx, h.Buffer(bufView), SamplerConfig{}, true)
			if err != nil {
				panic(err)
			}

			cm.Texture = texture
		}
	}

	return cm
}

func gltfConvertTransform(node gltf.Node) Transform {
	tr, scale, rot := node.TransformComponents()
	return Transform{
		Translation: tr,
		Scale:       scale,
		Rotation:    rot,
	}
}

func gltfConvertMesh(h *gltf.Handle, prim gltf.MeshPrimitive) *Mesh {
	if prim.Indices.IsNil() {
		panic(errors.New("can only load meshes with indices"))
	}

	// get and convert indices if necessary
	rawIndices := h.Resolve(prim.Indices.Get())
	indices := gltfConvertMeshIndices(rawIndices)

	vertices := h.Resolve(prim.MustGet("POSITION")).([]glm.Vec3f)
	mesh := MeshOf(indices, vertices)

	for key, value := range prim.Attributes {
		switch key {
		case "TEXCOORD_0":
			uv := h.Resolve(value).([]glm.Vec2f)
			mesh.WithAttributes(VertexAttributeUV, wgpu.ToBytes(uv))

		case "NORMAL":
			uv := h.Resolve(value).([]glm.Vec3f)
			mesh.WithAttributes(VertexAttributeNormal, wgpu.ToBytes(uv))
		}
	}

	return mesh
}

func gltfConvertMeshIndices(rawIndices any) []uint32 {
	if indices16, ok := rawIndices.([]uint16); ok {
		indices := make([]uint32, 0, len(indices16))
		for _, idx := range indices16 {
			indices = append(indices, uint32(idx))
		}

		return indices
	}

	return rawIndices.([]uint32)
}
