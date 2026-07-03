#import byke2d::mesh2d

struct ColorMaterial {
    color: vec4f,
}

@group(1)
@binding(0)
var<storage> materials: array<ColorMaterial>;

@group(1)
@binding(1)
var texture: texture_2d<f32>;

@group(1)
@binding(2)
var texture_sampler: sampler;

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out = default_mesh2d_vertex(in);
    out.color *= material.color;
    return out;
}

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    var out = default_mesh2d_fragment(vertex);

    #ifdef MESH2D_COLOR_HAS_TEXTURE
    out *= textureSample(texture, texture_sampler, vertex.uv);
    #endif

    return out;
}
