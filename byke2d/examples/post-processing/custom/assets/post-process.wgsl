import package::byke::fullscreen;

@group(0) @binding(0) var screen_texture: texture_2d<f32>;
@group(0) @binding(1) var screen_sampler: sampler;

// How far the red and blue channels move apart, as a fraction of the distance to the center.
const COLOR_SPLIT: f32 = 0.02;

// How much the corners darken.
const VIGNETTE: f32 = 0.7;

@vertex
fn fullscreen_vertex_shader(@builtin(vertex_index) vertex_index: u32) -> fullscreen::FullscreenVertexOutput {
    return fullscreen::vertex(vertex_index);
}

@fragment
fn fragment(in: fullscreen::FullscreenVertexOutput) -> @location(0) vec4<f32> {
    // both effects grow towards the edges of the screen
    let offset = in.uv - 0.5;

    // color split: sample red and blue a little outwards and inwards
    let shift = offset * COLOR_SPLIT;
    let red = textureSample(screen_texture, screen_sampler, in.uv + shift).r;
    let center = textureSample(screen_texture, screen_sampler, in.uv);
    let blue = textureSample(screen_texture, screen_sampler, in.uv - shift).b;

    // vignette: 0 at the center, 1 in the corners
    let distance = length(offset) * sqrt(2.0);
    let darken = 1.0 - VIGNETTE * smoothstep(0.4, 1.0, distance);

    return vec4(vec3(red, center.g, blue) * darken, center.a);
}
