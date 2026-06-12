#import byke2d::mesh3d

struct StandardMaterial {
    color: vec4f,
    emissive_scale: vec3f,
}

@group(2)
@binding(0)
var<uniform> material: StandardMaterial;

@group(2)
@binding(1)
var texture: texture_2d<f32>;

@group(2)
@binding(2)
var texture_sampler: sampler;

@group(2)
@binding(3)
var normalmap: texture_2d<f32>;

@group(2)
@binding(4)
var normalmap_sampler: sampler;

@group(2)
@binding(5)
var emissive: texture_2d<f32>;

@group(2)
@binding(6)
var emissive_sampler: sampler;

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out = default_mesh3d_vertex(in);
    out.color *= material.color;

    return out;
}

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    var out = default_mesh3d_fragment(vertex);

    #ifdef MESH3D_COLOR_HAS_TEXTURE
    let texcol = textureSample(texture, texture_sampler, vertex.uv);
    out *= texcol;
    out += texcol * vec4f(material.emissive_scale, 0.0);
    #endif

    #ifdef MESH3D_EMISSIVE_HAS_TEXTURE
    let emissive = textureSample(texture, texture_sampler, vertex.uv).rgb * material.emissive_scale;
    out += vec4f(emissive, 0.0);
    #endif

    return out;
}
