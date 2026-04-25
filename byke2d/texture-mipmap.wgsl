
struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) uv: vec2<f32>,
};


@vertex
fn vs_main(@builtin(vertex_index) index: u32) -> VertexOutput {
    // between 0 and 1
    // index vertices as p00, p01, p10, p11, this way
    // x and y can be derived from the lower bit of index
    let x = f32((index >> 1) & 1);
    let y = f32(index & 1);
    let uv = vec2f(x, y);

    var out: VertexOutput;
    out.position = vec4(uv * vec2(2, -2) + vec2(-1, 1), 0.0, 1.0);
    out.uv = uv;

    return out;
}

@group(0)
@binding(0)
var texture: texture_2d<f32>;

@group(0)
@binding(1)
var texture_sampler: sampler;

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    return textureSample(texture, texture_sampler, vertex.uv);
}
