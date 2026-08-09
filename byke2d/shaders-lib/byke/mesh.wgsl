import package::byke::globals::view;

@if(LIGHTING)
import package::byke::lights;

import package::byke::mesh::morph;

struct VertexInput {
    @builtin(vertex_index)
    index: u32,

    // truncated columns of the affine transform matrix
    @location(0)
    i_affine_0: vec3f,
    @location(1)
    i_affine_1: vec3f,
    @location(2)
    i_affine_2: vec3f,
    @location(3)
    i_affine_3: vec3f,

    // base index in the vertex buffer. we need this to offset index
    @location(4)
    i_base_vertex: u32,

    // index into materials array
    @location(5)
    i_material_index: u32,

    // index in morph info buffer
    @location(6)
    i_morph_index: u32,

    // vertex position from per-vertex buffer
    @location(7)
    v_position: vec3f,

    // vertex color from per-vertex buffer
    @if(MESH3D_VERTEX_ATTRIBUTES_COLOR)
    @location(8)
    v_color: vec4f,

    // vertex color from per-vertex buffer
    @if(MESH3D_VERTEX_ATTRIBUTES_NORMAL)
    @location(12)
    v_normal: vec3f,

    // tangent space from per-vertex buffer
    @if(MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE)
    @location(13)
    v_tangent_space: vec4f,

    // vertex color from per-vertex buffer
    @if(MESH3D_VERTEX_ATTRIBUTES_UV)
    @location(9)
    v_uv: vec2f,

    // vertex color from per-vertex buffer
    @if(SKINNED)
    @location(14)
    v_joint: vec4u,

    @if(SKINNED)
    @location(15)
    v_joint_weights: vec4f,
}

struct VertexOutput {
    @builtin(position)
    position: vec4f,
    @location(0)
    color: vec4f,
    @location(1)
    position_world: vec3f,
    @location(2)
    normal: vec3f,
    @location(3)
    tangent: vec3f,
    @location(4) @interpolate(flat)
    tangent_sign: f32,
    @location(5)
    uv: vec2f,

    // index into materials array (if any)
    @location(6) @interpolate(flat)
    material: u32,
};

// Size of the array must match the maxJoints constant
@if(SKINNED)
@group(0)
@binding(30)
var<uniform> joints: array<mat4x4f, 256>;

// TODO move to math
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

@if(SKINNED)
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

fn process_vertex(in: VertexInput) -> VertexOutput {
    // interpolate joint matrices
    @if(SKINNED)
    let world_from_local = in.v_joint_weights.x * joints[in.v_joint.x] +
        in.v_joint_weights.y * joints[in.v_joint.y] +
        in.v_joint_weights.z * joints[in.v_joint.z] +
        in.v_joint_weights.w * joints[in.v_joint.w];

    // transforms the four column vectors back to a full 4x4 matrix by adding the last row.
    @if(!SKINNED)
    let world_from_local = mat4x4f(
        vec4f(in.i_affine_0, 0),
        vec4f(in.i_affine_1, 0),
        vec4f(in.i_affine_2, 0),
        vec4f(in.i_affine_3, 1),
    );

    var position_local = in.v_position;

    var vertex_index = in.index - in.i_base_vertex;

    // morph the position of the position vector before skinning
    @if(MORPH)
    position_local = morph::morph_position(position_local, in.i_morph_index, vertex_index);

    let position_world = world_from_local * vec4f(position_local, 1.0);

    let position = view.world_to_screen * position_world;

    // move the vertex to the world
    var out: VertexOutput;
    out.position = position;
    out.position_world = position_world.xyz;
    out.color = vec4f(1.0, 1.0, 1.0, 1.0);
    out.material = in.i_material_index;

    @if(MESH3D_VERTEX_ATTRIBUTES_COLOR)
    {
        // need to add 1 to the vertex color to convert from byke2d.Color
        let v_color = in.v_color + vec4f(1, 1, 1, 1);
        out.color *= v_color;
    }

    // upper left of the model matrix
    let world_from_local_normal = mat3x3(
        world_from_local[0].xyz,
        world_from_local[1].xyz,
        world_from_local[2].xyz,
    );

    @if(MESH3D_VERTEX_ATTRIBUTES_NORMAL)
    {
        @if(SKINNED)
        out.normal = skin_normals(world_from_local, in.v_normal);

        @if(!SKINNED)
        // mikktspace: normalize in fragment shader
        out.normal = world_from_local_normal * in.v_normal;
    }

    @if(MESH3D_VERTEX_ATTRIBUTES_TANGENTSPACE)
    {
        // mikktspace: normalize in fragment shader
        out.tangent = world_from_local_normal * in.v_tangent_space.xyz;
        out.tangent_sign = in.v_tangent_space.w;
    }

    @if(MESH3D_VERTEX_ATTRIBUTES_UV)
    out.uv = in.v_uv;

    return out;
}

fn default_mesh3d_fragment(vertex: VertexOutput) -> vec4f {
    return vertex.color;
}
