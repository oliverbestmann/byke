//go:build !fakeClock

package byke

import "time"

func timeNow() time.Time {
	return time.Now()
}

func incrementTime(amount time.Duration) {
	// do nothing
}
