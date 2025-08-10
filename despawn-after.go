package byke

import "time"

type DespawnWithDelay struct {
	Component[DespawnWithDelay]
	Timer Timer
}

func DespawnAfter(duration time.Duration) DespawnWithDelay {
	return DespawnWithDelay{
		Timer: NewTimer(duration, TimerModeOnce),
	}
}

func despawnWithDelaySystem(
	commands *Commands,
	vt VirtualTime,
	query Query[struct {
		EntityId
		DespawnWithDelay *DespawnWithDelay
	}],
) {
	for item := range query.Items() {
		timer := &item.DespawnWithDelay.Timer
		if timer.Tick(vt.Delta).JustFinished() || timer.Finished() {
			commands.Entity(item.EntityId).Despawn()
		}
	}
}
