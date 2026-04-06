package byke2d

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/pulse/vyn"
)

func ExitOnEscapeSystem(writer *byke.MessageWriter[AppExit], keys Keys) {
	if keys.IsJustPressed(vyn.KeyEscape) {
		writer.Write(AppExitSuccess)
	}
}
