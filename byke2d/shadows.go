package byke2d

import (
	"fmt"
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

// ShadowView is the view that renders the shadow map of one spot light.
// Its ViewUniforms hold the view and projection of the light.
type ShadowView struct {
	byke.Component[ShadowView]

	// Layer in the shadow map array texture
	Layer uint32
}

// spotShadowMaps holds one depth layer per shadow casting spot light.
type spotShadowMaps struct {
	Texture *wgpu.Texture
	Size    uint32
	Layers  uint32

	// view of all layers, used for sampling in the main pass
	ArrayView *wgpu.TextureView

	// one view per layer, used as depth attachment in the shadow pass
	LayerViews []*wgpu.TextureView

	// comparison sampler
	Sampler *wgpu.Sampler
}

func pluginShadows(app *byke.App) {
	app.InsertResource(spotShadowMaps{})

	app.AddSystems(Render, byke.System(prepareSpotShadowViewsSystem).InSet(RenderPhaseSetupView))
	app.AddSystems(Render, byke.System(queueSpotShadowsSystem).InSet(RenderPhaseQueue))
	app.AddSystems(Render, byke.System(renderSpotShadowsSystem).Before(driveCameraSchedules).InSet(RenderPhaseExecute))
	app.AddSystems(Render, byke.System(cleanupBinnedRenderPhaseSystem[Shadow]).InSet(RenderPhaseCleanup))
}

// ensure (re)creates the texture if size or layer count changed.
func (m *spotShadowMaps) ensure(ctx *RenderContext, size, layers uint32) {
	if m.Texture != nil && m.Size == size && m.Layers == layers {
		return
	}

	m.release()

	m.Size = size
	m.Layers = layers

	m.Texture = ctx.CreateTexture(&wgpu.TextureDescriptor{
		Label:     "SpotShadowMaps",
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
	m.ArrayView = m.Texture.CreateView(&wgpu.TextureViewDescriptor{
		Label:           "SpotShadowMaps",
		Dimension:       wgpu.TextureViewDimension2DArray,
		MipLevelCount:   1,
		ArrayLayerCount: layers,
		Aspect:          wgpu.TextureAspectAll,
	})

	for layer := range layers {
		m.LayerViews = append(m.LayerViews, m.Texture.CreateView(&wgpu.TextureViewDescriptor{
			Label:           fmt.Sprintf("SpotShadowMap %d", layer),
			Dimension:       wgpu.TextureViewDimension2D,
			MipLevelCount:   1,
			BaseArrayLayer:  layer,
			ArrayLayerCount: 1,
			Aspect:          wgpu.TextureAspectAll,
		}))
	}

	// Linear filtering makes the hardware compare and blend 2x2 texels, which softens edges.
	// Reverse-z (near is depth 1, far is depth 0): a fragment is lit if its depth is >= the stored depth.
	m.Sampler = ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "SpotShadowMaps",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeLinear,
		MinFilter:    wgpu.FilterModeLinear,
		MipmapFilter: wgpu.MipmapFilterModeNearest,
		Compare:      wgpu.CompareFunctionGreaterEqual,
	})
}

func (m *spotShadowMaps) release() {
	for _, view := range m.LayerViews {
		view.Release()
	}

	m.LayerViews = m.LayerViews[:0]

	if m.ArrayView != nil {
		m.ArrayView.Release()
		m.ArrayView = nil
	}

	if m.Texture != nil {
		m.Texture.Release()
		m.Texture = nil
	}

	// the sampler is owned by the sampler cache of the RenderContext,
	// we dont need to free it.
}

// prepareSpotShadowViewsSystem keeps one ShadowView entity per shadow
// casting spot light and updates its view uniforms.
func prepareSpotShadowViewsSystem(
	commands *byke.Commands,
	ctx *RenderContext,
	lights ExtractedLights,
	shadowMaps *spotShadowMaps,
	viewsQuery byke.Query[struct {
		EntityId byke.EntityId
		View     ShadowView
		Uniforms *ViewUniforms
	}],
) {
	// shadow casting lights, indexed by their layer
	var shadowLights []*ExtractedSpotLight
	for idx := range lights.SpotLights {
		if light := &lights.SpotLights[idx]; light.ShadowLayer >= 0 {
			shadowLights = append(shadowLights, light)
		}
	}

	// keep at least one layer, because the bind group always needs a texture
	shadowMaps.ensure(ctx, lights.ShadowMapSize, max(1, uint32(len(shadowLights))))

	hasView := make([]bool, len(shadowLights))

	for view := range viewsQuery.Items() {
		layer := view.View.Layer
		if layer >= uint32(len(shadowLights)) || hasView[layer] {
			commands.Entity(view.EntityId).Despawn()
			continue
		}

		hasView[layer] = true
		*view.Uniforms = shadowViewUniforms(shadowLights[layer], lights.ShadowMapSize)
	}

	for layer, light := range shadowLights {
		if hasView[layer] {
			continue
		}

		commands.Spawn(
			ShadowView{Layer: uint32(layer)},
			shadowViewUniforms(light, lights.ShadowMapSize),
			BinnedRenderPhase[Shadow]{},
		)
	}
}

func shadowViewUniforms(light *ExtractedSpotLight, size uint32) ViewUniforms {
	cameraToScreen := light.ViewToClip
	cameraToScreenInv, _ := cameraToScreen.TryInverse()

	worldToCamera := light.WorldToView
	worldToCameraInv, _ := worldToCamera.TryInverse()

	return ViewUniforms{
		ViewportSize:      glm.Vec2f{float32(size), float32(size)},
		CameraToScreen:    cameraToScreen,
		CameraToScreenInv: cameraToScreenInv,
		WorldToCamera:     worldToCamera,
		WorldToCameraInv:  worldToCameraInv,
		WorldToScreen:     cameraToScreen.Mul(worldToCamera),
		WorldToScreenInv:  worldToCameraInv.Mul(cameraToScreenInv),
		CameraPosition:    light.Position,
		Exposure:          1,
	}
}

func queueSpotShadowsSystem(
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

func renderSpotShadowsSystem(
	world *byke.World,
	ctx *RenderContext,
	shadowMaps *spotShadowMaps,
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
		if view.View.Layer >= uint32(len(shadowMaps.LayerViews)) {
			// stale view whose despawn command has not been applied yet
			continue
		}

		// drawMeshesBatch looks up the view via the CurrentView
		world.InsertResource(CurrentView(view.EntityId))

		enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "SpotShadow"})

		pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
			Label: "SpotShadow",
			DepthStencilAttachment: &wgpu.RenderPassDepthStencilAttachment{
				View:            shadowMaps.LayerViews[view.View.Layer],
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
