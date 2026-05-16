package byke2d

import (
	"math"

	"github.com/oliverbestmann/byke"
	"github.com/oliverbestmann/byke/spoke"
	"github.com/oliverbestmann/pulse/glm"
)

var _ = byke.ValidateComponent[Microphone]()
var _ = byke.ValidateComponent[spatialSinkMarker]()

type spatialSinkMarker struct {
	byke.ImmutableComponent[spatialSinkMarker]
}

type Microphone struct {
	byke.ImmutableComponent[Microphone]
	LeftEarOffset  glm.Vec3f
	RightEarOffset glm.Vec3f
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

	micTr := mic.Transform.Affine
	leftGlobal := micTr.Transform3(mic.Microphone.LeftEarOffset)
	rightGlobal := micTr.Transform3(mic.Microphone.RightEarOffset)

	for item := range sinksQuery.Items() {
		spatialScale := item.Sink.ps.SpatialScale

		leftScaled := leftGlobal.Mul(spatialScale)
		rightScaled := rightGlobal.Mul(spatialScale)
		emitter := item.Transform.Affine.Transform3(glm.Vec3f{}).Mul(spatialScale)

		leftVolume, rightVolume := calculateSpatialVolume(emitter, leftScaled, rightScaled)

		item.Sink.SpatialVolume(leftVolume, rightVolume)
	}
}

func calculateSpatialVolume(emitter, left, right glm.Vec3f) (float64, float64) {
	leftDistSq := float64(left.Sub(emitter).LengthSqr())
	leftDist := math.Sqrt(leftDistSq)

	rightDistSq := float64(right.Sub(emitter).LengthSqr())
	rightDist := math.Sqrt(rightDistSq)

	maxDiff := float64(left.Sub(right).Length())

	leftDiffModifier := min(1, ((leftDist-rightDist)/maxDiff+1.0)/4.0+0.5)
	rightDiffModifier := min(1, ((rightDist-leftDist)/maxDiff+1.0)/4.0+0.5)

	leftDistModifier := min(1, 1.0/leftDistSq)
	rightDistModifier := min(1, 1.0/rightDistSq)

	return leftDiffModifier * leftDistModifier, rightDiffModifier * rightDistModifier
}
