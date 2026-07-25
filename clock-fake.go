//go:build fakeClock

package byke

import "time"

var globalTime time.Time = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func timeNow() time.Time {
	return globalTime
}

func incrementTime(amount time.Duration) {
	globalTime = globalTime.Add(amount)
}
