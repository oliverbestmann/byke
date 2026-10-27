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
    // projects from world space into the clip space of the shadow map
    world_to_clip: mat4x4f,
    // layer in shadow_maps_2d, -1 if the light casts no shadows
    shadow_layer: i32,
    shadow_depth_bias: f32,
    // already scaled to the texel size
    shadow_normal_bias: f32,
    // how much the shadow darkens the ambient and environment light, 0 to 1
    environment_shadow: f32,
}

struct PointLights {
    count: u32,
    lights: array<PointLight>,
}

struct PointLight {
    color: vec3f,
    position: vec3f,
    range: f32,
    // cube in point_shadow_maps, -1 if the light casts no shadows
    shadow_index: i32,
    shadow_depth_bias: f32,
    // already scaled to the texel size at a distance of one
    shadow_normal_bias: f32,
    shadow_near_z: f32,
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
    // layer in shadow_maps_2d, -1 if the light casts no shadows
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
var shadow_maps_2d: texture_depth_2d_array;

@if(MESH_SHADOWS)
@group(0)
@binding(15)
var shadow_sampler: sampler_comparison;

@if(MESH_SHADOWS)
@group(0)
@binding(16)
var point_shadow_maps: texture_depth_cube_array;

// Returns 0 if the position is in the shadow of the light and 1 if it is lit.
@if(MESH_SHADOWS)
fn point_shadow(light: PointLight, position: vec3f, normal: vec3f) -> f32 {
    if light.shadow_index < 0 {
        return 1.0;
    }

    // Each cube face has a 90 degree frustum aligned with the axes, so the view
    // space depth on the face we sample is the largest absolute component.
    let surface_to_light = light.position - position;
    let abs_surface_to_light = abs(surface_to_light);
    let distance_to_light = max(abs_surface_to_light.x, max(abs_surface_to_light.y, abs_surface_to_light.z));

    // texels get larger with distance, so the normal bias grows with it
    let offset_position = position
        + normal * light.shadow_normal_bias * distance_to_light
        + normalize(surface_to_light) * light.shadow_depth_bias;

    let light_to_surface = offset_position - light.position;
    let abs_light_to_surface = abs(light_to_surface);
    let major_axis = max(abs_light_to_surface.x, max(abs_light_to_surface.y, abs_light_to_surface.z));

    // the infinite reverse-z projection maps view depth z to near / z
    let depth = light.shadow_near_z / major_axis;

    // Cube maps are left-handed, flip z like the skybox does.
    // Use the Level variant: this runs in non-uniform control flow, where implicit mip selection is not allowed.
    return textureSampleCompareLevel(point_shadow_maps, shadow_sampler, light_to_surface * vec3f(1.0, 1.0, -1.0), light.shadow_index, depth);
}

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

    return sample_shadow_map_2d(light.world_to_clip, light.shadow_layer, offset_position);
}

// Returns 0 if the position is in the shadow of the light and 1 if it is lit.
@if(MESH_SHADOWS)
fn directional_shadow(light: DirectionalLight, position: vec3f, normal: vec3f) -> f32 {
    if light.shadow_layer < 0 {
        return 1.0;
    }

    // orthographic projection: texels have the same size everywhere
    let offset_position = position
        + normal * light.shadow_normal_bias
        - light.direction * light.shadow_depth_bias;

    return sample_shadow_map_2d(light.world_to_clip, light.shadow_layer, offset_position);
}

// Projects the position into the shadow map at the given layer of shadow_maps_2d.
// Returns 0 if it is in shadow and 1 if it is lit or outside of the map.
@if(MESH_SHADOWS)
fn sample_shadow_map_2d(world_to_clip: mat4x4f, layer: i32, position: vec3f) -> f32 {
    let clip = world_to_clip * vec4f(position, 1.0);
    if clip.w <= 0.0 {
        return 1.0;
    }

    let ndc = clip.xyz / clip.w;
    let uv = ndc.xy * vec2f(0.5, -0.5) + vec2f(0.5);
    if any(uv < vec2f(0.0)) || any(uv > vec2f(1.0)) || ndc.z < 0.0 || ndc.z > 1.0 {
        return 1.0;
    }

    // Reverse-z (near is depth 1, far is depth 0): lit if the fragment is at least as close as the stored depth.
    // Use the Level variant: this runs in non-uniform control flow, where implicit mip selection is not allowed.
    return textureSampleCompareLevel(shadow_maps_2d, shadow_sampler, uv, layer, ndc.z);
}
