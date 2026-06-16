#module byke2d::lights

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
    att_constant: f32,
    att_linear: f32,
    att_quadratic: f32,
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
    att_constant: f32,
    att_linear: f32,
    att_quadratic: f32,
}

