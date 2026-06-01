#import byke2d::mesh3d

struct StandardMaterial {
    color: vec4f,
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
    out *= textureSample(texture, texture_sampler, vertex.uv);
    #endif

    let light_pos = vec3f(100, 100, -100);

    // apply light to pixel, keep alpha
    let col = dot(normalize(light_pos - vertex.position_world), vertex.normal);
    out = vec4f(out.rgb * (0.25 + 0.75 * col), out.a);

    return out;
}
