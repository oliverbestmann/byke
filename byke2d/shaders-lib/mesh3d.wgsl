#module byke2d::mesh3d

#import byke2d::view
#import byke2d::view::bindings

#import byke2d::lights
#import byke2d::lights::bindings

struct VertexInput {
    @builtin(vertex_index) index: u32,

    // truncated columns of the affine transform matrix
    @location(0) i_affine_0: vec3f,
    @location(1) i_affine_1: vec3f,
    @location(2) i_affine_2: vec3f,
    @location(3) i_affine_3: vec3f,

    // index in morph info buffer
    @location(4) i_morph_index: u32,

    // vertex position from per-vertex buffer
    @location(9) v_position: vec3f,

#ifdef MESH3D_VERTEX_ATTRIBUTES_COLOR
    // vertex color from per-vertex buffer
    @location(MESH3D_VERTEX_ATTRIBUTES_COLOR) v_color: vec4f,
#endif

#ifdef MESH3D_VERTEX_ATTRIBUTES_NORMAL
    // vertex color from per-vertex buffer
    @location(MESH3D_VERTEX_ATTRIBUTES_NORMAL) v_normal: vec3f,
#endif

#ifdef MESH3D_VERTEX_ATTRIBUTES_UV
    // vertex color from per-vertex buffer
    @location(MESH3D_VERTEX_ATTRIBUTES_UV) v_uv: vec2f,
#endif

#ifdef SKINNED
    // vertex color from per-vertex buffer
    @location(MESH3D_VERTEX_ATTRIBUTES_JOINTS) v_joint: vec4u,
    @location(MESH3D_VERTEX_ATTRIBUTES_JOINTWEIGHTS) v_joint_weights: vec4f,
#endif
}

struct VertexOutput {
    @builtin(position) position: vec4f,
    @location(0) color: vec4f,
    @location(1) position_world: vec3f,
    @location(2) normal: vec3f,
    @location(3) uv: vec2f,
};


#ifdef SKINNED

// Size of the array must match the maxJoints constant
@group(0)
@binding(30)
var<uniform> joints: array<mat4x4f, 256>;

fn inverse_transpose_3x3m(in: mat3x3<f32>) -> mat3x3<f32> {
    let x = cross(in[1], in[2]);
    let y = cross(in[2], in[0]);
    let z = cross(in[0], in[1]);
    let det = dot(in[2], z);
    return mat3x3<f32>(
        x / det,
        y / det,
        z / det
    );
}

fn skin_normals(
    world_from_local: mat4x4<f32>,
    normal: vec3<f32>,
) -> vec3<f32> {
    return normalize(
        inverse_transpose_3x3m(
            mat3x3<f32>(
                world_from_local[0].xyz,
                world_from_local[1].xyz,
                world_from_local[2].xyz
            )
        ) * normal
    );
}

#endif


#ifdef MORPH
#import byke2d::mesh::morph
#endif

fn default_mesh3d_vertex(in: VertexInput) -> VertexOutput {
#ifdef SKINNED
    // interpolate joint matrices
    let world_from_local =
        in.v_joint_weights.x * joints[in.v_joint.x]+
        in.v_joint_weights.y * joints[in.v_joint.y]+
        in.v_joint_weights.z * joints[in.v_joint.z]+
        in.v_joint_weights.w * joints[in.v_joint.w];
#else
    // transforms the four column vectors back to a full 4x4 matrix by adding the last row.
    let world_from_local = mat4x4f(
        vec4f(in.i_affine_0, 0),
        vec4f(in.i_affine_1, 0),
        vec4f(in.i_affine_2, 0),
        vec4f(in.i_affine_3, 1),
    );
#endif

    var position_local = in.v_position;

#ifdef MORPH
    // morph the position of the position vector before skinning
    position_local = morph_position(position_local, in.i_morph_index, in.index);
#endif

    let position_world = world_from_local * vec4f(position_local, 1.0);

    let position = view.screen_to_ndc
        * view.world_to_screen
        * position_world;

    // move the vertex to the world
    var out: VertexOutput;
    out.position = position;
    out.position_world = position_world.xyz;
    out.color = vec4f(1.0, 1.0, 1.0, 1.0);

#ifdef MESH3D_VERTEX_ATTRIBUTES_COLOR
    // need to add 1 to the vertex color to convert from byke2d.Color
    let v_color = in.v_color + vec4f(1, 1, 1, 1);
    out.color *= v_color;
#endif

#ifdef MESH3D_VERTEX_ATTRIBUTES_NORMAL
    #ifdef SKINNED
        out.normal = skin_normals(world_from_local, in.v_normal);
    #else
        let world_from_local_normal = mat3x3(
            world_from_local[0].xyz,
            world_from_local[1].xyz,
            world_from_local[2].xyz,
        );

        out.normal = world_from_local_normal * in.v_normal;
    #endif
#endif

#ifdef MESH3D_VERTEX_ATTRIBUTES_UV
    out.uv = in.v_uv;
#endif

    return out;
}

fn default_mesh3d_fragment(vertex: VertexOutput) -> vec4f {
    var color = vertex.color;

#ifdef MESH3D_VERTEX_ATTRIBUTES_NORMAL
    var tint = light_config.ambient;

    // apply directional lights
    for (var i: u32 = 0; i < directional_lights.count; i++) {
        let light = directional_lights.lights[i];

        let l = normalize(light.direction);
        let n = normalize(vertex.normal);
        let n_dot_l = max(dot(n, l), 0.0);

        tint += light.color.rgb * n_dot_l;
    }

    // apply point lights
    for (var i: u32 = 0; i < point_lights.count; i++) {
        let light = point_lights.lights[i];

        let light_vec = light.position - vertex.position_world;
        let distance = length(light_vec);
        let l = normalize(light_vec);
        let n = normalize(vertex.normal);
        let n_dot_l = max(dot(n, l), 0.0);

        let attenuation =
            1.0 /
            (light.att_constant +
             light.att_linear * distance +
             light.att_quadratic * distance * distance);

        tint += light.color.rgb * attenuation * n_dot_l;
    }

    // apply spot lights
    // TODO

    // apply light
    color = vec4f(color.rgb * tint, color.a);
#endif

    return color;
}

