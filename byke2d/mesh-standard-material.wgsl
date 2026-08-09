import package::byke::globals::view;
import package::byke::mesh;
import package::byke::mesh::pbr;
import package::byke::lights;

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

@group(2)
@binding(9)
var metallic_roughness: texture_2d<f32>;

@group(2)
@binding(10)
var metallic_roughness_sampler: sampler;

@if(MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE)
fn calculate_normal(normal: vec3f, tangent: vec3f, tangent_sign: f32, uv: vec2f) -> vec3f {
    // normal from texture (in tangent space)
    let vNt = textureSample(normalmap, normalmap_sampler, uv).xyz * 2.0 - vec3f(1.0);;

    // calculate bi-tangent
    let bi_tangent = cross(normal, tangent) * tangent_sign;

    // calculate transformed normal
    return vNt.x * tangent + vNt.y * bi_tangent + vNt.z * normal;
}

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

    // transform the normal according to the normalmap
    @if(MESH3D_MAT_HAS_NORMAL && MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE)
    vertex.normal = calculate_normal(vertex.normal, vertex.tangent, vertex.tangent_sign, vertex.uv);

    if !front_facing && m.double_sided != 0 {
        // flip normal for double sided lighting
        vertex.normal = -vertex.normal;
    }

    @if(DEBUG_NORMALS)
    return vec4(normalize(vertex.normal) * 0.5 + 0.5, 1.0);

    // base color of the material comes from the vertex
    //  (material color was put there in the vertex shader)
    var base_color = vertex.color;

    // apply color texture to the color
    @if(MESH3D_MAT_HAS_TEXTURE)
    base_color *= textureSample(texture, texture_sampler, vertex.uv);

    // calculate lighting and update color accordingly
    var color = vec4f(
        calculate_lighting(vertex, base_color.rgb) * view.exposure,
        base_color.a,
    );

    @if(MESH3D_MAT_HAS_EMISSIVE)
    {
        // apply emissive light to the color
        let emissive_color = textureSample(emissive, emissive_sampler, vertex.uv).rgb;
        let emissive = emissive_color * m.emissive_scale;
        color += vec4f(emissive, 0.0);
    }

    // apply emissive scale to the base color itself
    @if(!MESH3D_MAT_HAS_EMISSIVE && MESH3D_MAT_HAS_TEXTURE)
    color += base_color * vec4f(m.emissive_scale, 0.0);

    @if(ALPHAMODE_OPAQUE)
    color.a = 1.0;

    @if(ALPHAMODE_MASK)
    {
        if color.a < m.alpha_cutoff {
            discard;
        }

        color.a = 1.0;
    }

    @if(ALPHAMODE_ALPHA_TO_COVERAGE)
    color.a = (color.a - 0.5) / max(fwidth(color.a), 0.0001) + 0.5;

    // DEBUG: paint all alpha blended meshes with half opacity
    //  to make them visible
    //
    //  @if(ALPHAMODE_BLEND)
    //  color.a *= 0.5;

    return color;
}

fn calculate_lighting(vertex: mesh::VertexOutput, base_color: vec3f) -> vec3f {
    var outcol = vec3f(0, 0, 0);

    // by default nothing is occluded
    var ambient_occlusion = 1.0;

    // module ambient occlusion from texture
    @if(MESH3D_MAT_HAS_OCCLUSION)
    ambient_occlusion *= textureSample(occlusion, occlusion_sampler, vertex.uv).r;

    @if(LIGHTING)
    {
        // TODO support logical expressions
        var normal = normalize(vertex.normal);

        // view direction and normal
        let V = normalize(view.camera_position - vertex.position_world);
        let N = normal;


        @if(!MESH3D_MAT_HAS_RMTEX)
        let metallic_scale: f32 = 1;

        @if(!MESH3D_MAT_HAS_RMTEX)
        let roughness_scale: f32 = 1;

        @if(MESH3D_MAT_HAS_RMTEX)
        let orm = textureSample(metallic_roughness, metallic_roughness_sampler, vertex.uv).rgb;

        @if(MESH3D_MAT_HAS_RMTEX)
        let metallic_scale = orm.b;

        @if(MESH3D_MAT_HAS_RMTEX)
        let roughness_scale = orm.g;

        let material = materials[vertex.material];
        let metallic = material.metallic * metallic_scale;
        let roughness = max(pbr::MIN_ROUGHNESS, material.perceptual_roughness * roughness_scale);

        // apply generic ambient light
        outcol += pbr::pbr_ambient_light(N, V, lights::light_config.ambient, base_color, metallic, roughness) * ambient_occlusion;

        // apply environment map to color
        @if(MESH_ENVMAP_LIGHT)
        outcol += pbr::sample_ibl(N, V, base_color, metallic, roughness)
            * pbr::pbr_env_options.intensity
            * ambient_occlusion;

        // apply point lights
        for (var i: u32 = 0; i < lights::point_lights.count; i++) {
            let light = lights::point_lights.lights[i];

            let to_light = light.position - vertex.position_world;
            let dist = length(to_light);

            if dist > 0.0001 {
                let L = to_light / dist;

                // Inverse-square falloff with a soft range fade.
                var attenuation = 1.0 / max(dist * dist, 0.01);

                if light.range > 0.0 {
                    let range_fade = 1.0 - smoothstep(0.8 * light.range, light.range, dist);
                    attenuation = attenuation * range_fade;
                }

                let radiance = light.color * attenuation;
                outcol += pbr::directPBR(N, V, L, radiance, base_color, metallic, roughness);
            }
        }

        // apply directional lights
        for (var i: u32 = 0; i < lights::directional_lights.count; i++) {
            let light = lights::directional_lights.lights[i];

            let L = light.direction;
            let radiance = light.color;
            outcol += pbr::directPBR(N, V, L, radiance, base_color, metallic, roughness);
        }

        // TODO spot lights
    }

    return outcol;
}
