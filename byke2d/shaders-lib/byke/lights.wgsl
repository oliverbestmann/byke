struct LightConfig {
    ambient: vec3f,
}

struct DirectionalLights {
    count: u32,
    lights: array<DirectionalLight>,
}

struct DirectionalLight {
    color: vec3f,
    direction: vec3f,
}

struct PointLights {
    count: u32,
    lights: array<PointLight>,
}

struct PointLight {
    color: vec3f,
    position: vec3f,
    range: f32,
}

struct SpotLights {
    count: u32,
    lights: array<SpotLight>,
}

struct SpotLight {
    color: vec3f,
    position: vec3f,
    direction: vec3f,
    inner_angle: f32,
    outer_angle: f32,
    range: f32,
}

@group(0)
@binding(10)
var<uniform> light_config: LightConfig;

@group(0)
@binding(11)
var<storage> directional_lights: DirectionalLights;

@group(0)
@binding(12)
var<storage> point_lights: PointLights;

@group(0)
@binding(13)
var<storage> spot_lights: SpotLights;
