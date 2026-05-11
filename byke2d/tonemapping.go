package byke2d

import (
	_ "embed"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/pre"
	"github.com/oliverbestmann/puffin-go"
	"github.com/oliverbestmann/pulse/glm"
	"github.com/oliverbestmann/pulse/wx"
	"github.com/oliverbestmann/webgpu/wgpu"
)

var _ = byke.ValidateComponent[ColorGrading]()
var _ = byke.ValidateComponent[Tonemapping]()
var _ = byke.ValidateComponent[DebandDither]()

//go:embed tonemapping.wgsl
var tonemappingShader string

var _D65Xy = glm.Vec2f{0.31272, 0.32903}
var _D65Lms = glm.Vec3f{0.975538, 1.01648, 1.08475}

// / The matrix that converts from the RGB to the LMS color space.
// /
// / To derive this, first we convert from RGB to [CIE 1931 XYZ]:
// /
// / ```text
// / ⎡ X ⎤   ⎡ 0.490  0.310  0.200 ⎤ ⎡ R ⎤
// / ⎢ Y ⎥ = ⎢ 0.177  0.812  0.011 ⎥ ⎢ G ⎥
// / ⎣ Z ⎦   ⎣ 0.000  0.010  0.990 ⎦ ⎣ B ⎦
// / ```
// /
// / Then we convert to LMS according to the [CAM16 standard matrix]:
// /
// / ```text
// / ⎡ L ⎤   ⎡  0.401   0.650  -0.051 ⎤ ⎡ X ⎤
// / ⎢ M ⎥ = ⎢ -0.250   1.204   0.046 ⎥ ⎢ Y ⎥
// / ⎣ S ⎦   ⎣ -0.002   0.049   0.953 ⎦ ⎣ Z ⎦
// / ```
// /
// / The resulting matrix is just the concatenation of these two matrices, to do
// / the conversion in one step.
// /
// / [CIE 1931 XYZ]: https://en.wikipedia.org/wiki/CIE_1931_color_space
// / [CAM16 standard matrix]: https://en.wikipedia.org/wiki/LMS_color_space
var _rgbToLms = glm.Mat3Of([3][3]float32{
	{0.311692, 0.0905138, 0.00764433},
	{0.652085, 0.901341, 0.0486554},
	{0.0362225, 0.00814478, 0.943700},
})

// / The inverse of the [`RGB_TO_LMS`] matrix, converting from the LMS color
// / space back to RGB.
var _lmsToRgb = glm.Mat3Of([3][3]float32{
	{4.06305, -0.40791, -0.0118812},
	{-2.93241, 1.40437, -0.0486532},
	{-0.130646, 0.00353630, 1.0605344},
})

type ColorGrading struct {
	byke.Component[ColorGrading]
	Global     ColorGradingGlobal
	Shadows    ColorGradingSection
	Midtones   ColorGradingSection
	Highlights ColorGradingSection
}

func (c ColorGrading) HasSectionalColorGrading() bool {
	return c.Shadows != defaultColorGradingSection ||
		c.Midtones != defaultColorGradingSection ||
		c.Highlights != defaultColorGradingSection
}

func (c ColorGrading) ToWGPU() []byte {
	// Compute the balance matrix that will be used to apply the white
	// balance adjustment to an RGB color. Our general approach will be to
	// convert both the color and the developer-supplied white point to the
	// LMS color space, apply the conversion, and then convert back.
	//
	// First, we start with the CIE 1931 *xy* values of the standard D65
	// illuminant:
	// <https://en.wikipedia.org/wiki/Standard_illuminant#D65_values>
	//
	// We then adjust them based on the developer's requested white balance.
	whitePointXY := _D65Xy.Add(glm.Vec2f{-c.Global.Temperature, c.Global.Tint})

	// Convert the white point from CIE 1931 *xy* to LMS. First, we convert to XYZ:
	//
	//                  Y          Y
	//     Y = 1    X = ─ x    Z = ─ (1 - x - y)
	//                  y          y
	//
	// Then we convert from XYZ to LMS color space, using the CAM16 matrix
	// from <https://en.wikipedia.org/wiki/LMS_color_space#Later_CIECAMs>:
	//
	//     ⎡ L ⎤   ⎡  0.401   0.650  -0.051 ⎤ ⎡ X ⎤
	//     ⎢ M ⎥ = ⎢ -0.250   1.204   0.046 ⎥ ⎢ Y ⎥
	//     ⎣ S ⎦   ⎣ -0.002   0.049   0.953 ⎦ ⎣ Z ⎦
	//
	// The following formula is just a simplification of the above.
	whitePointLMS := glm.Vec3f{0.701634, 1.15856, -0.904175}.Add(
		glm.Vec3f{-0.051461, 0.045854, 0.953127}.Add(
			glm.Vec3f{0.452749, -0.296122, -0.955206}.Scale(whitePointXY[0]),
		).Scale(1.0 / whitePointXY[1]),
	)

	// Now that we're in LMS space, perform the white point scaling.
	d := _D65Lms.Div(whitePointLMS)
	whitePointAdjustment := glm.Mat3Of([3][3]float32{
		{d[0], 0, 0},
		{0, d[1], 0},
		{0, 0, d[2]},
	})

	// Finally, combine the RGB → LMS → corrected LMS → corrected RGB
	// pipeline into a single 3×3 matrix.
	balance := _lmsToRgb.Mul(whitePointAdjustment).Mul(_rgbToLms)

	var w wx.StructWriter
	w.AppendMat3f(balance)
	w.AppendVec3f(glm.Vec3f{c.Shadows.Saturation, c.Midtones.Saturation, c.Highlights.Saturation})
	w.AppendVec3f(glm.Vec3f{c.Shadows.Contrast, c.Midtones.Contrast, c.Highlights.Contrast})
	w.AppendVec3f(glm.Vec3f{c.Shadows.Gamma, c.Midtones.Gamma, c.Highlights.Gamma})
	w.AppendVec3f(glm.Vec3f{c.Shadows.Gain, c.Midtones.Gain, c.Highlights.Gain})
	w.AppendVec3f(glm.Vec3f{c.Shadows.Lift, c.Midtones.Lift, c.Highlights.Lift})
	w.AppendVec2f(glm.Vec2f{c.Global.MidtonesRange.Begin, c.Global.MidtonesRange.EndExclusive})
	w.AppendFloat32(c.Global.Exposure)
	w.AppendFloat32(c.Global.Hue)
	w.AppendFloat32(c.Global.PostSaturation)
	return w.Bytes()
}

type ColorGradingGlobal struct {
	Exposure       float32
	Temperature    float32
	Tint           float32
	Hue            float32
	PostSaturation float32
	MidtonesRange  Range
}

var defaultColorGradingGlobal = ColorGradingGlobal{
	Exposure:       0,
	Temperature:    0,
	Tint:           0,
	Hue:            0,
	PostSaturation: 1.0,
	MidtonesRange: Range{
		Begin:        0.2,
		EndExclusive: 0.7,
	},
}

type ColorGradingSection struct {
	Saturation float32
	Contrast   float32
	Gamma      float32
	Gain       float32
	Lift       float32
}

var defaultColorGradingSection = ColorGradingSection{
	Saturation: 1,
	Contrast:   1,
	Gamma:      1,
	Gain:       1,
	Lift:       0,
}

var DefaultColorGrading = ColorGrading{
	Global:     defaultColorGradingGlobal,
	Shadows:    defaultColorGradingSection,
	Midtones:   defaultColorGradingSection,
	Highlights: defaultColorGradingSection,
}

type DebandDither struct {
	byke.Component[DebandDither]
	enable bool
}

var DebandDitherOn = DebandDither{enable: true}
var DebandDitherOff = DebandDither{enable: false}

type Tonemapping struct {
	byke.Component[Tonemapping]
	value uint8
}

func (t Tonemapping) String() string {
	switch t {
	case TonemappingSomewhatBoringDisplayTransform:
		return "SomewhatBoringDisplayTransform"
	case TonemappingAcesFitted:
		return "AcesFitted"
	case TonemappingReinhard:
		return "Reinhard"
	case TonemappingReinhardLuminance:
		return "ReinhardLuminance"
	case TonemappingTonyMcMapface:
		return "TonyMcMapface"
	case TonemappingAgX:
		return "AgX"
	case TonemappingBlenderFilmic:
		return "BlenderFilmic"
	default:
		return "None"
	}
}

var TonemappingNone = Tonemapping{value: 0}
var TonemappingSomewhatBoringDisplayTransform = Tonemapping{value: 1}
var TonemappingAcesFitted = Tonemapping{value: 2}
var TonemappingReinhard = Tonemapping{value: 3}
var TonemappingReinhardLuminance = Tonemapping{value: 4}
var TonemappingTonyMcMapface = Tonemapping{value: 5}
var TonemappingAgX = Tonemapping{value: 6}
var TonemappingBlenderFilmic = Tonemapping{value: 7}

type tonemappingPipelineConfig struct {
	TargetFormat          wgpu.TextureFormat
	Tonemapping           Tonemapping
	DebandDither          DebandDither
	HueRotate             bool
	WhiteBalance          bool
	SectionalColorGrading bool
}

func (c tonemappingPipelineConfig) Specialize(ctx *RenderContext) *wgpu.RenderPipeline {
	var defs = pre.Values{}

	switch c.Tonemapping {
	case TonemappingSomewhatBoringDisplayTransform:
		defs.Define("TONEMAP_METHOD_SOMEWHAT_BORING_DISPLAY_TRANSFORM", true)
	case TonemappingAcesFitted:
		defs.Define("TONEMAP_METHOD_ACES_FITTED", true)
	case TonemappingReinhard:
		defs.Define("TONEMAP_METHOD_REINHARD", true)
	case TonemappingReinhardLuminance:
		defs.Define("TONEMAP_METHOD_REINHARD_LUMINANCE", true)
	case TonemappingTonyMcMapface:
		defs.Define("TONEMAP_METHOD_TONY_MC_MAPFACE", true)
	case TonemappingAgX:
		defs.Define("TONEMAP_METHOD_AGX", true)
	case TonemappingBlenderFilmic:
		defs.Define("TONEMAP_METHOD_BLENDER_FILMIC", true)
	default:
		defs.Define("TONEMAP_METHOD_NONE", true)
	}

	defs.Define("DEBAND_DITHER", c.DebandDither.enable)

	defs.Define("HUE_ROTATE", c.HueRotate)
	defs.Define("WHITE_BALANCE", c.WhiteBalance)
	defs.Define("SECTIONAL_COLOR_GRADING", c.SectionalColorGrading)

	shaderSource, err := pre.Process(tonemappingShader, defs)
	if err != nil {
		panic(err)
	}

	module := ctx.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      "Tonemapping",
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: shaderSource},
	})

	vertexState, primitiveState := prepareFullscreenShader(ctx)

	return ctx.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label: "",
		Fragment: &wgpu.FragmentState{
			Module:     module,
			EntryPoint: "fragment",
			Targets: []wgpu.ColorTargetState{
				{
					Format:    c.TargetFormat,
					Blend:     &wgpu.BlendStateReplace,
					WriteMask: wgpu.ColorWriteMaskAll,
				},
			},
		},
		Vertex:    vertexState,
		Primitive: primitiveState,
		Multisample: wgpu.MultisampleState{
			Count: 1,
			Mask:  0xffffffff,
		},
	})
}

func tonemappingSystem(
	ctx *RenderContext,
	pipelines Pipelines[tonemappingPipelineConfig],
	uniforms *ComponentUniforms[ColorGrading],
	luts *TonemappingLutTextures,
	viewQuery ViewQuery[struct {
		ColorGrading       ColorGrading
		Tonemapping        Tonemapping
		DebandDither       DebandDither
		ViewTarget         *ViewTarget
		ColorGradingOffset DynamicOffset[ColorGrading]
	}],
) {
	defer puffin.NewScope("Tonemapping").End()

	view := viewQuery.Get()

	pp := view.ViewTarget.PostProcess()

	pipeline := pipelines.Specialize(tonemappingPipelineConfig{
		TargetFormat:          view.ViewTarget.Format,
		Tonemapping:           view.Tonemapping,
		DebandDither:          view.DebandDither,
		HueRotate:             view.ColorGrading.Global.Hue != 0,
		WhiteBalance:          view.ColorGrading.Global.Temperature != 0 || view.ColorGrading.Global.Tint != 0,
		SectionalColorGrading: view.ColorGrading.HasSectionalColorGrading(),
	})

	sampler := ctx.CreateSampler(wgpu.SamplerDescriptor{
		Label:        "Tonemapping",
		AddressModeU: wgpu.AddressModeClampToEdge,
		AddressModeV: wgpu.AddressModeClampToEdge,
		AddressModeW: wgpu.AddressModeClampToEdge,
		MagFilter:    wgpu.FilterModeNearest,
		MinFilter:    wgpu.FilterModeNearest,
		MipmapFilter: wgpu.MipmapFilterModeNearest,
	})

	bindGroupEntries := []wgpu.BindGroupEntry{
		uniforms.Binding(),
		BindingTextureView(pp.Source),
		BindingSampler(sampler),
	}

	switch view.Tonemapping {
	case TonemappingTonyMcMapface:
		bindGroupEntries = append(bindGroupEntries,
			BindingTextureView(luts.TonyMcMapface(ctx).TextureView),
			BindingSampler(luts.TonyMcMapface(ctx).Sampler),
		)
	case TonemappingAgX:
		bindGroupEntries = append(bindGroupEntries,
			BindingTextureView(luts.AgX(ctx).TextureView),
			BindingSampler(luts.AgX(ctx).Sampler),
		)
	case TonemappingBlenderFilmic:
		bindGroupEntries = append(bindGroupEntries,
			BindingTextureView(luts.BlenderFilmic(ctx).TextureView),
			BindingSampler(luts.BlenderFilmic(ctx).Sampler),
		)
	}

	bindGroup := ctx.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:   "Tonemapping",
		Layout:  pipeline.GetBindGroupLayout(0),
		Entries: Sequential(bindGroupEntries...),
	})

	defer bindGroup.Release()

	enc := ctx.CreateCommandEncoder(&wgpu.CommandEncoderDescriptor{Label: "Tonemapping"})
	defer enc.Release()

	pass := enc.BeginRenderPass(&wgpu.RenderPassDescriptor{
		Label: "Tonemapping",
		ColorAttachments: []wgpu.RenderPassColorAttachment{
			pp.Target.UnsampledAttachment(),
		},
	})

	pass.SetPipeline(pipeline.Get())
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Draw(3, 1, 0, 0)
	pass.End()

	ctx.Submit(enc.Finish(nil))
}
