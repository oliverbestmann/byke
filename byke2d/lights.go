package byke2d

import (
	"math"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/spoke"
)

var (
	_ = byke.ValidateComponent[DirectionalLight]()
	_ = byke.ValidateComponent[PointLight]()
	_ = byke.ValidateComponent[SpotLight]()
)

// GlobalAmbientLight configures the global ambient light, which lights the entire scene equally.
// This resource is inserted by default and is set to a low ambient light.
type GlobalAmbientLight struct {
	// Color of the global ambient light
	Color Color

	// A direct scale factor multiplied with color before being passed to the shader.
	// After applying this multiplier, the resulting value should be in units of cd/m^2.
	Brightness float32
}

var DefaultGlobalAmbientLight = GlobalAmbientLight{
	Color:      ColorWhite,
	Brightness: 80.0,
}

var GlobalAmbientLightNone = GlobalAmbientLight{}

// DirectionalLight component.
//
// Directional lights don't exist in reality but they are a good
// approximation for light sources VERY far away, like the sun or
// the moon.
//
// The light shines along the forward direction of the entity's transform. With a default transform
// this would be along the negative-Z axis.
//
// Valid values for `illuminance` are:
//
// | Illuminance (lux) | Surfaces illuminated by                        |
// |-------------------|------------------------------------------------|
// | 0.0001            | Moonless, overcast night sky (starlight)       |
// | 0.002             | Moonless clear night sky with airglow          |
// | 0.05–0.3          | Full moon on a clear night                     |
// | 3.4               | Dark limit of civil twilight under a clear sky |
// | 20–50             | Public areas with dark surroundings            |
// | 50                | Family living room lights                      |
// | 80                | Office building hallway/toilet lighting        |
// | 100               | Very dark overcast day                         |
// | 150               | Train station platforms                        |
// | 320–500           | Office lighting                                |
// | 400               | Sunrise or sunset on a clear day.              |
// | 1000              | Overcast day; typical TV studio lighting       |
// | 10,000–25,000     | Full daylight (not direct sun)                 |
// | 32,000–100,000    | Direct sunlight                                |
//
// Source: [Wikipedia](https://en.wikipedia.org/wiki/Lux)
type DirectionalLight struct {
	byke.Component[DirectionalLight]

	// The color of the light
	Color Color

	// Illuminance in lux (lumens per square meter), representing the amount of
	// light projected onto surfaces by this light source. Lux is used here
	// instead of lumens because a directional light illuminates all surfaces
	// more-or-less the same way (depending on the angle of incidence). Lumens
	// can only be specified for light sources which emit light from a specific
	// area.
	// The default is roughly ambient daylight at 10,000.
	Illuminance float32
}

var DefaultDirectionalLight = DirectionalLight{
	Color:       ColorWhite,
	Illuminance: 10_000,
}

func (DirectionalLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

type PointLight struct {
	byke.Component[PointLight]

	// The color of this light source.
	Color Color

	// Luminous power in lumens, representing the amount of light
	// emitted by this source in all directions.
	Intensity float32

	// Cut-off for the light's area-of-effect. Fragments outside this range will not be affected by
	// this light at all, so it's important to tune this together with `intensity` to prevent hard
	// lighting cut-offs.
	Range float32
}

var DefaultPointLight = PointLight{
	Color:     ColorWhite,
	Intensity: 1_000_000,
	Range:     20,
}

func (PointLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

// SpotLight describes a spotlight.
// It shines into local -z direction.
type SpotLight struct {
	byke.Component[SpotLight]
	Color Color

	// Luminous power in lumens, representing the amount of light
	// emitted by this source in all directions.
	Intensity float32

	Range      float32
	InnerAngle glm.Rad
	OuterAngle glm.Rad
}

var DefaultSpotLight = SpotLight{
	Color:      ColorWhite,
	Intensity:  1_000_000,
	Range:      20,
	OuterAngle: math.Pi / 4,
}

func (SpotLight) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

func pluginLights(app *byke.App) {
	app.InsertResource(DefaultGlobalAmbientLight)
	app.InsertResource(ExtractedLights{})
	app.InsertResource(lightsStorage{})
	app.AddSystems(Render, byke.System(extractLightsSystem).InSet(RenderPhaseExtract))
	app.AddSystems(Render, byke.System(prepareLightsStorageSystem).InSet(RenderPhasePrepareResources))
}
