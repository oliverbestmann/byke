struct Globals {
    time: f32,
    time_delta: f32,
    frame_index: u32,
    random: u32,
};

struct View {
    // in pixels, x, y, w, h
    viewport: vec4f,

    // projects from camera space to the screen
    camera_to_screen: mat4x4<f32>,
    camera_to_screen_inv: mat4x4<f32>,

    // from world to camera
    world_to_camera: mat4x4<f32>,
    world_to_camera_inv: mat4x4<f32>,

    // camera_projection * world_to_camera
    world_to_screen: mat4x4<f32>,
    world_to_screen_inv: mat4x4<f32>,

    // camera position in world space
    camera_position: vec3f,

    // the cameras exposure, used for pbr rendering
    exposure: f32,
};

@group(0)
@binding(0)
var<uniform> view: View;

@group(0)
@binding(1)
var<uniform> globals: Globals;
