#module byke2d::mesh::morph


struct MorphDescriptor {
    // number of targets in the current mesh. This is equal
    //  to the number of weights per vertex
    target_count: u32,

    // number of vertices in the current mesh
    vertex_count: u32,

    // index into the morph_weights buffer
    weights_index: u32,

    // the index into the morphs attributes
    first_attributes_index: u32,
}

struct MorphAttributes {
    position: vec3f,
    normal: vec3f,
    tangent: vec3f,
}

// The list of morph infos for all meshes.
@group(0)
@binding(20)
var<storage> morph_descriptors: array<MorphDescriptor>;

// The morph weights for all meshes
@group(0)
@binding(21)
var <storage> morph_weights: array<f32>;

// The morph attributes for the current mesh. Must contain at least
// one entry per morph target per vertex.
@group(1)
@binding(0)
var <storage> morph_attributes: array<MorphAttributes>;

fn morph_position(pos: vec3f, morph_info_index: u32, vertex_index: u32) -> vec3f {
    var result: vec3f = pos;

    let info = morph_descriptors[morph_info_index];

    for (var ta: u32 = 0; ta < info.target_count; ta++) {
        let idx = ta * info.vertex_count + vertex_index;

        let attrs = morph_attributes[info.first_attributes_index + idx];
        let weight = morph_weights[info.weights_index + ta];

        result += attrs.position * weight;
    }

    return result;
}
