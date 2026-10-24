//go:build !js

package byke

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStartCPUProfile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpu.pprof")
	t.Setenv("BYKE_CPUPROFILE", path)

	stop, err := startCPUProfile()
	if err != nil {
		t.Fatal(err)
	}

	stop()

	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if stat.Size() == 0 {
		t.Fatal("cpu profile is empty")
	}
}
