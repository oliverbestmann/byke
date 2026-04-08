struct VertexOutput {
    @builtin(position) position: vec4f,
    @location(0) color: vec4f,
    @location(1) uv: vec2f,
};

@vertex
fn vs_main(@builtin(vertex_index) index: u32) -> VertexOutput {
    // between 0 and 1
    // index vertices as p00, p01, p10, p11, this way
    // x and y can be derived from the lower bit of index
    let x = f32((index >> 1) & 1);
    let y = f32(index & 1);
    let vertex = vec2f(x, y);

    var result: VertexOutput;
    result.position = vec4(vertex, 0.0, 1.0);
    result.uv = vec2(vertex);
    result.color = vec4(1.0, 0.2, 0.8, 1.0);
    return result;
}

@group(0)
@binding(0)
var texture: texture_2d<f32>;

@group(0)
@binding(1)
var texSampler: sampler;

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    let tex = textureSample(texture, texSampler, vertex.uv);
    return tex * vertex.color;
}
