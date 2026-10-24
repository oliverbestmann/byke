package byke2d

import (
	"testing"
	"time"

	"github.com/oliverbestmann/byke/byke2d/vyn"
)

func TestOffscreenWindowRun(t *testing.T) {
	countFrames := func(win offscreenWindow) int {
		var frames int
		_ = win.Run(func(vyn.UpdateInputState) error {
			frames++
			time.Sleep(time.Millisecond)
			return nil
		})
		return frames
	}

	if n := countFrames(offscreenWindow{}); n != 0 {
		t.Errorf("no limit: expected 0 frames, got %d", n)
	}

	if n := countFrames(offscreenWindow{FrameCount: 5}); n != 5 {
		t.Errorf("frame count: expected 5 frames, got %d", n)
	}

	if n := countFrames(offscreenWindow{FrameCount: 5, Duration: time.Hour}); n != 5 {
		t.Errorf("frame count before duration: expected 5 frames, got %d", n)
	}

	start := time.Now()
	if n := countFrames(offscreenWindow{Duration: 20 * time.Millisecond}); n == 0 {
		t.Error("duration: expected some frames")
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("duration: ran for %s", elapsed)
	}
}
