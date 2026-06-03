package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[PointLight]()

type PointLight struct {
	byke.Component[PointLight]
	Color        glm.Vec3f
	Intensity    float32
	AttConstant  float32
	AttLinear    float32
	AttQuadratic float32
}

func (PointLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

func pluginLights(app *byke.App) {
	app.InsertResource(DefaultLightConfig)
	app.InsertResource(ExtractedLights{})
	app.InsertResource(lightsStorage{})
	app.InsertResource(LightsBindGroup{})
	app.AddSystems(Render, byke.System(extractLights).InSet(RenderPhaseExtract))
	app.AddSystems(Render, byke.System(prepareLightsStorage).InSet(RenderPhasePrepareResources))
	app.AddSystems(Render, byke.System(prepareLightsBindGroup).InSet(RenderPhasePrepareBindGroups))
}

type ExtractedLights struct {
	PointLights []ExtractedPointLight
}

func (l *ExtractedLights) Clear() {
	l.PointLights = l.PointLights[:0]
}

type ExtractedPointLight struct {
	Position glm.Vec3f
	PointLight
}

func (l *ExtractedPointLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Position)
	w.AppendFloat32(l.Intensity)
	w.AppendFloat32(l.AttConstant)
	w.AppendFloat32(l.AttLinear)
	w.AppendFloat32(l.AttQuadratic)
}

func extractLights(
	lights *ExtractedLights,

	lightsQuery byke.Query[struct {
		Light     PointLight
		Transform GlobalTransform
	}],
) {
	lights.Clear()

	for light := range lightsQuery.Items() {
		if light.Light.Intensity <= 0 || light.Light.Color == (glm.Vec3f{}) {
			// off, no light
			continue
		}

		lights.PointLights = append(lights.PointLights, ExtractedPointLight{
			Position:   light.Transform.Affine.Translation(),
			PointLight: light.Light,
		})
	}
}

type lightsStorage struct {
	Writer    wgsl.StructWriter
	Buffer    *wgpu.Buffer
	BindGroup *wgpu.BindGroup
}

type LightsBindGroup struct {
	BindGroup *wgpu.BindGroup
}

type LightConfig struct {
	Ambient Color
}

var DefaultLightConfig = LightConfig{
	Ambient: ColorLinearRGBA(0.1, 0.1, 0.1, 1.0),
}

func prepareLightsStorage(
	ctx *RenderContext,
	uniforms *lightsStorage,
	lights ExtractedLights,
	lightConfig LightConfig,
) {
	uniforms.Writer.Clear()

	uniforms.Writer.AppendVec3f(lightConfig.Ambient.ToVec().Truncate())
	uniforms.Writer.AppendUint(uint32(len(lights.PointLights)))

	// TODO start nested struct and keep alignment correct
	for _, light := range lights.PointLights {
		light.WriteTo(&uniforms.Writer)
	}

	uniforms.Writer.WriteTo(ctx, &uniforms.Buffer, wgpu.BufferUsageStorage)
}

var LightsBindGroupLayout = SequentialLayout(
	BindingLayoutBuffer(wgpu.BufferBindingTypeReadOnlyStorage, false),
)

func prepareLightsBindGroup(
	ctx *RenderContext,
	pipelines *PipelineCache,
	bindGroup *LightsBindGroup,
	lights *lightsStorage,
) {
	bindGroup.BindGroup.Release()
	bindGroup.BindGroup = ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   "Lights",
		Layout:  pipelines.BindGroupLayout(LightsBindGroupLayout),
		Entries: Sequential(BindingBuffer(lights.Buffer)),
	})
}
