#module byke2d::view

struct View {
    screen_to_ndc: mat3x3<f32>,
    world_to_screen: mat3x3<f32>,
};
