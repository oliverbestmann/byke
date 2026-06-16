#module byke2d::lights::bindings

#import byke2d::lights

@group(0)
@binding(10)
var<uniform> light_config: LightConfig;

@group(0)
@binding(11)
var<storage> directional_lights: DirectionalLights;

@group(0)
@binding(12)
var<storage> point_lights: PointLights;

@group(0)
@binding(13)
var<storage> spot_lights: SpotLights;
