package partycle

import "github.com/oliverbestmann/byke"

var Systems = &byke.SystemSet{}

//goland:noinspection GoNameStartsWithPackageName
func Plugin(app *byke.App) {
	app.AddSystems(byke.Update, byke.System(emitterSystem, particleSystem).
		Chain().
		InSet(Systems))
}
