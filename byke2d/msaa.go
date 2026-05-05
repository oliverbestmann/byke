package byke2d

import "github.com/oliverbestmann/byke"

var _ = byke.ValidateComponent[MSAA]()

type MSAA struct {
	byke.ImmutableComponent[MSAA]
}
