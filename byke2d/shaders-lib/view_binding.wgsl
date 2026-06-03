#module byke2d::view::binding

#import byke2d::view
#import byke2d::globals

@group(0)
@binding(0)
var<uniform> view: View;

@group(0)
@binding(1)
var<uniform> globals: Globals;
