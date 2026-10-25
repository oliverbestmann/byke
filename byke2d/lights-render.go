package byke2d

import (
	"github.com/chewxy/math32"
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

	// Width and height of each spot light shadow map
	ShadowMapSize uint32
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

	// Cube in the point shadow map array, -1 if the light casts no shadows
	ShadowIndex      int32
	ShadowDepthBias  float32
	ShadowNormalBias float32
	ShadowMapNearZ   float32
}

func (l ExtractedPointLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Position)
	w.AppendFloat32(l.Range)
	w.AppendInt(l.ShadowIndex)
	w.AppendFloat32(l.ShadowDepthBias)
	w.AppendFloat32(l.ShadowNormalBias)
	w.AppendFloat32(l.ShadowMapNearZ)
	w.Sync()
}

type ExtractedSpotLight struct {
	Color      glm.Vec3f
	Position   glm.Vec3f
	Direction  glm.Vec3f
	InnerAngle glm.Rad
	OuterAngle glm.Rad
	Range      float32

	// Layer in the shadow map array, -1 if the light casts no shadows
	ShadowLayer      int32
	ShadowDepthBias  float32
	ShadowNormalBias float32

	// view and projection of the shadow map, only set if ShadowLayer >= 0
	WorldToView glm.Mat4f
	ViewToClip  glm.Mat4f
}

func (l ExtractedSpotLight) WriteTo(w *wgsl.StructWriter) {
	w.AppendVec3f(l.Color)
	w.AppendVec3f(l.Position)
	w.AppendVec3f(l.Direction)
	w.AppendFloat32(float32(l.InnerAngle))
	w.AppendFloat32(float32(l.OuterAngle))
	w.AppendFloat32(l.Range)
	w.AppendMat4f(l.ViewToClip.Mul(l.WorldToView))
	w.AppendInt(l.ShadowLayer)
	w.AppendFloat32(l.ShadowDepthBias)
	w.AppendFloat32(l.ShadowNormalBias)
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
	shadowConfig ShadowMapConfig,

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
	lights.ShadowMapSize = shadowConfig.Size

	var shadowLayer, shadowIndex int32

	for item := range pointLights.Items() {
		if item.Light.Color.IsBlack() {
			continue
		}

		extracted := ExtractedPointLight{
			Position:    item.Transform.Affine.Translation(),
			Color:       item.Light.Color.ToVec3f().Scale(item.Light.Intensity),
			Range:       item.Light.Range,
			ShadowIndex: -1,
		}

		if item.Light.ShadowsEnabled {
			extracted.ShadowIndex = shadowIndex
			shadowIndex++

			extracted.ShadowMapNearZ = item.Light.ShadowMapNearZ
			if extracted.ShadowMapNearZ == 0 {
				extracted.ShadowMapNearZ = DefaultPointLight.ShadowMapNearZ
			}

			// a cube face covers 90 degrees, so a texel at distance 1 is 2/size wide.
			// sqrt2 accounts for the worst case diagonal offset
			texelSize := 2 / float32(shadowConfig.Size)
			extracted.ShadowNormalBias = item.Light.ShadowNormalBias * texelSize * math32.Sqrt2
			extracted.ShadowDepthBias = item.Light.ShadowDepthBias
		}

		lights.PointLights = append(lights.PointLights, extracted)
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

		position := item.Transform.Affine.Translation()

		extracted := ExtractedSpotLight{
			Color:       item.Light.Color.ToVec3f().Scale(item.Light.Intensity),
			Position:    position,
			Range:       item.Light.Range,
			Direction:   direction,
			InnerAngle:  item.Light.InnerAngle,
			OuterAngle:  item.Light.OuterAngle,
			ShadowLayer: -1,
		}

		if item.Light.ShadowsEnabled {
			extracted.ShadowLayer = shadowLayer
			shadowLayer++

			near := item.Light.ShadowMapNearZ
			if near == 0 {
				near = DefaultSpotLight.ShadowMapNearZ
			}

			// any up vector works for a round cone, as long as it is not parallel to the direction
			up := glm.Vec3f{0, 1, 0}
			if math32.Abs(direction[1]) > 0.99 {
				up = glm.Vec3f{1, 0, 0}
			}

			extracted.WorldToView = glm.LookAt(position, position.Add(direction), up)
			extracted.ViewToClip = glm.Perspective(2*item.Light.OuterAngle, 1, near, 0)

			// the normal bias is given in texels, scale it to the texel size at a distance of 1.
			// sqrt2 accounts for the worst case diagonal offset
			texelSize := 2 * math32.Tan(float32(item.Light.OuterAngle)) / float32(shadowConfig.Size)
			extracted.ShadowNormalBias = item.Light.ShadowNormalBias * texelSize * math32.Sqrt2
			extracted.ShadowDepthBias = item.Light.ShadowDepthBias
		}

		lights.SpotLights = append(lights.SpotLights, extracted)
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
