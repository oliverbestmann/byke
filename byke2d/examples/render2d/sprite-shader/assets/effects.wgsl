import package::byke::sprite;
import package::byke::globals;

@vertex
fn vs_main(in: sprite::VertexInput) -> sprite::VertexOutput {
    return sprite::default_sprite_vertex(in);
}

fn hash(p: vec2f) -> f32 {
    return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453);
}

// smooth value noise between 0 and 1
fn noise(p: vec2f) -> f32 {
    let i = floor(p);
    let f = fract(p);
    let u = f * f * (3.0 - 2.0 * f);

    let a = hash(i);
    let b = hash(i + vec2(1.0, 0.0));
    let c = hash(i + vec2(0.0, 1.0));
    let d = hash(i + vec2(1.0, 1.0));

    return mix(mix(a, b, u.x), mix(c, d, u.x), u.y);
}

// sums up noise of increasing frequency for more detail
fn fbm(p: vec2f) -> f32 {
    var value = 0.0;
    var amplitude = 0.5;
    var q = p;

    for (var i = 0; i < 5; i++) {
        value += amplitude * noise(q);
        q *= 2.0;
        amplitude *= 0.5;
    }

    return value;
}

// A procedural fire, the texture is not used.
@fragment
fn fire(vo: sprite::VertexOutput) -> @location(0) vec4f {
    let time = globals::globals.time;

    // 0 at the bottom, 1 at the top
    let height = 1.0 - vo.uv.y;

    // distance from the vertical center line
    let dx = abs(vo.uv.x - 0.5) * 2.0;

    // noise that moves upwards
    let n = fbm(vec2(vo.uv.x * 5.0, vo.uv.y * 4.0 + time * 2.5));

    // the flame gets thinner towards the top and the sides
    let v = clamp((n - height * 0.8 - dx * dx * 0.7) * 3.0, 0.0, 1.0);

    // red to yellow, white only in the hottest parts
    let color = vec3(1.4 * v, 1.1 * v * v, 0.8 * v * v * v * v);

    // fade in at the bottom edge
    let alpha = clamp(v * 2.0, 0.0, 1.0) * smoothstep(0.0, 0.1, height);

    return vec4(min(color, vec3(1.0)), alpha);
}

// Moves the texture in waves, like a flag in the wind.
@fragment
fn wave(vo: sprite::VertexOutput) -> @location(0) vec4f {
    let time = globals::globals.time;
    let phase = vo.uv.x * 10.0 - time * 4.0;

    // leave some room at the borders for the wave to move into
    var shifted = vo;
    shifted.uv = (vo.uv - 0.5) * 1.15 + 0.5 + vec2(0.0, sin(phase) * 0.05);

    let color = sprite::default_sprite_fragment(shifted);
    let inside = all(shifted.uv >= vec2(0.0)) && all(shifted.uv <= vec2(1.0));

    // darken the parts of the wave that face away
    let shade = 1.0 + cos(phase) * 0.2;

    return select(vec4(0.0), vec4(color.rgb * shade, color.a), inside);
}

// Burns the texture away along a noise pattern and lets it grow back.
@fragment
fn dissolve(vo: sprite::VertexOutput) -> @location(0) vec4f {
    let time = globals::globals.time;
    // starts fully visible, burns away to 80% and grows back
    let threshold = (1.0 - cos(time * 2.0)) * 0.4;

    let color = sprite::default_sprite_fragment(vo);
    let n = fbm(vo.uv * 6.0);

    // pixels below the threshold are gone, just above it they glow
    let visible = step(threshold, n);
    let glow = (1.0 - smoothstep(threshold, threshold + 0.08, n)) * visible;

    let rgb = mix(color.rgb, vec3(1.0, 0.55, 0.1), glow);
    return vec4(rgb, color.a * visible);
}
