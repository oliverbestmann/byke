struct VertexInput {
    @builtin(vertex_index) index: u32,

    @location(0) i_translation: vec2<f32>,
    @location(1) i_scale: vec2<f32>,
    @location(2) i_rotation: f32,
    @location(3) i_uv_offset: vec2<f32>,
    @location(4) i_uv_scale: vec2<f32>,
    @location(5) i_color: vec4<f32>,
}

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) @interpolate(flat) color: vec4<f32>,
    @location(1) uv: vec2<f32>,
};

struct View {
    screen_to_ndc: mat3x3<f32>,
    world_to_screen: mat3x3<f32>,
};

@group(0)
@binding(0)
var<uniform> view: View;

fn mat3_scale(scale: vec2<f32>) -> mat3x3<f32> {
    return mat3x3(
        vec3<f32>(scale.x, 0, 0),
        vec3<f32>(0, scale.y, 0),
        vec3<f32>(0, 0, 1),
    );
}

fn mat3_rotation(rotation: f32) -> mat3x3<f32> {
    let s = sin(rotation);
    let c = cos(rotation);

    return mat3x3(
        vec3<f32>(c, s, 0),
        vec3<f32>(-s, c, 0),
        vec3<f32>(0, 0, 1),
    );
}

fn mat3_translation(translation: vec2<f32>) -> mat3x3<f32> {
    let x = translation.x;
    let y = translation.y;

    return mat3x3(
        vec3<f32>(1, 0, 0),
        vec3<f32>(0, 1, 0),
        vec3<f32>(x, y, 1),
    );
}

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    let model_to_world = mat3_translation(in.i_translation)
        * mat3_scale(in.i_scale*2)
        * mat3_rotation(in.i_rotation);

    // between 0 and 1
    // index vertices as p00, p01, p10, p11, this way
    // x and y can be derived from the lower bit of index
    let x = f32((in.index >> 1) & 1);
    let y = f32(in.index & 1);
    let vertex_position = vec2f(x, y);

    let identity = mat3x3f(1, 0, 0,  0, 1, 0,  0, 0, 1);
    let position = identity
        * view.screen_to_ndc
        * view.world_to_screen
        * model_to_world
        * vec3<f32>(vertex_position * 64.0, 1.0);

    // move the vertex to the world
    var out: VertexOutput;
    out.position = vec4(position.xy, 0.0, 1.0);
    out.uv = vertex_position;
    out.color = vec4f(1, 1, 1, 1);

    return out;
}

@group(1)
@binding(0)
var texture: texture_2d<f32>;

@group(1)
@binding(1)
var texture_sampler: sampler;

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    let tex = textureSample(texture, texture_sampler, vertex.uv);
    return tex * vertex.color + vec4f(0.5, 0, 0.4, 1.0);
}
