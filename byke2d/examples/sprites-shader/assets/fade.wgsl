#import byke2d::sprite

@fragment
fn fade_sprite(vo: VertexOutput) -> @location(0) vec4f {
    let org = default_sprite_fragment(vo);

    let uv0 = vo.uv * 2 - 1;

    let fade = 1 - dot(uv0, uv0);

    return vec4(org.rgb, fade);
}
