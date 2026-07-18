#module byke2d::mesh2d

#import byke2d::view
#import byke2d::view::bindings

struct VertexInput {
    @builtin(vertex_index) index: u32,

    @location(0) i_affine_0: vec3f,
    @location(1) i_affine_1: vec3f,
    @location(2) i_affine_2: vec3f,
    @location(3) i_affine_3: vec3f,

    @location(4) i_material_index: u32,

    // vertex position from per-vertex buffer
    @location(5) v_position: vec3f,

#ifdef MESH2D_VERTEX_ATTRIBUTES_COLOR
    // vertex color from per-vertex buffer
    @location(MESH2D_VERTEX_ATTRIBUTES_COLOR) v_color: vec4f,
#endif

#ifdef MESH2D_VERTEX_ATTRIBUTES_UV
    // vertex color from per-vertex buffer
    @location(MESH2D_VERTEX_ATTRIBUTES_UV) v_uv: vec2f,
#endif
}

struct VertexOutput {
    @builtin(position) position: vec4f,
    @location(0) color: vec4f,
    @location(1) uv: vec2f,
};

fn default_mesh2d_vertex(in: VertexInput) -> VertexOutput {
    // transforms the four column vectors back to a full 4x4 matrix by adding the last row.
    let model_to_world = mat4x4f(
        vec4f(in.i_affine_0, 0),
        vec4f(in.i_affine_1, 0),
        vec4f(in.i_affine_2, 0),
        vec4f(in.i_affine_3, 1),
    );

    let position = view.world_to_screen
        * model_to_world
        * vec4f(in.v_position, 1.0);

    // move the vertex to the world
    var out: VertexOutput;
    out.position = position;
    out.color = vec4f(1.0, 1.0, 1.0, 1.0);

#ifdef MESH2D_VERTEX_ATTRIBUTES_COLOR
    // need to add 1 to the vertex color to convert from byke2d.Color
    let v_color = in.v_color + vec4f(1, 1, 1, 1);
    out.color *= v_color;
#endif

#ifdef MESH2D_VERTEX_ATTRIBUTES_UV
    out.uv = in.v_uv;
#endif

    return out;
}

fn default_mesh2d_fragment(vertex: VertexOutput) -> vec4f {
    return vertex.color;
}

