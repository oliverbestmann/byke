package byke2d

//go:generate stringer -type=AlphaMode
type AlphaMode uint8

const (
	AlphaModeOpaque AlphaMode = iota
	AlphaModeMask
	AlphaModeBlend
	Premultiplied
	AlphaModeAlphaToCoverage
	AlphaModeAdd
	AlphaModeMultiply
)
