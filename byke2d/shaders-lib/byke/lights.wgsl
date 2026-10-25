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
    // projects from world space into the clip space of the shadow map
    world_to_clip: mat4x4f,
    // layer in spot_shadow_maps, -1 if the light casts no shadows
    shadow_layer: i32,
    shadow_depth_bias: f32,
    // already scaled to the texel size at a distance of one
    shadow_normal_bias: f32,
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

@if(MESH_SHADOWS)
@group(0)
@binding(14)
var spot_shadow_maps: texture_depth_2d_array;

@if(MESH_SHADOWS)
@group(0)
@binding(15)
var spot_shadow_sampler: sampler_comparison;

// Returns 0 if the position is in the shadow of the light and 1 if it is lit.
@if(MESH_SHADOWS)
fn spot_shadow(light: SpotLight, position: vec3f, normal: vec3f) -> f32 {
    if light.shadow_layer < 0 {
        return 1.0;
    }

    let surface_to_light = light.position - position;

    // texels get larger with distance, so the normal bias grows with it
    let distance_along_axis = dot(light.direction, -surface_to_light);

    let offset_position = position
        + normal * light.shadow_normal_bias * distance_along_axis
        + normalize(surface_to_light) * light.shadow_depth_bias;

    let clip = light.world_to_clip * vec4f(offset_position, 1.0);
    if clip.w <= 0.0 {
        return 1.0;
    }

    let ndc = clip.xyz / clip.w;
    let uv = ndc.xy * vec2f(0.5, -0.5) + vec2f(0.5);
    if any(uv < vec2f(0.0)) || any(uv > vec2f(1.0)) {
        return 1.0;
    }

    // Reverse-z (near is depth 1, far is depth 0): lit if the fragment is at least as close as the stored depth.
    // Use the Level variant: this runs in non-uniform control flow, where implicit mip selection is not allowed.
    return textureSampleCompareLevel(spot_shadow_maps, spot_shadow_sampler, uv, light.shadow_layer, ndc.z);
}
