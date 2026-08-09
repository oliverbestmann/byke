import package::byke::mesh;

struct ColorMaterial {
    color: vec4f,
    alpha_cutoff: f32,
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
fn vs_main(in: mesh::VertexInput) -> mesh::VertexOutput {
    var out = mesh::process_vertex(in);
    out.color *= materials[out.material].color;
    return out;
}

@fragment
fn fs_main(param: mesh::VertexOutput, @builtin(front_facing) front_facing: bool) -> @location(0) vec4f {
    var vertex = param;

    let m = materials[vertex.material];

    var out = mesh::default_mesh3d_fragment(vertex);

    @if(MESH3D_MAT_HAS_TEXTURE)
    {
        let texcol = textureSample(texture, texture_sampler, vertex.uv);
        out *= texcol;
    }

    @if(ALPHAMODE_OPAQUE)
    out.a = 1.0;

    @if(ALPHAMODE_MASK)
    {
        if out.a < m.alpha_cutoff {
            discard;
        }

        out.a = 1.0;
    }

    @if(ALPHAMODE_ALPHA_TO_COVERAGE)
    out.a = (out.a - 0.5) / max(fwidth(out.a), 0.0001) + 0.5;

    return out;
}
