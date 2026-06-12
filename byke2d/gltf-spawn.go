package byke2d

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/gltf"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type SceneInstance struct {
	byke.ImmutableComponent[SceneInstance]
}

func (s SceneInstance) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
		InheritVisibility,
	}
}

type SceneRoot struct {
	byke.ComparableComponent[SceneRoot]
	Handle *gltf.Handle
	Scene  gltf.Ref
}

func pluginGltf(app *byke.App) {
	app.AddSystems(byke.PostUpdate, spawnGltfSceneSystem)
}

func spawnGltfSceneSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	scenesQuery byke.Query[struct {
		_         byke.Changed[SceneRoot]
		SceneRoot SceneRoot
	}],
) {
	for item := range scenesQuery.Items() {
		handle := toGltfHandle(item.SceneRoot.Handle)

		sc := &spawnContext{
			Commands:      commands,
			Handle:        handle,
			RenderContext: ctx,
			nodes:         map[gltf.Ref]byke.EntityId{},
			images:        map[gltf.Ref]*Texture{},
			meshes:        map[gltf.Ref][]byke.EntityId{},
		}

		scene := sc.Handle.Scenes[item.SceneRoot.Scene]
		slog.Info("Spawning gtlf scene", slog.String("name", scene.Name))

		sc.SpawnScene(item.SceneRoot.Scene)
	}
}

type spawnContext struct {
	Commands      *byke.Commands
	Handle        gltfHandle
	RenderContext *RenderContext

	// the root entity
	root byke.EntityId

	// map from reachable nodes to entitys
	nodes map[gltf.Ref]byke.EntityId

	// map from imageId to texture
	images map[gltf.Ref]*Texture

	// map from node to mesh entities
	meshes map[gltf.Ref][]byke.EntityId
}

func (sc *spawnContext) SpawnScene(sceneId gltf.Ref) {
	// spawn root entity
	sc.root = sc.Commands.Spawn(SceneInstance{}).Id()

	// first step, spawn nodes
	for _, node := range sc.Handle.Scene(sceneId) {
		sc.spawnNodeTree(sc.root, node)
	}

	// now walk through all nodes again and spawn objects
	for _, node := range sc.Handle.Nodes {
		if node.Mesh.IsSet {
			// spawn mesh on node
			sc.spawnMeshInNode(node)
		}

		// if the mesh has a skin, spawn that one too
		if node.Skin.IsSet {
			sc.spawnSkinInNode(node)
		}

		// spawn light on node
		light := gltfExtensionOf[gltf.KHRLightsPunctualInNode](node, "KHR_lights_punctual")
		if light != nil {
			sc.spawnLightInNode(node, light)
		}
	}

	// for all animations, create animation targets and link to the root node
	for _, animation := range sc.Handle.Animations {
		sc.spawnAnimationTargets(sc.root, animation)
	}

	// spawn the first animation on the root entity
	sc.Commands.Entity(sc.root).Insert(
		ActiveAnimation{
			Animation: sc.buildAnimation(sc.Handle.Animations[0]),
		},
	)
}

func (sc *spawnContext) spawnNodeTree(parentId byke.EntityId, node gltf.Node) {
	transform := gltfConvertTransform(node)

	// spawn a new entity for the node
	entityId := sc.Commands.
		Spawn(byke.ChildOf{Parent: parentId}, InheritVisibility, transform).
		Id()

	// record it in the lookup table
	sc.nodes[node.Id] = entityId

	// spawn child nodes
	for _, node := range sc.Handle.ChildNodes(node) {
		sc.spawnNodeTree(entityId, node)
	}
}

func (sc *spawnContext) spawnMeshInNode(node gltf.Node) {
	// if node was not spawned, skip it
	entityId, ok := sc.nodes[node.Id]
	if !ok {
		return
	}

	// get the mesh
	mesh := sc.Handle.Meshes[node.Mesh.Get()]

	for _, prim := range mesh.Primitives {
		var material StandardMaterial

		if ma := prim.Material; ma.IsSet {
			material = sc.materialAt(ma.Get())
		}

		// TODO meshes can be re-used, so we can probably just
		//  instantiate all meshes first and then re-use
		mesh3d := gltfConvertPrimitiveMesh(&sc.Handle, prim)

		entityCommands := sc.Commands.Spawn(
			byke.ChildOf{Parent: entityId},
			Mesh3d{Mesh: mesh3d},
			material,
		)

		if name := mesh.Name; name != "" {
			entityCommands.Insert(byke.Named(name))
		}

		sc.meshes[node.Id] = append(sc.meshes[node.Id], entityCommands.Id())
	}
}

func (sc *spawnContext) spawnLightInNode(node gltf.Node, ext *gltf.KHRLightsPunctualInNode) {
	// if node was not spawned, skip it
	entityId, ok := sc.nodes[node.Id]
	if !ok {
		return
	}

	light := &sc.Handle.Lights.Lights[ext.Light]

	if light.Type == "point" {
		sc.Commands.Spawn(
			byke.ChildOf{Parent: entityId},
			PointLight{
				Color:        light.Color,
				Intensity:    light.Intensity,
				AttConstant:  0,
				AttLinear:    0,
				AttQuadratic: 1,
			})
	}
}

func (sc *spawnContext) spawnSkinInNode(node gltf.Node) {
	// if node was not spawned, skip it
	if _, ok := sc.nodes[node.Id]; !ok {
		return
	}

	// get the skin we want to translate
	skin := sc.Handle.Skins[node.Skin.Get()]

	var skinned SkinnedMesh

	for _, joint := range skin.Joints {
		jointId, ok := sc.nodes[joint]
		if !ok {
			panic(fmt.Errorf("joint node not spawned: %d", joint))
		}

		skinned.Joints = append(skinned.Joints, jointId)
		skinned.InverseBind = append(skinned.InverseBind, glm.IdentityMat4f())
	}

	if skin.InverseBindMatrices.IsSet {
		matrices := sc.Handle.Resolve(skin.InverseBindMatrices.Get()).([]glm.Mat4f)
		for idx := range skinned.InverseBind {
			skinned.InverseBind[idx] = matrices[idx]
		}
	}

	for _, entityId := range sc.meshes[node.Id] {
		sc.Commands.Entity(entityId).Insert(skinned)
	}
}

func (sc *spawnContext) materialAt(matId gltf.Ref) StandardMaterial {
	var m StandardMaterial

	mat := sc.Handle.Materials[matId]

	// parse base color
	m.Tint = ColorOf(mat.BaseColor())

	if mr := mat.MetallicRoughness; mr != nil {
		if baseColorTex := mr.BaseColorTexture; baseColorTex != nil {
			// parse texture
			m.Texture = sc.textureAt(baseColorTex.Index)
		}
	}

	return m
}

func (sc *spawnContext) textureAt(texId gltf.Ref) *Texture {
	tex := sc.Handle.Textures[texId]

	// get the image for this texture and create a shallow copy of the texture
	texture := *sc.imageOf(tex.Source)

	sampler := sc.Handle.Samplers[tex.Sampler]

	// TODO set sampler according to sampler config
	texture.Sampler = sc.RenderContext.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "gltf: " + sampler.Name,
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeLinear,
	})

	// TODO cache together with sample?
	return new(texture)
}

func (sc *spawnContext) imageOf(imageId gltf.Ref) *Texture {
	if cached, ok := sc.images[imageId]; ok {
		return cached
	}

	image := sc.Handle.Images[imageId]

	// get the buffer
	buffer := sc.Handle.Buffer(image.BufferView)

	slog.Debug("Load texture from memory",
		slog.String("name", image.Name),
		slog.Any("imageId", imageId),
		slog.Int("size", len(buffer)),
	)

	texture, err := DecodeTextureFromMemory(sc.RenderContext, buffer, SamplerConfig{}, true)
	if err != nil {
		panic(err)
	}

	sc.images[imageId] = texture

	return texture
}

func (sc *spawnContext) spawnAnimationTargets(animator byke.EntityId, animation gltf.Animation) {
	for _, ch := range animation.Channels {
		entityId, ok := sc.nodes[ch.Target.Node]
		if !ok {
			continue
		}

		targetId := sc.animationTargetOf(ch)

		// update entity accordingly
		sc.Commands.Entity(entityId).Insert(
			AnimatedBy{Animator: animator},
			targetId,
		)
	}
}

func (sc *spawnContext) animationTargetOf(ch gltf.AnimationChannel) AnimationTargetId {
	// TODO maybe full path?
	return AnimationTargetIdOf(fmt.Sprintf("%d", ch.Target.Node))
}

func (sc *spawnContext) buildAnimation(anim gltf.Animation) AnimationClip {
	var clip AnimationClip

	for _, ch := range anim.Channels {
		curve := sc.animationCurveOf(anim, ch)
		if curve == nil {
			continue
		}

		targetId := sc.animationTargetOf(ch)
		clip.Add(targetId, curve)
	}

	return clip
}

func gltfConvertTransform(node gltf.Node) Transform {
	tr, scale, rot := node.TransformComponents()

	return Transform{
		Translation: tr,
		Scale:       scale,
		Rotation:    rot.Inverse(),
	}
}

func gltfConvertPrimitiveMesh(h *gltfHandle, prim gltf.MeshPrimitive) *Mesh {
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
		case "POSITION":
			// handled above

		case "TEXCOORD_0":
			uv := h.Resolve(value).([]glm.Vec2f)
			mesh.WithAttributes(VertexAttributeUV, wgpu.ToBytes(uv))

		case "NORMAL":
			uv := h.Resolve(value).([]glm.Vec3f)
			mesh.WithAttributes(VertexAttributeNormal, wgpu.ToBytes(uv))

		case "JOINTS_0":
			uv := h.Resolve(value).([]glm.Vec4uh)
			mesh.WithAttributes(VertexAttributeJoints, wgpu.ToBytes(uv))

		case "WEIGHTS_0":
			uv := h.Resolve(value).([]glm.Vec4f)
			mesh.WithAttributes(VertexAttributeJointWeights, wgpu.ToBytes(uv))

		default:
			slog.Warn("Cannot map vertex attributes from gltf", slog.String("name", key))
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

func toGltfHandle(h *gltf.Handle) gltfHandle {
	var e gltfExtensions

	if len(h.Extensions) > 0 {
		if err := json.Unmarshal(h.Extensions, &e); err != nil {
			panic(fmt.Errorf("deserialize extensions %T: %w", e, err))
		}
	}

	return gltfHandle{h, e}
}

func gltfExtensionOf[T any](node gltf.Node, name string) *T {
	ext, err := gltf.ExtensionOf[T](node.Extensions, name)
	if err != nil {
		panic(fmt.Errorf("read light extension: %w", err))
	}

	return ext
}
