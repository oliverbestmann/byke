import package::byke::mesh;

// Depth only pass for shadow maps. The view uniforms of a shadow view
// hold the projection of the light, so the regular mesh vertex processing works.
@vertex
fn vs_main(in: mesh::VertexInput) -> mesh::VertexOutput {
    return mesh::process_vertex(in);
}
