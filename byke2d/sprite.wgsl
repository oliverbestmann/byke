import package::byke::sprite;

@vertex
fn vs_main(in: sprite::VertexInput) -> sprite::VertexOutput {
    return sprite::default_sprite_vertex(in);
}

@fragment
fn fs_main(vertex: sprite::VertexOutput) -> @location(0) vec4f {
    return sprite::default_sprite_fragment(vertex);
}
