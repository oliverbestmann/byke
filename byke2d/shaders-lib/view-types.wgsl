#module byke2d::view

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
};
