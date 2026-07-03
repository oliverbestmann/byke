#import byke2d::mesh3d

struct StandardMaterial {
    color: vec4f,
    emissive_scale: vec3f,
}

@group(2)
@binding(0)
var<storage, read> materials: array<StandardMaterial>;

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

#ifdef MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE
fn calculate_normal(normal: vec3f, tangent_space: vec4f, uv: vec2f) -> vec3f {
    // normal from texture
    let vNt = textureSample(normalmap, normalmap_sampler, uv);

    // decode tangent space
    let sign = tangent_space.w;
    let tangent = tangent_space.xyz;

    // calculate bi-tangent
    let bi_tangent = cross(normal, tangent) * sign;

    // calculate transformed normal
    return normalize(vNt.x * tangent + vNt.y * bi_tangent + vNt.z * normal);
}
#endif

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    var out = default_mesh3d_vertex(in);
    out.color *= materials[out.material].color;
    return out;
}

@fragment
fn fs_main(param: VertexOutput) -> @location(0) vec4f {

#ifdef MESH3D_COLOR_HAS_NORMAL
    var vertex = param;
    vertex.normal = calculate_normal(vertex.normal, vertex.tangent_space, vertex.uv);
#else
    let vertex = param;
#endif

    var out = default_mesh3d_fragment(vertex);

#ifdef MESH3D_COLOR_HAS_TEXTURE
    let texcol = textureSample(texture, texture_sampler, vertex.uv);
    out *= texcol;
    out += texcol * vec4f(materials[vertex.material].emissive_scale, 0.0);
#endif

#ifdef MESH3D_COLOR_HAS_EMISSIVE@interpolate(flat)
    let emissive = textureSample(texture, texture_sampler, vertex.uv).rgb * materials[vertex.material].emissive_scale;
    out += vec4f(emissive, 0.0);
#endif

    return out;
}


