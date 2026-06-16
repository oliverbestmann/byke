#module byke2d::view

struct View {
    screen_to_ndc: mat4x4<f32>,
    world_to_screen: mat4x4<f32>,
};
