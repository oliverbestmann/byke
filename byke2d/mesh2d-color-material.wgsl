#import byke2d::mesh3d

struct ColorMaterial {
    color: vec4f,
}

@group(2)
@binding(0)
var<storage, read> materials: array<ColorMaterial>;

@group(2)
@binding(1)
var texture: texture_2d<f32>;

@group(2)
@binding(2)
var texture_sampler: sampler;

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out = default_mesh3d_vertex(in);
    out.color *= materials[out.material].color;
    return out;
}

@fragment
fn fs_main(param: VertexOutput, @builtin(front_facing) front_facing: bool) -> @location(0) vec4f {
    var vertex = param;

    let m = materials[vertex.material];

    var fin: FragmentIn;
    fin.ambient_occlusion = 1.0;

    var out = default_mesh3d_fragment(vertex, fin);

#ifdef MESH3D_MAT_HAS_TEXTURE
    let texcol = textureSample(texture, texture_sampler, vertex.uv);
    out *= texcol;
#endif

#ifdef ALPHAMODE_ALPHA_TO_COVERAGE
    out.a = (out.a - 0.5) / max(fwidth(out.a), 0.0001) + 0.5;
#endif

    return out;
}


