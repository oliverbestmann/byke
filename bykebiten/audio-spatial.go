package bykebiten

import (
	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/gm"
	"github.com/oliverbestmann/byke/spoke"
	"math"
)

var _ = byke.ValidateComponent[Microphone]()
var _ = byke.ValidateComponent[spatialSinkMarker]()

type spatialSinkMarker struct {
	byke.ImmutableComponent[spatialSinkMarker]
}

type Microphone struct {
	byke.ImmutableComponent[Microphone]
	LeftEarOffset  gm.Vec
	RightEarOffset gm.Vec
}

func (Microphone) RequireComponents() []spoke.ErasedComponent {
	return []spoke.ErasedComponent{
		NewTransform(),
	}
}

func adjustSpatialAudioVolume(
	microphoneQuery byke.Query[struct {
		Microphone Microphone
		Transform  GlobalTransform
	}],

	sinksQuery byke.Query[struct {
		_         byke.With[spatialSinkMarker]
		Sink      *AudioSink
		Transform GlobalTransform
	}],
) {
	mic, ok := microphoneQuery.Single()
	if !ok {
		return
	}

	micTr := mic.Transform.AsAffine()
	leftGlobal := micTr.Transform(mic.Microphone.LeftEarOffset)
	rightGlobal := micTr.Transform(mic.Microphone.RightEarOffset)

	for item := range sinksQuery.Items() {
		spatialScale := item.Sink.ps.SpatialScale

		leftScaled := leftGlobal.MulEach(spatialScale)
		rightScaled := rightGlobal.MulEach(spatialScale)
		emitter := item.Transform.Translation.MulEach(spatialScale)

		leftVolume, rightVolume := calculateSpatialVolume(emitter, leftScaled, rightScaled)

		item.Sink.SpatialVolume(leftVolume, rightVolume)
	}
}

func calculateSpatialVolume(emitter, left, right gm.Vec) (float64, float64) {
	leftDistSq := left.DistanceToSqr(emitter)
	leftDist := math.Sqrt(leftDistSq)

	rightDistSq := right.DistanceToSqr(emitter)
	rightDist := math.Sqrt(rightDistSq)

	maxDiff := left.DistanceTo(right)

	leftDiffModifier := min(1, ((leftDist-rightDist)/maxDiff+1.0)/4.0+0.5)
	rightDiffModifier := min(1, ((rightDist-leftDist)/maxDiff+1.0)/4.0+0.5)

	leftDistModifier := min(1, 1.0/leftDistSq)
	rightDistModifier := min(1, 1.0/rightDistSq)

	return leftDiffModifier * leftDistModifier, rightDiffModifier * rightDistModifier
}
