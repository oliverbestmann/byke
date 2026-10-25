import package::byke::sprite;

@vertex
fn vs_main(in: sprite::VertexInput) -> sprite::VertexOutput {
    return sprite::default_sprite_vertex(in);
}

@fragment
fn fade_sprite(vo: sprite::VertexOutput) -> @location(0) vec4f {
    let org = sprite::default_sprite_fragment(vo);

    let uv0 = vo.uv * 2 - 1;

    let fade = 1 - dot(uv0, uv0);

    return vec4(org.rgb, fade);
}
