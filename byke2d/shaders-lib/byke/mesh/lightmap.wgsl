
// The lightmap for this mesh
@group(1)
@binding(1)
var lightmap_texture: texture_2d<f32>;

// The lightmap sampler for this mesh
@group(1)
@binding(2)
var lightmap_sampler: sampler;

fn lightmap_sample(uv: vec2f, exposure: f32) -> vec3f {
    let color = textureSample(lightmap_texture, lightmap_sampler, uv);
    return color.rgb * exposure;
}
