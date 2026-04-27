package byke2d

import "github.com/oliverbestmann/byke"

var _ = byke.ValidateComponent[Msaa]()

type Msaa struct {
	byke.Component[Msaa]
	On bool
}

var MsaaOn = Msaa{On: true}
var MsaaOff = Msaa{}
