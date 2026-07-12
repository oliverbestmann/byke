package glm

import "math"

// done by chatgpt

// TryInverse returns the inverse of m and true if the inverse exists.
// Mat4f is assumed to be stored in column-major order: m[column][row].
func (m Mat4f) TryInverse() (Mat4f, bool) {
	var aug [4][8]float32 // row-major temporary: [A | I]

	// Build augmented matrix using row/column access conversion.
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			aug[r][c] = m[c][r]
		}
		aug[r][r+4] = 1
	}

	const eps = 1e-8

	// Gauss-Jordan elimination.
	for col := 0; col < 4; col++ {
		// Find pivot row.
		pivot := col
		max := float32(math.Abs(float64(aug[pivot][col])))

		for r := col + 1; r < 4; r++ {
			v := float32(math.Abs(float64(aug[r][col])))
			if v > max {
				max = v
				pivot = r
			}
		}

		if max < eps {
			return Mat4f{}, false
		}

		// Swap pivot row into place.
		if pivot != col {
			aug[col], aug[pivot] = aug[pivot], aug[col]
		}

		// Normalize pivot row.
		p := aug[col][col]
		for c := 0; c < 8; c++ {
			aug[col][c] /= p
		}

		// Eliminate this column from other rows.
		for r := 0; r < 4; r++ {
			if r == col {
				continue
			}

			f := aug[r][col]
			if math.Abs(float64(f)) < eps {
				continue
			}

			for c := 0; c < 8; c++ {
				aug[r][c] -= f * aug[col][c]
			}
		}
	}

	// Convert right half back to column-major Mat4f.
	var inv Mat4f
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			inv[c][r] = aug[r][c+4]
		}
	}

	return inv, true
}
