package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/pulse/vyn"
)

func main() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.LevelDebug,
	})

	slog.SetDefault(slog.New(handler))

	var app byke.App
	app.AddPlugin(byke2d.Plugin)
	app.AddSystems(byke.Update, byke2d.ExitOnEscapeSystem)

	app.AddSystems(byke.Update, func(buttons byke2d.MouseButtons, cursor byke2d.MouseCursor) {
		if buttons.IsJustPressed(vyn.MouseButton(0)) {
			fmt.Println(cursor.XY())
		}
	})

	app.MustRun()
}
