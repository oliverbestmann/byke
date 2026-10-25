package byke2d

import (
	"fmt"
	"math"
	"reflect"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var (
	_ = byke.ValidateComponent[ShadowView]()
	_ = byke.ValidateComponent[BinnedRenderPhase[Shadow]]()
)

// Shadow is the render phase of meshes drawn into a shadow map.
type Shadow struct{}

// ShadowView is the view that renders one shadow map layer: the map of a spot
// light, or one cube face of a point light. Its ViewUniforms hold the view
// and projection of the light.
type ShadowView struct {
	byke.Component[ShadowView]

	// Point is true if the layer is in the point light cube maps,
	// false if it is in the spot light maps
	Point bool

	// Layer in the shadow map array texture
	Layer uint32
}

// cubeMapFaces holds the look direction and up vector of each cube face, in the
// layer order of a cube texture (+x, -x, +y, -y, +z, -z). Cube maps are left-handed,
// so the z faces are swapped and the shader flips z when sampling. Same as Bevy.
var cubeMapFaces = [6]struct{ Target, Up glm.Vec3f }{
	{glm.Vec3f{1, 0, 0}, glm.Vec3f{0, 1, 0}},
	{glm.Vec3f{-1, 0, 0}, glm.Vec3f{0, 1, 0}},
	{glm.Vec3f{0, 1, 0}, glm.Vec3f{0, 0, 1}},
	{glm.Vec3f{0, -1, 0}, glm.Vec3f{0, 0, -1}},
	{glm.Vec3f{0, 0, -1}, glm.Vec3f{0, 1, 0}},
	{glm.Vec3f{0, 0, 1}, glm.Vec3f{0, 1, 0}},
}

// shadowMaps holds the depth textures of all shadow casting lights.
type shadowMaps struct {
	// one layer per spot light, sampled as 2d array
	Spot shadowMapTexture

	// six layers per point light, sampled as cube array
	Point shadowMapTexture

	// comparison sampler, shared by both textures
	Sampler *wgpu.Sampler
}

func (m *shadowMaps) texture(point bool) *shadowMapTexture {
	if point {
		return &m.Point
	}

	return &m.Spot
}

// shadowMapTexture is a depth array texture with one render view per layer.
type shadowMapTexture struct {
	Texture *wgpu.Texture
	Size    uint32
	Layers  uint32

	// view of all layers, used for sampling in the main pass
	SampleView *wgpu.TextureView

	// one view per layer, used as depth attachment in the shadow pass
	LayerViews []*wgpu.TextureView
}

func pluginShadows(app *byke.App) {
	app.InsertResource(shadowMaps{})

	app.AddSystems(Render, byke.System(prepareShadowViewsSystem).InSet(RenderPhaseSetupView))
	app.AddSystems(Render, byke.System(queueShadowsSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(renderShadowsSystem).Before(driveCameraSchedules).InSet(RenderPhaseExecute))
	app.AddSystems(Render, byke.System(cleanupBinnedRenderPhaseSystem[Shadow]).InSet(RenderPhaseCleanup))
}

// ensure (re)creates the texture if size or layer count changed.
// The sample view has the given dimension (2DArray or CubeArray).
func (m *shadowMapTexture) ensure(ctx *RenderContext, label string, size, layers uint32, dimension wgpu.TextureViewDimension) {
	if m.Texture != nil && m.Size == size && m.Layers == layers {
		return
	}

	m.release()

	m.Size = size
	m.Layers = layers

	m.Texture = ctx.CreateTexture(&wgpu.TextureDescriptor{
		Label:     label,
		Usage:     wgpu.TextureUsageRenderAttachment | wgpu.TextureUsageTextureBinding,
		Dimension: wgpu.TextureDimension2D,
		Format:    wgpu.TextureFormatDepth32Float,
		Size: wgpu.Extent3D{
			Width:              size,
			Height:             size,
			DepthOrArrayLayers: layers,
		},
		MipLevelCount: 1,
		SampleCount:   1,
	})

	// must be explicit: the default view of a single layer texture is 2D, not 2DArray
	m.SampleView = m.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           label,
		Dimension:       dimension,
		MipLevelCount:   1,
		ArrayLayerCount: layers,
		Aspect:          wgpu.TextureAspectAll,
	})

	for layer := range layers {
		m.LayerViews = append(m.LayerViews, m.Texture.CreateView(&wgpu.TextureViewDescriptor{
			Label:           fmt.Sprintf("%s %d", label, layer),
			Dimension:       wgpu.TextureViewDimension2D,
			MipLevelCount:   1,
			BaseArrayLayer:  layer,
			ArrayLayerCount: 1,
			Aspect:          wgpu.TextureAspectAll,
		}))
	}
}

func (m *shadowMapTexture) release() {
	for _, view := range m.LayerViews {
		view.Release()
	}

	m.LayerViews = m.LayerViews[:0]

	if m.SampleView != nil {
		m.SampleView.Release()
		m.SampleView = nil
	}

	if m.Texture != nil {
		m.Texture.Release()
		m.Texture = nil
	}
}

type shadowViewKey struct {
	Point bool
	Layer uint32
}

// prepareShadowViewsSystem keeps one ShadowView entity per spot light shadow map
// and six per point light shadow map, and updates their view uniforms.
func prepareShadowViewsSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	lights ExtractedLights,
	maps *shadowMaps,
	viewsQuery byke.Query[struct {
		EntityId byke.EntityId
		View     ShadowView
		Uniforms *ViewUniforms
	}],
) {
	size := lights.ShadowMapSize

	// the view uniforms of every shadow view we need this frame
	wanted := map[shadowViewKey]ViewUniforms{}

	var spotCount uint32
	for idx := range lights.SpotLights {
		light := &lights.SpotLights[idx]
		if light.ShadowLayer < 0 {
			continue
		}

		key := shadowViewKey{Layer: uint32(light.ShadowLayer)}
		wanted[key] = shadowViewUniforms(light.WorldToView, light.ViewToClip, light.Position, size)
		spotCount++
	}

	var pointCount uint32
	for idx := range lights.PointLights {
		light := &lights.PointLights[idx]
		if light.ShadowIndex < 0 {
			continue
		}

		viewToClip := glm.Perspective(math.Pi/2, 1, light.ShadowMapNearZ, 0)

		for face, dir := range cubeMapFaces {
			worldToView := glm.LookAt(light.Position, light.Position.Add(dir.Target), dir.Up)
			key := shadowViewKey{Point: true, Layer: uint32(light.ShadowIndex)*6 + uint32(face)}
			wanted[key] = shadowViewUniforms(worldToView, viewToClip, light.Position, size)
		}

		pointCount++
	}

	// keep at least one layer (one cube), because the bind group always needs a texture
	maps.Spot.ensure(ctx, "SpotShadowMaps", size, max(1, spotCount), wgpu.TextureViewDimension2DArray)
	maps.Point.ensure(ctx, "PointShadowMaps", size, 6*max(1, pointCount), wgpu.TextureViewDimensionCubeArray)

	// Linear filtering makes the hardware compare and blend 2x2 texels, which softens edges.
	// Reverse-z (near is depth 1, far is depth 0): a fragment is lit if its depth is >= the stored depth.
	// The sampler is owned by the sampler cache of the RenderContext, do not release it.
	maps.Sampler = ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "ShadowMaps",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeNearest,
		Compare:      wgpu.CompareFunctionGreaterEqual,
	})

	for view := range viewsQuery.Items() {
		key := shadowViewKey{Point: view.View.Point, Layer: view.View.Layer}

		uniforms, ok := wanted[key]
		if !ok {
			// light is gone, or this is a duplicate of a view we already updated
			commands.Entity(view.EntityId).Despawn()
			continue
		}

		*view.Uniforms = uniforms
		delete(wanted, key)
	}

	// spawn the views that do not exist yet
	for key, uniforms := range wanted {
		commands.Spawn(
			ShadowView{Point: key.Point, Layer: key.Layer},
			uniforms,
			BinnedRenderPhase[Shadow]{},
		)
	}
}

func shadowViewUniforms(worldToCamera, cameraToScreen glm.Mat4f, position glm.Vec3f, size uint32) ViewUniforms {
	cameraToScreenInv, _ := cameraToScreen.TryInverse()
	worldToCameraInv, _ := worldToCamera.TryInverse()

	return ViewUniforms{
		ViewportSize:      glm.Vec2f{float32(size), float32(size)},
		CameraToScreen:    cameraToScreen,
		CameraToScreenInv: cameraToScreenInv,
		WorldToCamera:     worldToCamera,
		WorldToCameraInv:  worldToCameraInv,
		WorldToScreen:     cameraToScreen.Mul(worldToCamera),
		WorldToScreenInv:  worldToCameraInv.Mul(cameraToScreenInv),
		CameraPosition:    position,
		Exposure:          1,
	}
}

func queueShadowsSystem(
	meshes *ExtractedMeshes,
	viewsQuery byke.Query[struct {
		_     byke.With[ShadowView]
		Phase *BinnedRenderPhase[Shadow]
	}],
) {
	for view := range viewsQuery.Items() {
		for idx := range meshes.Meshes {
			mesh := &meshes.Meshes[idx]

			// transparent meshes do not cast shadows
			if !mesh.Material.BindGroup().OrderIndependent {
				continue
			}

			view.Phase.Append(meshRenderItem(idx), meshKeyOf(mesh))
		}
	}
}

func renderShadowsSystem(
	world *byke.World,
	ctx *RenderContext,
	maps *shadowMaps,
	viewsQuery byke.Query[struct {
		EntityId byke.EntityId
		View     ShadowView
		Phase    *BinnedRenderPhase[Shadow]
	}],
) {
	defer world.RemoveResource(reflect.TypeFor[CurrentView]())

	// collect first: drawing runs systems, which is not allowed while a query is iterating
	views := viewsQuery.AppendTo(nil)

	for _, view := range views {
		layerViews := maps.texture(view.View.Point).LayerViews
		if view.View.Layer >= uint32(len(layerViews)) {
			// stale view whose despawn command has not been applied yet
			continue
		}

		// drawMeshesBatch looks up the view via the CurrentView
		world.InsertResource(CurrentView(view.EntityId))

		enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Shadow"})

		pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
			Label: "Shadow",
			DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
				View:            layerViews[view.View.Layer],
				DepthLoadOp:     wgpu.LoadOpClear,
				DepthStoreOp:    wgpu.StoreOpStore,
				DepthClearValue: 0,
			},
		})

		view.Phase.Dispatch(world, pass)
		pass.End()
		pass.Release()

		buf := enc.Finish(nil)
		ctx.Submit(buf)
		buf.Release()
		enc.Release()
	}
}
