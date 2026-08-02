#module byke2d::pbr

#import byke2d::math

const MIN_ROUGHNESS: f32 = 0.045;

struct EnvironmentMap {
    intensity: f32,
    // specular_max_miplevel: u32,
};

#ifdef MESH_ENVMAP_LIGHT

@group(0)
@binding(40)
var<uniform> pbr_env_options: EnvironmentMap;

@group(0)
@binding(41)
var pbr_env_map_diffuse: texture_cube<f32>;

@group(0)
@binding(42)
var pbr_env_map_specular: texture_cube<f32>;

@group(0)
@binding(43)
var pbr_brdf_lookup: texture_2d<f32>;

@group(0)
@binding(44)
var pbr_env_sampler: sampler;

#endif

fn fresnelSchlick(cosTheta: f32, F0: vec3<f32>) -> vec3<f32> {
    return F0 + (vec3<f32>(1.0) - F0) * pow5(1.0 - cosTheta);
}

fn distributionGGX(N: vec3<f32>, H: vec3<f32>, roughness: f32) -> f32 {
    let a = roughness * roughness;
    let a2 = a * a;
    let NdotH = saturate(dot(N, H));
    let NdotH2 = NdotH * NdotH;

    let denom = (NdotH2 * (a2 - 1.0) + 1.0);
    return a2 / max(PI * denom * denom, 0.000001);
}

fn geometrySchlickGGX(NdotV: f32, roughness: f32) -> f32 {
    let r = roughness + 1.0;
    let k = (r * r) / 8.0; // Common direct-lighting approximation
    return NdotV / max(NdotV * (1.0 - k) + k, 0.000001);
}

fn geometrySmith(N: vec3<f32>, V: vec3<f32>, L: vec3<f32>, roughness: f32) -> f32 {
    let NdotV = saturate(dot(N, V));
    let NdotL = saturate(dot(N, L));
    let ggx1 = geometrySchlickGGX(NdotV, roughness);
    let ggx2 = geometrySchlickGGX(NdotL, roughness);
    return ggx1 * ggx2;
}

fn directPBR(
    N: vec3<f32>,
    V: vec3<f32>,
    L: vec3<f32>,
    radiance: vec3<f32>,
    baseColor: vec3<f32>,
    metallic: f32,
    roughness: f32
) -> vec3<f32> {
    let H = normalize(V + L);

    let NdotL = saturate(dot(N, L));
    let NdotV = saturate(dot(N, V));
    let VdotH = saturate(dot(V, H));

    if (NdotL <= 0.0 || NdotV <= 0.0) {
        return vec3<f32>(0.0);
    }

    let F0 = mix(vec3<f32>(0.04), baseColor, vec3<f32>(metallic));
    let F = fresnelSchlick(VdotH, F0);
    let NDF = distributionGGX(N, H, roughness);
    let G = geometrySmith(N, V, L, roughness);

    let numerator = NDF * G * F;
    let denominator = max(4.0 * NdotV * NdotL, 0.001);
    let specular = numerator / denominator;

    let kS = F;
    let kD = (vec3<f32>(1.0) - kS) * (1.0 - metallic);

    let diffuse = kD * baseColor / PI;

    return (diffuse + specular) * radiance * NdotL;
}

#ifdef MESH_ENVMAP_LIGHT

fn sample_ibl(
    N: vec3<f32>,
    V: vec3<f32>,
    baseColor: vec3<f32>,
    metallic: f32,
    roughness: f32,
) -> vec3<f32> {
    let NdotV = saturate(dot(N, V));
    let R = reflect(-V, N);

    let F0 = mix(vec3<f32>(0.04), baseColor, vec3<f32>(metallic));
    let F = fresnelSchlick(NdotV, F0);

    let irradiance = textureSampleLevel(pbr_env_map_diffuse, pbr_env_sampler, N, 0.0).rgb;
    let diffuse_ibl = irradiance * baseColor;

    let max_mip_level = f32(textureNumLevels(pbr_env_map_specular) - 1);
    let mip_level = max_mip_level * roughness;
    let prefiltered_color = textureSampleLevel(pbr_env_map_specular, pbr_env_sampler, R, mip_level).rgb;

    let brdf = F_AB(roughness, NdotV);

    // Split-sum approximation
    let specular_ibl = prefiltered_color * (F0 * brdf.x + vec3<f32>(brdf.y));

    let kS = F;
    let kD = (vec3<f32>(1.0) - kS) * (1.0 - metallic);

    return (kD * diffuse_ibl + specular_ibl);
}

#endif

fn pbr_ambient_light(
    N: vec3<f32>,
    V: vec3<f32>,
    ambient: vec3<f32>,
    baseColor: vec3<f32>,
    metallic: f32,
    roughness: f32,
) -> vec3<f32> {
    let NdotV = saturate(dot(N, V));
    let R = reflect(-V, N);

    let F0 = mix(vec3<f32>(0.04), baseColor, vec3<f32>(metallic));
    let F = fresnelSchlick(NdotV, F0);

    let brdf = F_AB(roughness, NdotV);

    // Split-sum approximation
    let specular_ibl = ambient * (F0 * brdf.x + vec3<f32>(brdf.y));

    let kS = F;
    let kD = (vec3<f32>(1.0) - kS) * (1.0 - metallic);

    return (kD * (ambient * baseColor) + specular_ibl);
}

// Scale/bias approximation
fn F_AB(perceptual_roughness: f32, NdotV: f32) -> vec2<f32> {
#ifdef MESH_ENVMAP_LIGHT
    return textureSampleLevel(pbr_brdf_lookup, pbr_env_sampler, vec2<f32>(NdotV, perceptual_roughness), 0.0).rg;
#else
    // Polynomial approximation, see https://www.unrealengine.com/en-US/blog/physically-based-shading-on-mobile
    let c0 = vec4<f32>(-1.0, -0.0275, -0.572, 0.022);
    let c1 = vec4<f32>(1.0, 0.0425, 1.04, -0.04);
    let r = perceptual_roughness * c0 + c1;
    let a004 = min(r.x * r.x, exp2(-9.28 * NdotV)) * r.x + r.y;
    // Keep F_ab positive to avoid divide-by-zero in downstream BRDF terms.
    let f_ab_epsilon = 0.00005;
    return max(vec2<f32>(-1.04, 1.04) * a004 + r.zw, vec2<f32>(f_ab_epsilon));
#endif
}