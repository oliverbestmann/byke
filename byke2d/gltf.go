package byke2d

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/gltf"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type Scene struct {
	Handle *gltf.Handle
	Index  int
}

func SceneRoot(world *byke.World, handle *gltf.Handle, sceneId gltf.Ref) byke.ErasedComponent {
	h := toGltfHandle(handle)

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

func gltfConvert(ctx *RenderContext, h *gltfHandle, node gltf.Node) []byke.ErasedComponent {
	if m := node.Mesh; !m.IsNil() {
		return gltfConvertMeshNode(ctx, h, node, m)
	}

	l, err := gltf.ExtensionOf[gltf.KHRLightsPunctualInNode](node.Extensions, "KHR_lights_punctual")
	if err != nil {
		panic(err)
	}

	if l != nil {
		transform := gltfConvertTransform(node)

		light := &h.Lights.Lights[l.Light]

		// return a node for the light
		var bundle []byke.ErasedComponent
		bundle = append(bundle, InheritVisibility)
		bundle = append(bundle, transform)

		if light.Type == "point" {
			bundle = append(bundle, PointLight{
				Color:        light.Color,
				Intensity:    light.Intensity,
				AttConstant:  0,
				AttLinear:    0,
				AttQuadratic: 1,
			})
		}

		return bundle
	}

	return nil
}

func gltfConvertMeshNode(ctx *RenderContext, h *gltfHandle, node gltf.Node, m gltf.OptionRef) []byke.ErasedComponent {
	transform := gltfConvertTransform(node)

	// we'll return an intermediary node
	var bundle []byke.ErasedComponent
	bundle = append(bundle, InheritVisibility)
	bundle = append(bundle, transform)

	// spawn each mesh as a direct child
	for _, prim := range h.Meshes[m.Get()].Primitives {
		var material StandardMaterial

		if ma := prim.Material; ma.IsSet {
			material = gltfConvertMaterial(ctx, h, prim.Material.Get())
		}

		mesh := gltfConvertMesh(h, prim)
		bundle = append(bundle, byke.SpawnChild(
			Mesh3d{Mesh: mesh},
			material,
		))
	}

	// recurse into child nodes and spawn them as children
	for _, child := range h.ChildNodes(node) {
		components := gltfConvert(ctx, h, child)
		if len(components) == 0 {
			continue
		}

		bundleChild := byke.SpawnChild(components...)
		bundle = append(bundle, bundleChild)
	}

	return bundle
}

func gltfConvertMaterial(ctx *RenderContext, h *gltfHandle, ma gltf.Ref) StandardMaterial {
	var m StandardMaterial

	mat := h.Materials[ma]
	m.Tint = ColorOf(mat.BaseColor())

	if mr := mat.MetallicRoughness; mr != nil {
		if tex := mr.BaseColorTexture; tex != nil {
			bufView := h.Images[h.Textures[tex.Index].Source].BufferView

			texture, err := DecodeTextureFromMemory(ctx, h.Buffer(bufView), SamplerConfig{}, true)
			if err != nil {
				panic(err)
			}

			m.Texture = texture
		}
	}

	return m
}

func gltfConvertTransform(node gltf.Node) Transform {
	tr, scale, rot := node.TransformComponents()
	return Transform{
		Translation: tr,
		Scale:       scale,
		Rotation:    rot,
	}
}

func gltfConvertMesh(h *gltfHandle, prim gltf.MeshPrimitive) *Mesh {
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

type gltfHandle struct {
	*gltf.Handle
	gltfExtensions
}

type gltfExtensions struct {
	Lights gltf.KHRLightsPunctualInFile `json:"KHR_lights_punctual"`
}

func toGltfHandle(h *gltf.Handle) *gltfHandle {
	var e gltfExtensions

	if len(h.Extensions) > 0 {
		if err := json.Unmarshal(h.Extensions, &e); err != nil {
			panic(fmt.Errorf("deserialize extensions %T: %w", e, err))
		}
	}

	return new(gltfHandle{h, e})
}
