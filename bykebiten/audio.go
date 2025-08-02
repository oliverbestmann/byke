package bykebiten

import (
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"log/slog"
	"time"
	"unsafe"
)

var _ = byke.ValidateComponent[AudioPlayer]()
var _ = byke.ValidateComponent[AudioSink]()
var _ = byke.ValidateComponent[PlaybackSettings]()
var _ = byke.ValidateComponent[playbackDespawnMarker]()
var _ = byke.ValidateComponent[playbackRemoveMarker]()

const SampleRate = 48_000

var audioContext = audio.NewContext(SampleRate)

type AudioContext struct {
	*audio.Context
}

type AudioSource struct {
	// 16 bit 2 channel audio at SampleRate
	samples []int16
}

func samplesAsBytes[S int16 | float32](samples []S) []byte {
	ptr := (*byte)(unsafe.Pointer(unsafe.SliceData(samples)))
	return unsafe.Slice(ptr, unsafe.Sizeof(S(0))*uintptr(len(samples)))
}

func bytesAsSamples[S int16 | float32](buf []byte) []S {
	ptr := unsafe.Pointer(unsafe.SliceData(buf))

	if uintptr(ptr)%unsafe.Alignof(S(0)) != 0 {
		panic("buffer not aligned with sample type")
	}

	return unsafe.Slice((*S)(ptr), uintptr(len(buf))/unsafe.Sizeof(S(0)))
}

type AudioPlayer struct {
	byke.ImmutableComponent[AudioPlayer]
	Source *AudioSource
}

func AudioPlayerOf(source *AudioSource) *AudioPlayer {
	return &AudioPlayer{Source: source}
}

func (AudioPlayer) RequireComponents() []spoke.ErasedComponent {
	once := PlaybackSettingsOnce
	return []spoke.ErasedComponent{&once}
}

type AudioSink struct {
	byke.ImmutableComponent[AudioSink]
	ps     PlaybackSettings
	player *audio.Player
}

func createAudioSink(source *AudioSource, ps PlaybackSettings) AudioSink {
	player := audioContext.NewPlayerFromBytes(samplesAsBytes(source.samples))

	player.SetVolume(ps.Volume)
	if ps.Muted {
		player.SetVolume(0)
	}

	if ps.StartAt > 0 {
		_ = player.SetPosition(ps.StartAt)
	}

	if !ps.Paused {
		player.Play()
	}

	return AudioSink{
		ps:     ps,
		player: player,
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
}

func (as *AudioSink) Play() {
	as.ps.Paused = false

	if p := as.player; p != nil {
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
}

func (as *AudioSink) Unmute() {
	as.ps.Muted = false

	if p := as.player; p != nil {
		p.SetVolume(max(0, min(1, as.ps.Volume)))
	}
}

func (as *AudioSink) IsMuted() bool {
	return as.ps.Muted
}

func (as *AudioSink) Stop() {
	if p := as.player; p != nil {
		as.player = nil
		_ = p.Close()
	}
}

type PlaybackSettings struct {
	byke.ImmutableComponent[PlaybackSettings]
	Mode   PlaybackMode
	Volume float64
	Paused bool
	Muted  bool

	// StartAt indicates where to start the audio source
	StartAt time.Duration

	// Duration indicates the duration of the audio to play.
	// If Duration is set to zero, the audio will be played to the end.
	Duration time.Duration
}

func (p PlaybackSettings) WithStartAt(startAt time.Duration) PlaybackSettings {
	p.StartAt = startAt
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

		// create a new audio sink and insert it into the entity
		sink := createAudioSink(item.Player.Source, item.PlaybackSettings)

		switch item.PlaybackSettings.Mode {
		case PlaybackModeOnce, PlaybackModeLoop:
			commands.Entity(item.EntityId).Insert(sink)
		case PlaybackModeDespawn:
			commands.Entity(item.EntityId).Insert(sink, playbackDespawnMarker{})
		case PlaybackModeRemove:
			commands.Entity(item.EntityId).Insert(sink, playbackRemoveMarker{})
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
			slog.Debug("Despawn AudioPlayer", slog.Any("entityId", item.EntityId))
			commands.Entity(item.EntityId).Despawn()
		}
	}

	for item := range removeQuery.Items() {
		if item.AudioSink.Empty() {
			commands.Entity(item.EntityId).Update(
				byke.RemoveComponent[AudioPlayer](),
				byke.RemoveComponent[AudioSink](),
				byke.RemoveComponent[PlaybackSettings](),
				byke.RemoveComponent[playbackRemoveMarker](),
			)
		}
	}
}
