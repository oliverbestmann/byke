package bykebiten

import (
	"log/slog"
	"time"

	eaudio "github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/bykebiten/audio"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
)

var _ = byke.ValidateComponent[AudioPlayer]()
var _ = byke.ValidateComponent[AudioSink]()
var _ = byke.ValidateComponent[PlaybackSettings]()
var _ = byke.ValidateComponent[playbackDespawnMarker]()
var _ = byke.ValidateComponent[playbackRemoveMarker]()

const SampleRate = 48_000

var audioContext = eaudio.NewContext(SampleRate)

type AudioContext struct {
	*eaudio.Context
}

type GlobalVolume struct {
	Volume float64
}

type GlobalSpatialScale struct {
	Scale gm.Vec
}

type AudioSource struct {
	factory func() audio.AudioStream[float32]
}

func (a *AudioSource) NewStream() audio.AudioStream[float32] {
	return a.factory()
}

type AudioPlayer struct {
	byke.ImmutableComponent[AudioPlayer]
	Source *AudioSource
}

func AudioPlayerOf(source *AudioSource) *AudioPlayer {
	return &AudioPlayer{Source: source}
}

func (AudioPlayer) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{PlaybackSettingsOnce}
}

type AudioSink struct {
	byke.Component[AudioSink]
	ps PlaybackSettings

	// The player. For spatial audio, this is the left ear.
	player *eaudio.Player

	// A second player. For spatial audio, this one is the right ear.
	second *eaudio.Player
}

func createAudioSink(source *AudioSource, ps PlaybackSettings) AudioSink {
	newChannel := func(channel int) *eaudio.Player {
		stream := source.NewStream()

		if ps.Mode == PlaybackModeLoop {
			stream = audio.Loop(stream)
		}

		if channel >= 0 {
			stream = audio.SelectChannel(stream, channel)
		}

		player, _ := audioContext.NewPlayerF32(audio.ToReadSeeker(stream))

		player.SetVolume(ps.Volume)
		if ps.Muted || channel >= 0 {
			player.SetVolume(0)
		}

		if ps.StartAt > 0 {
			_ = player.SetPosition(ps.StartAt)
		}

		if !ps.Paused {
			player.Play()
		}

		return player
	}

	var player, second *eaudio.Player

	if ps.Spatial {
		// we need a player for each ear
		player = newChannel(0)
		second = newChannel(1)
	} else {
		// we need only one player playing all channels
		player = newChannel(-1)
	}

	return AudioSink{
		ps:     ps,
		player: player,
		second: second,
	}
}

// Empty indicates that the audio sink has played all its media and is
// now stopped. It can not be restarted.
func (as *AudioSink) Empty() bool {
	if as.player == nil {
		return true
	}

	return !as.ps.Paused && !as.player.IsPlaying()
}

func (as *AudioSink) Pause() {
	as.ps.Paused = true

	if p := as.player; p != nil {
		p.Pause()
	}

	if p := as.second; p != nil {
		p.Pause()
	}
}

func (as *AudioSink) Play() {
	as.ps.Paused = false

	if p := as.player; p != nil {
		p.Play()
	}

	if p := as.second; p != nil {
		p.Play()
	}
}

func (as *AudioSink) IsPaused() bool {
	return as.ps.Paused
}

func (as *AudioSink) Mute() {
	as.ps.Muted = true

	if p := as.player; p != nil {
		p.SetVolume(0)
	}

	if p := as.second; p != nil {
		p.SetVolume(0)
	}
}

func (as *AudioSink) Unmute() {
	as.ps.Muted = false

	if p := as.player; p != nil {
		p.SetVolume(max(0, min(1, as.ps.Volume)))
	}

	if p := as.second; p != nil {
		p.SetVolume(max(0, min(1, as.ps.Volume)))
	}
}

func (as *AudioSink) IsMuted() bool {
	return as.ps.Muted
}

func (as *AudioSink) SpatialVolume(left, right float64) {
	if as.IsMuted() {
		return
	}

	if p := as.player; p != nil {
		p.SetVolume(as.ps.Volume * left)
	}

	if p := as.second; p != nil {
		p.SetVolume(as.ps.Volume * right)
	}
}

func (as *AudioSink) Stop() {
	if p := as.player; p != nil {
		as.player = nil
		_ = p.Close()
	}

	if p := as.second; p != nil {
		as.second = nil
		_ = p.Close()
	}
}

type PlaybackSettings struct {
	byke.ImmutableComponent[PlaybackSettings]
	Volume float64

	// If non zero, the SpatialScale will override the GlobalSpatialScale value for
	// this playback instance
	SpatialScale gm.Vec

	// StartAt indicates where to start the audio source
	StartAt time.Duration

	// Duration indicates the duration of the audio to play.
	// If Duration is set to zero, the audio will be played to the end.
	Duration time.Duration

	Mode PlaybackMode

	Paused  bool
	Muted   bool
	Spatial bool
}

func (p PlaybackSettings) WithStartAt(startAt time.Duration) PlaybackSettings {
	p.StartAt = startAt
	return p
}

func (p PlaybackSettings) WithDuration(duration time.Duration) PlaybackSettings {
	p.Duration = duration
	return p
}

func (p PlaybackSettings) WithVolume(volume float64) PlaybackSettings {
	p.Volume = volume
	return p
}

func (p PlaybackSettings) WithSpatial() PlaybackSettings {
	p.Spatial = true
	return p
}

func (p PlaybackSettings) WithSpatialScale(spatialScale gm.Vec) PlaybackSettings {
	p.Spatial = true
	p.SpatialScale = spatialScale
	return p
}

var PlaybackSettingsLoop = PlaybackSettings{
	Mode:   PlaybackModeLoop,
	Volume: 1,
}

var PlaybackSettingsOnce = PlaybackSettings{
	Mode:   PlaybackModeOnce,
	Volume: 1,
}

var PlaybackSettingsDespawn = PlaybackSettings{
	Mode:   PlaybackModeDespawn,
	Volume: 1,
}

var PlaybackSettingsRemove = PlaybackSettings{
	Mode:   PlaybackModeRemove,
	Volume: 1,
}

type PlaybackMode uint8

const (
	PlaybackModeOnce PlaybackMode = iota
	PlaybackModeLoop
	PlaybackModeDespawn
	PlaybackModeRemove
)

type playbackDespawnMarker struct {
	byke.ImmutableComponent[playbackDespawnMarker]
}

type playbackRemoveMarker struct {
	byke.ImmutableComponent[playbackRemoveMarker]
}

func createAudioSinkSystem(
	commands *byke.Commands,
	globalVolume GlobalVolume,
	globalSpatialScale GlobalSpatialScale,
	query byke.Query[struct {
		_ byke.Added[AudioPlayer]
		byke.EntityId
		Player           AudioPlayer
		PlaybackSettings PlaybackSettings
		AudioSink        byke.Option[AudioSink]
	}],
) {
	for item := range query.Items() {
		// stop any existing audio sink
		if sink, ok := item.AudioSink.Get(); ok {
			sink.Stop()
		}

		ps := item.PlaybackSettings
		ps.Volume *= globalVolume.Volume
		if ps.SpatialScale == gm.VecZero {
			ps.SpatialScale = globalSpatialScale.Scale
		}

		// create a new audio sink and insert it into the entity
		sink := createAudioSink(item.Player.Source, ps)

		var entity byke.EntityCommands

		switch item.PlaybackSettings.Mode {
		case PlaybackModeOnce, PlaybackModeLoop:
			entity = commands.Entity(item.EntityId).Insert(sink)
		case PlaybackModeDespawn:
			entity = commands.Entity(item.EntityId).Insert(sink, playbackDespawnMarker{})
		case PlaybackModeRemove:
			entity = commands.Entity(item.EntityId).Insert(sink, playbackRemoveMarker{})
		}

		if sink.ps.Spatial {
			entity.Insert(&spatialSinkMarker{})
		}
	}
}

func cleanupAudioSinkSystem(
	commands *byke.Commands,
	removeQuery byke.Query[struct {
		_ byke.With[playbackRemoveMarker]
		byke.EntityId
		AudioSink AudioSink
	}],
	despawnQuery byke.Query[struct {
		_ byke.With[playbackDespawnMarker]
		byke.EntityId
		AudioSink AudioSink
	}],
) {
	for item := range despawnQuery.Items() {
		if item.AudioSink.Empty() {
			item.AudioSink.Stop()

			slog.Debug("Despawn AudioPlayer", slog.Any("entityId", item.EntityId))
			commands.Entity(item.EntityId).Despawn()
		}
	}

	for item := range removeQuery.Items() {
		if item.AudioSink.Empty() {
			item.AudioSink.Stop()

			commands.Entity(item.EntityId).Update(
				byke.RemoveComponent[AudioPlayer](),
				byke.RemoveComponent[AudioSink](),
				byke.RemoveComponent[PlaybackSettings](),
				byke.RemoveComponent[playbackRemoveMarker](),
				byke.RemoveComponent[spatialSinkMarker](),
			)
		}
	}
}
