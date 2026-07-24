package byke2d

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
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

	// PreferAlphaToCoverage indicates that we prefer to use alpha to coverage in the material
	// settings instead of alpha mode blend
	PreferAlphaToCoverage bool
}

func (s SceneRoot) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
		InheritVisibility,
	}
}

func pluginGltf(app *byke.App) {
	app.AddSystems(byke.PostUpdate, spawnGltfSceneSystem)
}

func spawnGltfSceneSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	assets *Assets,
	scenesQuery byke.Query[struct {
		_         byke.Changed[SceneRoot]
		EntityId  byke.EntityId
		SceneRoot SceneRoot
	}],
) {
	for item := range scenesQuery.Items() {
		handle := toGltfHandle(item.SceneRoot.Handle)

		sc := &spawnContext{
			Commands:              commands,
			Handle:                handle,
			RenderContext:         ctx,
			Assets:                assets,
			nodes:                 map[gltf.Ref]byke.EntityId{},
			nodeWorldTransform:    map[gltf.Ref]glm.Mat4f{},
			images:                map[imageCacheKey]*Texture{},
			textures:              map[textureCacheKey]*Texture{},
			nodeToMesh:            map[gltf.Ref][]byke.EntityId{},
			meshes:                map[meshKey]*Mesh{},
			preferAlphaToCoverage: item.SceneRoot.PreferAlphaToCoverage,
		}

		scene := sc.Handle.Scenes[item.SceneRoot.Scene]
		slog.Info("Spawning gtlf scene", slog.String("name", scene.Name))

		sc.SpawnScene(item.EntityId, item.SceneRoot.Scene)
	}
}

type meshKey struct {
	MeshId gltf.Ref
	SubId  uint32
}

type imageCacheKey struct {
	Ref    gltf.Ref
	Linear bool
}

type textureCacheKey struct {
	ImageId   gltf.Ref
	SamplerId gltf.Ref
	Linear    bool
}

type spawnContext struct {
	Commands      *byke.Commands
	Handle        gltfHandle
	RenderContext *RenderContext
	Assets        *Assets

	// the root entity
	root byke.EntityId

	// map from reachable nodes to entities
	nodes map[gltf.Ref]byke.EntityId

	// map from reachable node to its world transform.
	// we need this to calculate the mesh materials winding order
	nodeWorldTransform map[gltf.Ref]glm.Mat4f

	// map from imageId to loaded texture data
	images map[imageCacheKey]*Texture

	// map from textureId to loaded texture data & sampler
	textures map[textureCacheKey]*Texture

	// map from node to mesh entities
	nodeToMesh map[gltf.Ref][]byke.EntityId

	// primitive meshes that can be used for instantiation if
	// referenced multiple times
	meshes map[meshKey]*Mesh

	// prefer alpha to coverage over blend
	preferAlphaToCoverage bool
}

func (sc *spawnContext) SpawnScene(parentId byke.EntityId, sceneId gltf.Ref) {
	// spawn root entity
	sc.root = sc.Commands.Spawn(
		SceneInstance{},
		NewTransform().WithScaleXYZ(1, 1, -1),
		byke.ChildOf{Parent: parentId}).Id()

	// first step, spawn nodes
	for _, node := range sc.Handle.Scene(sceneId) {
		worldTransform := glm.IdentityMat4f()
		sc.spawnNodeTree(sc.root, node, worldTransform)
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
		light := node.Extensions.MustParse[gltf.KHRLightsPunctualInNode]()
		if light != nil {
			sc.spawnLightInNode(node, light)
		}
	}

	// for all animations, create animation targets and link to the root node
	for _, animation := range sc.Handle.Animations {
		sc.spawnAnimationTargets(sc.root, animation)
	}

	var mergedAnimation AnimationClip
	for _, anim := range sc.Handle.Animations {
		slog.Debug("Animation", slog.String("name", anim.Name))
		mergedAnimation = mergedAnimation.MergeWith(sc.buildAnimation(anim))
	}

	if !mergedAnimation.IsEmpty() {
		// spawn the first animation on the root entity
		sc.Commands.Entity(sc.root).Insert(
			ActiveAnimation{
				Animation: mergedAnimation,
			},
		)
	}

	// 	if len(sc.Handle.Animations) > 0 {
	// 		// spawn the first animation on the root entity
	// 		sc.Commands.Entity(sc.root).Insert(
	// 			ActiveAnimation{
	// 				Animation: sc.buildAnimation(sc.Handle.Animations[0]),
	// 			},
	// 		)
	// 	}

	slog.Debug(
		"Spawn scene summary",
		slog.Int("textures", len(sc.images)),
		slog.Int("meshes", len(sc.meshes)),
	)
}

func (sc *spawnContext) spawnNodeTree(parentId byke.EntityId, node gltf.Node, parentTransform glm.Mat4f) {
	transform := gltfConvertTransform(node)

	// track the world transform
	worldTransform := parentTransform.Mul(transform.Affine3())

	// spawn a new entity for the node
	entityId := sc.Commands.
		Spawn(byke.ChildOf{Parent: parentId}, InheritVisibility, transform).
		Id()

	// record it in the lookup table
	sc.nodes[node.Id] = entityId
	sc.nodeWorldTransform[node.Id] = worldTransform

	// spawn child nodes
	for _, node := range sc.Handle.ChildNodes(node) {
		sc.spawnNodeTree(entityId, node, worldTransform)
	}
}

func (sc *spawnContext) spawnMeshInNode(node gltf.Node) {
	// if node was not spawned, skip it
	entityId, ok := sc.nodes[node.Id]
	if !ok {
		return
	}

	// get the mesh
	meshId := node.Mesh.Get()
	mesh := sc.Handle.Meshes[meshId]

	for idx, prim := range mesh.Primitives {
		var material StandardMaterial

		if ma := prim.Material; ma.IsSet {
			material = sc.materialAt(ma.Get())
		}

		material.FrontFace = wgpu.FrontFaceCW

		worldTransform := sc.nodeWorldTransform[node.Id]
		flipWinding := worldTransform[0][0]*worldTransform[1][1]*worldTransform[2][2] < 0
		if flipWinding {
			material.FrontFace = wgpu.FrontFaceCCW
		}

		meshInst := sc.instantiateMesh(meshId, idx)

		entityCommands := sc.Commands.Spawn(
			byke.ChildOf{Parent: entityId},
			MeshInstance{Mesh: meshInst},
			material,
		)

		if name := mesh.Name; name != "" {
			entityCommands.Insert(byke.Named(name))
		}

		if len(mesh.Weights) > 0 {
			// we got mesh target weights
			entityCommands.Insert(meshMorphWeights{})
		}

		sc.nodeToMesh[node.Id] = append(sc.nodeToMesh[node.Id], entityCommands.Id())
	}

	if len(mesh.Weights) > 0 {
		// insert MeshWeights into the mesh itself.
		sc.Commands.Entity(entityId).Insert(
			MorphWeights{
				Weights: mesh.Weights,
				Names:   mesh.Extras.TargetNames,
			},
		)
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
				Color:        ColorLinearRGB(glm.Vec3f(light.Color).Scale(light.Intensity).XYZ()),
				AttConstant:  0,
				AttLinear:    0,
				AttQuadratic: 1,
			},
		)
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

	for _, entityId := range sc.nodeToMesh[node.Id] {
		sc.Commands.Entity(entityId).Insert(skinned)
	}
}

func (sc *spawnContext) materialAt(matId gltf.Ref) StandardMaterial {
	var m StandardMaterial

	mat := sc.Handle.Materials[matId]

	// parse base color
	m.Tint = ColorOf(mat.BaseColor())

	// enable double sided rendering
	m.DoubleSided = mat.DoubleSided

	switch mat.AlphaMode {
	case "OPAQUE":
		m.AlphaMode = AlphaModeOpaque

	case "BLEND":
		m.AlphaMode = AlphaModeBlend
		if sc.preferAlphaToCoverage {
			m.AlphaMode = AlphaModeAlphaToCoverage
		}

	case "MASK":
		m.AlphaMode = AlphaModeMask
		m.AlphaCutoff = derefOr(mat.AlphaCutoff, 0.5)
	}

	if mr := mat.MetallicRoughness; mr != nil {
		if baseColorTex := mr.BaseColorTexture; baseColorTex != nil {
			// parse texture as srgb
			m.Texture = sc.textureAt(baseColorTex.Index, false)
		}
	}

	if no := mat.NormalTexture; no != nil {
		// TODO handle no.Scale & no.TexCoords
		m.NormalTexture = sc.textureAt(no.Index, true)
	}

	if oc := mat.OcclusionTexture; oc != nil {
		m.OcclusionTexture = sc.textureAt(oc.Index, false)
	}

	if em := mat.EmissiveTexture; em != nil {
		m.EmissiveTexture = sc.textureAt(em.Index, false)
	}

	m.EmissiveScale = mat.EmissiveFactor

	emissiveStrength := mat.Extensions.MustParse[gltf.KHRMaterialsEmissiveStrengthInMaterial]()
	if emissiveStrength != nil {
		m.EmissiveScale = m.EmissiveScale.Scale(emissiveStrength.EmissiveStrength)
	}

	return m
}

func (sc *spawnContext) textureAt(texId gltf.Ref, linearColors bool) *Texture {
	tex := sc.Handle.Textures[texId]

	key := textureCacheKey{
		ImageId:   tex.Source,
		SamplerId: tex.Sampler,
		Linear:    linearColors,
	}

	if cachedTexture, ok := sc.textures[key]; ok {
		return cachedTexture
	}

	// get the image for this texture and create a shallow copy of the texture
	texture := new(*sc.imageOf(tex.Source, linearColors))

	sampler := sc.Handle.Samplers[tex.Sampler]

	// TODO translate sampler
	texture.Sampler = sc.RenderContext.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "gltf: " + sampler.Name,
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeLinear,
	})

	sc.textures[key] = texture

	return texture
}

func (sc *spawnContext) imageOf(imageId gltf.Ref, linearColors bool) *Texture {
	key := imageCacheKey{Ref: imageId, Linear: linearColors}

	if cached, ok := sc.images[key]; ok {
		return cached
	}

	img := sc.Handle.Images[imageId]

	var texture *Texture
	if img.Uri != "" {
		// load from URL
		settings := &LoadTextureSettings{Linear: linearColors}
		texture = sc.Assets.TextureWithSettings(img.Uri, settings).Await()
	} else {

		// get the buffer
		buffer := sc.Handle.Buffer(img.BufferView)

		slog.Debug(
			"Load texture from memory",
			slog.String("name", img.Name),
			slog.Any("imageId", imageId),
			slog.Int("size", len(buffer)),
			slog.Bool("linear", linearColors),
		)

		var err error

		// load the texture data into memory
		src, _, err := image.Decode(bytes.NewReader(buffer))
		if err != nil {
			panic(fmt.Errorf("decode image from memory: %w", err))
		}

		texture = NewTextureFromImage(sc.RenderContext, src, TextureFromImageOptions{
			Label:  "gltf:" + img.Name,
			Linear: linearColors,
		})
	}

	wgpu.Share(texture.Texture)
	wgpu.Share(texture.TextureView)
	wgpu.Share(texture.Sampler)

	sc.images[key] = texture

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

func (sc *spawnContext) instantiateMesh(meshId gltf.Ref, subId int) *Mesh {
	key := meshKey{MeshId: meshId, SubId: uint32(subId)}

	// lookup in cache first
	cached, ok := sc.meshes[key]
	if ok {
		return cached
	}

	primitive := sc.Handle.Meshes[meshId].Primitives[subId]
	mesh := gltfConvertPrimitiveMesh(&sc.Handle, primitive)

	// put into cache
	sc.meshes[key] = mesh

	return mesh
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
	vertices := h.Resolve(prim.MustGet("POSITION")).([]glm.Vec3f)

	var indices []uint32

	if prim.Indices.IsSet {
		// get and convert indices if necessary
		rawIndices := h.Resolve(prim.Indices.Value)
		indices = gltfConvertMeshIndices(rawIndices)
	} else {
		for idx := range vertices {
			indices = append(indices, uint32(idx))
		}
	}

	mesh := MeshOf(indices, vertices)

	for key, value := range prim.Attributes {
		switch key {
		case "POSITION":
			// handled above

		case "TEXCOORD_0":
			values := h.Resolve(value).([]glm.Vec2f)
			mesh.WithAttributes(VertexAttributeUV, values)

		case "TEXCOORD_1":
			values := h.Resolve(value).([]glm.Vec2f)
			mesh.WithAttributes(VertexAttributeUV1, values)

		case "TEXCOORD_2":
			values := h.Resolve(value).([]glm.Vec2f)
			mesh.WithAttributes(VertexAttributeUV2, values)

		case "NORMAL":
			values := h.Resolve(value).([]glm.Vec3f)
			mesh.WithAttributes(VertexAttributeNormal, values)

		case "TANGENT":
			values := h.Resolve(value).([]glm.Vec4f)
			mesh.WithAttributes(VertexAttributeTangentSpace, values)

		case "JOINTS_0":
			values := h.Resolve(value).([]glm.Vec4uh)
			mesh.WithAttributes(VertexAttributeJoints, values)

		case "WEIGHTS_0":
			values := h.Resolve(value).([]glm.Vec4f)
			mesh.WithAttributes(VertexAttributeJointWeights, values)

		default:
			slog.Warn("Cannot map vertex attributes from gltf", slog.String("name", key))
		}
	}

	if !mesh.HasAttribute(VertexAttributeNormal) {
		mesh.ComputeNormals()
	}

	if !mesh.HasAttribute(VertexAttributeTangentSpace) {
		mesh.ComputeTangents()
	}

	for _, target := range prim.Targets {
		morphAttributes := convertMorphAttributes(h, len(vertices), target)
		mesh.WithMorphTarget(morphAttributes)
	}

	return mesh
}

func convertMorphAttributes(h *gltfHandle, vertexCount int, target gltf.MorphTarget) []MorphAttributes {
	attributes := make([]MorphAttributes, vertexCount)

	if target.Positions.IsSet {
		offsets := h.Resolve(target.Positions.Value).([]glm.Vec3f)
		for i := range vertexCount {
			attributes[i].Position = offsets[i]
		}
	}

	if target.Normals.IsSet {
		offsets := h.Resolve(target.Normals.Value).([]glm.Vec3f)
		for i := range vertexCount {
			attributes[i].Normal = offsets[i]
		}
	}

	if target.Tangents.IsSet {
		offsets := h.Resolve(target.Tangents.Value).([]glm.Vec3f)
		for i := range vertexCount {
			attributes[i].Tangent = offsets[i]
		}
	}

	return attributes
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
