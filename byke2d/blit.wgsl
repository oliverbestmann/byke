import package::byke::fullscreen;

@vertex
fn fullscreen_vertex_shader(@builtin(vertex_index) vertex_index: u32) -> fullscreen::FullscreenVertexOutput {
    return fullscreen::vertex(vertex_index);
}

@group(0) @binding(0) var in_texture: texture_2d<f32>;
@group(0) @binding(1) var in_sampler: sampler;

@fragment
fn fs_main(in: fullscreen::FullscreenVertexOutput) -> @location(0) vec4<f32> {
    return textureSample(in_texture, in_sampler, in.uv);
}
