package vyn

import "github.com/oliverbestmann/webgpu/wgpu"

type Window interface {
	// GetSize returns the size of the window
	GetSize() (uint32, uint32)

	// SurfaceDescriptor returns a surface descriptor for the window.
	// This can be used to initialize a wgpu context.
	SurfaceDescriptor() *wgpu.SurfaceDescriptor

	// Run calls the given functions in an event loop.
	Run(render func(inputState UpdateInputState) error) error

	// Terminate closes the window
	Terminate()
}
