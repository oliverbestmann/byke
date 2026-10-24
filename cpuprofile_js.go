package byke

// startCPUProfile does nothing: runtime/pprof cannot sample CPU on js/wasm.
func startCPUProfile() (stop func(), err error) {
	return func() {}, nil
}
