package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[DirectionalLight]()
var _ = byke.ValidateComponent[PointLight]()
var _ = byke.ValidateComponent[SpotLight]()

type LightConfig struct {
	Ambient Color
}

var DefaultLightConfig = LightConfig{
	Ambient: ColorSRGB(0.1, 0.1, 0.1),
}

type DirectionalLight struct {
	byke.Component[DirectionalLight]
	Color Color
}

func (DirectionalLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

type PointLight struct {
	byke.Component[PointLight]
	Color        Color
	AttConstant  float32
	AttLinear    float32
	AttQuadratic float32
}

func (PointLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

type SpotLight struct {
	byke.Component[SpotLight]
	Color        Color
	InnerAngle   glm.Rad
	OuterAngle   glm.Rad
	AttConstant  float32
	AttLinear    float32
	AttQuadratic float32
}

func (SpotLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

func pluginLights(app *byke.App) {
	app.InsertResource(DefaultLightConfig)
	app.InsertResource(ExtractedLights{})
	app.InsertResource(lightsStorage{})
	app.InsertResource(meshViewBindGroup{})
	app.AddSystems(Render, byke.System(extractLights).InSet(RenderPhaseExtract))
	app.AddSystems(Render, byke.System(prepareLightsStorage).InSet(RenderPhasePrepareResources))
}

type ExtractedLights struct {
	// Ambient light color
	Ambient glm.Vec3f

	DirectionalLights []ExtractedDirectionalLight
	PointLights       []ExtractedPointLight
	SpotLights        []ExtractedSpotLight
}

func (l *ExtractedLights) Clear() {
	l.DirectionalLights = l.DirectionalLights[:0]
	l.PointLights = l.PointLights[:0]
	l.SpotLights = l.SpotLights[:0]
}

type ExtractedPointLight struct {
	Position     glm.Vec3f
	Color        glm.Vec3f
	AttConstant  float32
	AttLinear    float32
	AttQuadratic float32
}

func (l ExtractedPointLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color.ToVec3f())
	w.AppendVec3f(l.Position)
	w.AppendFloat32(l.AttConstant)
	w.AppendFloat32(l.AttLinear)
	w.AppendFloat32(l.AttQuadratic)
	w.Sync()
}

type ExtractedSpotLight struct {
	Color        glm.Vec3f
	Position     glm.Vec3f
	Direction    glm.Vec3f
	InnerAngle   glm.Rad
	OuterAngle   glm.Rad
	AttConstant  float32
	AttLinear    float32
	AttQuadratic float32
}

func (l ExtractedSpotLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Position)
	w.AppendVec3f(l.Direction)
	w.AppendFloat32(float32(l.InnerAngle))
	w.AppendFloat32(float32(l.OuterAngle))
	w.AppendFloat32(l.AttConstant)
	w.AppendFloat32(l.AttLinear)
	w.AppendFloat32(l.AttQuadratic)
	w.Sync()
}

type ExtractedDirectionalLight struct {
	Color     glm.Vec3f
	Direction glm.Vec3f
}

func (l ExtractedDirectionalLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Direction)
	w.Sync()
}

func extractLights(
	lights *ExtractedLights,

	config LightConfig,

	pointLights byke.Query[struct {
		Light     PointLight
		Transform GlobalTransform
	}],

	spotLights byke.Query[struct {
		Light     SpotLight
		Transform GlobalTransform
	}],

	directionalLights byke.Query[struct {
		Light     DirectionalLight
		Transform GlobalTransform
	}],
) {
	lights.Clear()

	lights.Ambient = config.Ambient.ToVec3f()

	for item := range pointLights.Items() {
		if item.Light.Color.ToVec3f() == (glm.Vec3f{}) {
			// off, no light
			continue
		}

		lights.PointLights = append(lights.PointLights, ExtractedPointLight{
			Position:     item.Transform.Affine.Translation(),
			Color:        item.Light.Color.ToVec3f(),
			AttConstant:  item.Light.AttConstant,
			AttLinear:    item.Light.AttLinear,
			AttQuadratic: item.Light.AttQuadratic,
		})
	}

	for item := range spotLights.Items() {
		if item.Light.Color.ToVec3f() == (glm.Vec3f{}) {
			// off, no light
			continue
		}

		// light into the negative z axis
		direction := item.Transform.Affine.
			Transform(glm.Vec4f{0, 0, -1, 0}).
			Truncate().
			Normalize()

		lights.SpotLights = append(lights.SpotLights, ExtractedSpotLight{
			Color:        item.Light.Color.ToVec3f(),
			Position:     item.Transform.Affine.Translation(),
			Direction:    direction,
			InnerAngle:   item.Light.InnerAngle,
			OuterAngle:   item.Light.OuterAngle,
			AttConstant:  item.Light.AttConstant,
			AttLinear:    item.Light.AttLinear,
			AttQuadratic: item.Light.AttQuadratic,
		})
	}

	for item := range directionalLights.Items() {
		if item.Light.Color.ToVec3f() == (glm.Vec3f{}) {
			// off, no light
			continue
		}

		// light into the negative z axis
		direction := item.Transform.Affine.
			Transform(glm.Vec4f{0, 0, -1, 0}).
			Truncate().
			Normalize()

		lights.DirectionalLights = append(lights.DirectionalLights, ExtractedDirectionalLight{
			Color:     item.Light.Color.ToVec3f(),
			Direction: direction,
		})
	}
}

type lightsStorage struct {
	BindGroup *wgpu.BindGroup

	BufConfig            *wgpu.Buffer
	BufPointLights       *wgpu.Buffer
	BufDirectionalLights *wgpu.Buffer
	BufSpotLights        *wgpu.Buffer

	staging wgsl.StructWriter
}

func prepareLightsStorage(
	ctx *RenderContext,
	uniforms *lightsStorage,
	lights ExtractedLights,
) {
	s := &uniforms.staging

	s.Clear()
	s.AppendVec3f(lights.Ambient)
	s.WriteTo(ctx, &uniforms.BufConfig, "LightConfig", wgpu.BufferUsageUniform)

	writeSliceToStructWriter(s, lights.DirectionalLights)
	s.WriteTo(ctx, &uniforms.BufDirectionalLights, "DirectionalLights", wgpu.BufferUsageStorage)

	writeSliceToStructWriter(s, lights.PointLights)
	s.WriteTo(ctx, &uniforms.BufPointLights, "PointLights", wgpu.BufferUsageStorage)

	writeSliceToStructWriter(s, lights.SpotLights)
	s.WriteTo(ctx, &uniforms.BufSpotLights, "SpotLights", wgpu.BufferUsageStorage)

	uniforms.staging.AppendUint(uint32(len(lights.PointLights)))
}

func writeSliceToStructWriter[T writerTo](wr *wgsl.StructWriter, values []T) {
	wr.Clear()

	// write number of entries in slice
	wr.AppendUint(uint32(len(values)))

	// write each slice value
	for idx := range values {
		values[idx].WriteTo(wr)
	}
}

type writerTo interface {
	WriteTo(s *wgsl.StructWriter)
}
