// must match fullscreen_vertex.wgsl
struct FullscreenVertexOutput {
    @builtin(position)
    position: vec4<f32>,

    @location(0)
    uv: vec2<f32>,
};

@group(0) @binding(0) var in_texture: texture_2d<f32>;
@group(0) @binding(1) var in_sampler: sampler;

@fragment
fn fs_main(in: FullscreenVertexOutput) -> @location(0) vec4<f32> {
    return textureSample(in_texture, in_sampler, in.uv);
}
