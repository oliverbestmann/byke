#import byke2d::mesh3d

struct StandardMaterial {
    color: vec4f,
    emissive_scale: vec3f,
    double_sided: u32,
    alpha_cutoff: f32,
    metallic: f32,
    perceptual_roughness: f32,
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

#ifdef MESH3D_MAT_HAS_NORMAL
    #ifdef MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE
        // transform the normal according to the normalmap
        vertex.normal = calculate_normal(vertex.normal, vertex.tangent, vertex.tangent_sign, vertex.uv);
    #endif
#endif

    if ! front_facing && m.double_sided != 0 {
        // flip normal for double sided lighting
        vertex.normal = -vertex.normal;
    }

#ifdef DEBUG_NORMALS
    return vec4(normalize(vertex.normal) * 0.5 + 0.5, 1.0);
#endif

    // base color of the material comes from the vertex
    //  (material color was put there in the vertex shader)
    var color = vertex.color;

#ifdef MESH3D_MAT_HAS_TEXTURE
    // apply texture to the color
    let texcol = textureSample(texture, texture_sampler, vertex.uv);
    color *= texcol;
    color += texcol * vec4f(m.emissive_scale, 0.0);
#endif

#ifdef MESH3D_MAT_HAS_EMISSIVE
    // apply emissive light to the color
    let emissive_color = textureSample(emissive, emissive_sampler, vertex.uv).rgb;
    let emissive = emissive_color * m.emissive_scale;
    color += vec4f(emissive, 0.0);
#endif

    // calculate lighting and update color accordingly
    let light = calculate_lighting(vertex, color.rgb);
    color = vec4f(light, color.a);

#ifdef ALPHAMODE_OPAQUE
    color.a = 1.0;
#endif

#ifdef ALPHAMODE_MASK
    if color.a < m.alpha_cutoff {
        discard;
    }

    color.a = 1.0;
#endif

#ifdef ALPHAMODE_ALPHA_TO_COVERAGE
    color.a = (color.a - 0.5) / max(fwidth(color.a), 0.0001) + 0.5;
#endif

    return color;
}

fn calculate_lighting(vertex: VertexOutput, base_color: vec3f) -> vec3f {
    var tint = vec3f(0, 0, 0);

    // by default nothing is occluded
    var ambient_occlusion = 1.0;

#ifdef MESH3D_MAT_HAS_OCCLUSION
    // module ambient occlusion from texture
    ambient_occlusion *= textureSample(occlusion, occlusion_sampler, vertex.uv).r;
#endif

    // apply generic ambient light
    tint += light_config.ambient * ambient_occlusion;

    // TODO support logical expressions
#ifdef LIGHTING
#ifdef MESH3D_VERTEX_ATTRIBUTES_NORMAL
    var normal = normalize(vertex.normal);

    // view direction and normal
    let V = normalize(view.camera_position - vertex.position_world);
    let N = normal;


    let material = materials[vertex.material];
    let metallic = material.metallic;
    let roughness = max(MIN_ROUGHNESS, material.perceptual_roughness);

#ifdef MESH_ENVMAP_LIGHT
    // apply environment map to color
    tint += sampleIBL(N, V, base_color, metallic, roughness)
        // * pbr_env_options.intensity
        * ambient_occlusion;
#endif

    // apply point lights
    for (var i: u32 = 0; i < point_lights.count; i++) {
        let light = point_lights.lights[i];

        let to_light = light.position - vertex.position_world;
        let dist = length(to_light);

        if (dist > 0.0001) {
          let L = to_light / dist;

          // Inverse-square falloff with a soft range fade.
          var attenuation = 1.0 / max(dist * dist, 0.01);

          // TODO add range?
          // if (light.range > 0.0) {
          //   let range_fade = 1.0 - smoothstep(0.8 * light.range, light.range, dist);
          //   attenuation = attenuation * range_fade;
          // }

          let radiance = light.color * attenuation;
          tint += directPBR(N, V, L, radiance, base_color, metallic, roughness);
        }
    }

    // TODO spot lights

#endif
#endif

    return tint;
}

