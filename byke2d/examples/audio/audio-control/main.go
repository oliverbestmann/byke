package main

import (
	. "github.com/oliverbestmann/byke"
	. "github.com/oliverbestmann/byke/byke2d"
	"github.com/oliverbestmann/byke/byke2d/examples/shared"
	"github.com/oliverbestmann/byke/byke2d/glm"
	"github.com/oliverbestmann/byke/byke2d/vyn"
)

// Adapted from https://bevy.org/examples/audio/audio-control/
// Loops music that can be paused and muted, and plays a one shot sound on demand.

type Music struct {
	ImmutableComponent[Music]
}

type Sounds struct {
	Catch *AudioSource
}

func main() {
	var app App

	app.AddPlugin(PluginRender)
	app.AddPlugin(shared.PluginHint("Space to pause, M to mute, B to play a sound"))

	app.AddSystems(Startup, setupSystem)
	app.AddSystems(Update, ExitOnEscapeSystem, controlMusicSystem, playSoundSystem)

	shared.RunAppInTest(app, shared.FramesToSnapshot{10})
}

func setupSystem(commands *Commands, assets *Assets) {
	commands.Spawn(
		Camera{},
		Camera2d,
		OrthographicProjection{ViewportOrigin: glm.Vec2f{0.5, 0.5}},
	)

	// the AudioSink is added to this entity once playback starts
	commands.Spawn(
		Music{},
		AudioPlayerOf(assets.Audio("music.ogg").Await()),
		PlaybackSettingsLoop.WithVolume(0.5),
		AnchorCenter,
		Text{Text: "Playing", Size: 48, Color: ColorWhite},
	)

	commands.InsertResource(Sounds{Catch: assets.Audio("catch.ogg").Await()})
}

func controlMusicSystem(
	keys Keys,
	music Single[struct {
		_    With[Music]
		Sink *AudioSink
		Text *Text
	}],
) {
	sink := music.Value.Sink

	if keys.IsJustPressed(vyn.KeySpace) {
		if sink.IsPaused() {
			sink.Play()
		} else {
			sink.Pause()
		}
	}

	if keys.IsJustPressed(vyn.KeyM) {
		if sink.IsMuted() {
			sink.Unmute()
		} else {
			sink.Mute()
		}
	}

	status := "Playing"
	if sink.IsPaused() {
		status = "Paused"
	}

	if sink.IsMuted() {
		status += " (muted)"
	}

	if music.Value.Text.Text != status {
		music.Value.Text.Text = status
	}
}

func playSoundSystem(commands *Commands, keys Keys, sounds Sounds) {
	if keys.IsJustPressed(vyn.KeyB) {
		// the entity despawns itself once the sound has finished
		commands.Spawn(AudioPlayerOf(sounds.Catch), PlaybackSettingsDespawn)
	}
}
