#module byke2d::mesh2d

#import byke2d::view
#import byke2d::view::binding

struct VertexInput {
    @builtin(vertex_index) index: u32,

    @location(0) i_affine_0: vec3f,
    @location(1) i_affine_1: vec3f,
    @location(2) i_affine_2: vec3f,
    @location(3) i_affine_3: vec3f,

    @location(4) i_color: vec4f,

    // vertex position from per-vertex buffer
    @location(5) v_position: vec3f,

#ifdef MESH2D_VERTEX_ATTRIBUTES_COLOR
    // vertex color from per-vertex buffer
    @location(MESH2D_VERTEX_ATTRIBUTES_COLOR) v_color: vec4f,
#endif
}

struct VertexOutput {
    @builtin(position) position: vec4f,
    @location(0) color: vec4f,
};

fn default_mesh2d_vertex(in: VertexInput) -> VertexOutput {
    // transforms the four column vectors back to a full 4x4 matrix by adding the last row.
    let model_to_world = mat4x4f(
        vec4f(in.i_affine_0, 0),
        vec4f(in.i_affine_1, 0),
        vec4f(in.i_affine_2, 0),
        vec4f(in.i_affine_3, 1),
    );

    let position = view.screen_to_ndc
        * view.world_to_screen
        * model_to_world
        * vec4f(in.v_position, 1.0);

    // move the vertex to the world
    var out: VertexOutput;
    out.position = vec4(position.xyz, 1.0);
    out.color = in.i_color;

#ifdef MESH2D_VERTEX_ATTRIBUTES_COLOR
    // need to add 1 to the vertex color to convert from byke2d.Color
    let v_color = in.v_color + vec4f(1, 1, 1, 1);
    out.color *= v_color;
#endif

    return out;
}

fn default_mesh2d_fragment(vertex: VertexOutput) -> vec4f {
    return vertex.color;
}

@vertex
fn vs_main(in: VertexInput) -> VertexOutput {
    return default_mesh2d_vertex(in);
}

@fragment
fn fs_main(vertex: VertexOutput) -> @location(0) vec4f {
    return default_mesh2d_fragment(vertex);
}
