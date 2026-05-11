package byke2d

import "github.com/oliverbestmann/byke"

// The Core2d schedule does 2d rendering. The Core2d schedule is executed once
// per camera and will have an ActiveView set on the active camera. You can use
// the ViewQuery system parameter to extract values for the currently active
// Camera & ViewTarget.
var Core2d = byke.MakeScheduleId("Core2d")
