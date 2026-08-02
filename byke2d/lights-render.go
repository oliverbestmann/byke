package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/wgsl"
	"github.com/oliverbestmann/webgpu/wgpu"
)

type ExtractedLights struct {
	// Ambient light color
	Ambient glm.Vec3f

	DirectionalLights []ExtractedDirectionalLight
	PointLights       []ExtractedPointLight
	SpotLights        []ExtractedSpotLight
}

func (l *ExtractedLights) Clear() {
	l.Ambient = glm.Vec3f{}
	l.DirectionalLights = l.DirectionalLights[:0]
	l.PointLights = l.PointLights[:0]
	l.SpotLights = l.SpotLights[:0]
}

type ExtractedPointLight struct {
	Position glm.Vec3f
	Color    glm.Vec3f
	Range    float32
}

func (l ExtractedPointLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Position)
	w.AppendFloat32(l.Range)
	w.Sync()
}

type ExtractedSpotLight struct {
	Color      glm.Vec3f
	Position   glm.Vec3f
	Direction  glm.Vec3f
	InnerAngle glm.Rad
	OuterAngle glm.Rad
	Range      float32
}

func (l ExtractedSpotLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Position)
	w.AppendVec3f(l.Direction)
	w.AppendFloat32(float32(l.InnerAngle))
	w.AppendFloat32(float32(l.OuterAngle))
	w.AppendFloat32(l.Range)
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

func extractLightsSystem(
	lights *ExtractedLights,

	ambient GlobalAmbientLight,

	pointLights byke.Query[struct {
		Transform GlobalTransform
		Light     PointLight
	}],

	spotLights byke.Query[struct {
		Transform GlobalTransform
		Light     SpotLight
	}],

	directionalLights byke.Query[struct {
		Transform GlobalTransform
		Light     DirectionalLight
	}],
) {
	lights.Clear()

	lights.Ambient = ambient.Color.ToVec3f().Scale(ambient.Brightness)

	for item := range pointLights.Items() {
		if item.Light.Color.IsBlack() {
			continue
		}

		lights.PointLights = append(lights.PointLights, ExtractedPointLight{
			Position: item.Transform.Affine.Translation(),
			Color:    item.Light.Color.ToVec3f().Scale(item.Light.Intensity),
			Range:    item.Light.Range,
		})
	}

	for item := range spotLights.Items() {
		if item.Light.Color.IsBlack() {
			continue
		}

		// light into the negative z axis
		direction := item.Transform.Affine.
			Transform(glm.Vec4f{0, 0, -1, 0}).
			Truncate().
			Normalize()

		lights.SpotLights = append(lights.SpotLights, ExtractedSpotLight{
			Color:      item.Light.Color.ToVec3f().Scale(item.Light.Intensity),
			Position:   item.Transform.Affine.Translation(),
			Range:      item.Light.Range,
			Direction:  direction,
			InnerAngle: item.Light.InnerAngle,
			OuterAngle: item.Light.OuterAngle,
		})
	}

	for item := range directionalLights.Items() {
		if item.Light.Color.IsBlack() {
			continue
		}

		// light into the negative z axis
		direction := item.Transform.Affine.
			Transform(glm.Vec4f{0, 0, -1, 0}).
			Truncate().
			Normalize()

		lights.DirectionalLights = append(lights.DirectionalLights, ExtractedDirectionalLight{
			Color:     item.Light.Color.ToVec3f().Scale(item.Light.Illuminance),
			Direction: direction,
		})
	}
}

type lightsStorage struct {
	BufConfig            *wgpu.Buffer
	BufPointLights       *wgpu.Buffer
	BufDirectionalLights *wgpu.Buffer
	BufSpotLights        *wgpu.Buffer

	// keep around to reuse the memory allocation
	staging wgsl.StructWriter
}

func prepareLightsStorageSystem(
	ctx *RenderContext,
	uniforms *lightsStorage,
	lights ExtractedLights,
) {
	s := &uniforms.staging

	writeLightConfigToStructWriter(s, lights)
	s.WriteTo(ctx, &uniforms.BufConfig, "LightConfig", wgpu.BufferUsageUniform)

	writeSliceToStructWriter(s, lights.DirectionalLights)
	s.WriteTo(ctx, &uniforms.BufDirectionalLights, "DirectionalLights", wgpu.BufferUsageStorage)

	writeSliceToStructWriter(s, lights.PointLights)
	s.WriteTo(ctx, &uniforms.BufPointLights, "PointLights", wgpu.BufferUsageStorage)

	writeSliceToStructWriter(s, lights.SpotLights)
	s.WriteTo(ctx, &uniforms.BufSpotLights, "SpotLights", wgpu.BufferUsageStorage)

	uniforms.staging.AppendUint(uint32(len(lights.PointLights)))
}

func writeLightConfigToStructWriter(s *wgsl.StructWriter, lights ExtractedLights) {
	s.Clear()
	s.AppendVec3f(lights.Ambient)
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
