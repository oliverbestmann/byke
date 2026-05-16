#module byke2d::sprite

#import byke2d::view
#import byke2d::view::binding

struct VertexInput {
    @builtin(vertex_index) index: u32,

    @location(0) i_affine_0: vec4<f32>,
    @location(1) i_affine_1: vec4<f32>,
    @location(2) i_affine_2: vec4<f32>,
    @location(3) i_affine_3: vec4<f32>,
    @location(4) i_uv_offset: vec2<f32>,
    @location(5) i_uv_scale: vec2<f32>,
    @location(6) i_color: vec4<f32>,
    @location(7) i_flags: u32,
}

struct VertexOutput {
    @builtin(position) position: vec4<f32>,
    @location(0) @interpolate(flat) color: vec4<f32>,
    @location(1) uv: vec2<f32>,
    @location(2) flags: u32,
};

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

const indices = array<u32, 6>(2, 0, 1, 1, 3, 2);

fn default_sprite_vertex(in: VertexInput) -> VertexOutput {
    let index = indices[in.index];

    // transforms the unit square to its target coordinates
    let model_to_world = mat4x4f(in.i_affine_0, in.i_affine_1, in.i_affine_2, in.i_affine_3);

    // between 0 and 1
    // index vertices as p00, p01, p10, p11, this way
    // x and y can be derived from the lower bit of index
    let x = f32((index >> 1) & 1);
    let y = f32(index & 1);
    let vertex_position = vec2f(x, y);

    let identity = mat4x4f(1, 0, 0, 0,  0, 1, 0, 0,  0, 0, 1, 0,  0, 0, 0, 1);
    let position = identity
        * view.screen_to_ndc
        * view.world_to_screen
        * model_to_world
        * vec4f(vertex_position, 0.0, 1.0);

    // move the vertex to the world
    var out: VertexOutput;
    out.position = vec4(position.xy, 0.0, 1.0);
    out.uv = vertex_position * in.i_uv_scale + in.i_uv_offset;
    out.color = in.i_color;
    out.flags = in.i_flags;

    return out;
}

fn default_sprite_fragment(vertex: VertexOutput) -> vec4f {
    let tex = textureSample(texture, texture_sampler, vertex.uv);

    if (vertex.flags & 1) != 0 {
        // use the red channel of the texture as alpha
        return vec4(vertex.color.rgb, tex.r * vertex.color.a);
    }

    return tex * vertex.color;
}

@group(1)
@binding(0)
var texture: texture_2d<f32>;

@group(1)
@binding(1)
var texture_sampler: sampler;

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    return default_sprite_vertex(in);
}

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    return default_sprite_fragment(vertex);
}
