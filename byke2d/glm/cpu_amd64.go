package glm

import "golang.org/x/sys/cpu"

// useSimd reports whether the assembly (AVX) and archsimd (FMA) code paths can run.
var useSimd = cpu.X86.HasAVX && cpu.X86.HasFMA
