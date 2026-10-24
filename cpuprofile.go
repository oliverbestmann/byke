//go:build !js

package byke

import (
	"fmt"
	"os"
	"runtime/pprof"
)

// startCPUProfile writes a CPU profile to the file named in BYKE_CPUPROFILE, if set.
func startCPUProfile() (stop func(), err error) {
	path := os.Getenv("BYKE_CPUPROFILE")
	if path == "" {
		return func() {}, nil
	}

	fp, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create cpu profile: %w", err)
	}

	if err := pprof.StartCPUProfile(fp); err != nil {
		_ = fp.Close()
		return nil, fmt.Errorf("start cpu profile: %w", err)
	}

	stop = func() {
		pprof.StopCPUProfile()

		if err := fp.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "close cpu profile: %s\n", err)
		}
	}

	return stop, nil
}
