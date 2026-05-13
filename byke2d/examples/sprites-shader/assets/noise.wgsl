#import byke2d::sprite

fn hash12(pos: vec2f) -> f32 {
    return fract(sin(dot(pos, vec2(12.9898, 78.233))) * 43758.5453);
}

@fragment
fn noisy_sprite(vo: VertexOutput) -> @location(0) vec4f {
    let org = default_sprite_fragment(vo);

    let h0 = hash12(vo.uv);

    let offset = vec3(
        hash12(vo.uv),
        hash12(vo.uv + vec2(h0, 0)),
        hash12(vo.uv + vec2(0, h0)),
    );

    let rgb_offset = vec4((offset - 0.5) * 0.2, 0.0);

    return org + rgb_offset;
}
