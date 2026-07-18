#import byke2d::mesh3d

struct StandardMaterial {
    color: vec4f,
    emissive_scale: vec3f,
    double_sided: u32,
    alpha_cutoff: f32,
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

@group(2)
@binding(7)
var occlusion: texture_2d<f32>;

@group(2)
@binding(8)
var occlusion_sampler: sampler;

#ifdef MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE
fn calculate_normal(normal: vec3f, tangent: vec3f, tangent_sign: f32, uv: vec2f) -> vec3f {
    // normal from texture (in tangent space)
    let vNt = textureSample(normalmap, normalmap_sampler, uv).xyz * 2.0 - vec3f(1.0);;

    // calculate bi-tangent
    let bi_tangent = cross(normal, tangent) * tangent_sign;

    // calculate transformed normal
    return vNt.x * tangent + vNt.y * bi_tangent + vNt.z * normal;
}
#endif

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

#ifdef MESH3D_MAT_HAS_OCCLUSION
    // get ambient occlusion
    fin.ambient_occlusion = textureSample(occlusion, occlusion_sampler, vertex.uv).r;
#endif

#ifdef MESH3D_MAT_HAS_NORMAL
    #ifdef MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE
        vertex.normal = calculate_normal(vertex.normal, vertex.tangent, vertex.tangent_sign, vertex.uv);
    #endif
#endif

    if ! front_facing && m.double_sided != 0 {
        // flip normal for double sided lighting
        vertex.normal = -vertex.normal;
    }

    var out = default_mesh3d_fragment(vertex, fin);

#ifdef MESH3D_MAT_HAS_TEXTURE
    let texcol = textureSample(texture, texture_sampler, vertex.uv);
    out *= texcol;
    out += texcol * vec4f(m.emissive_scale, 0.0);
#endif

#ifdef MESH3D_MAT_HAS_EMISSIVE
    let emissive_color = textureSample(emissive, emissive_sampler, vertex.uv).rgb;
    let emissive = emissive_color * m.emissive_scale;
    out += vec4f(emissive, 0.0);
#endif

#ifdef ALPHAMODE_OPAQUE
    out.a = 1.0;
#endif

#ifdef ALPHAMODE_MASK
    if out.a < m.alpha_cutoff {
        discard;
    }

    out.a = 1.0;
#endif

#ifdef ALPHAMODE_ALPHA_TO_COVERAGE
    out.a = (out.a - 0.5) / max(fwidth(out.a), 0.0001) + 0.5;
#endif

    return out;
}


