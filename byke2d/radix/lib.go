package radix

type Value struct {
	Key   float32
	Index uint32
}

type Cache struct {
	buf []Value
}

func Sort(sortCache *Cache, values []Value) {
	if len(values) == 0 {
		return
	}

	if cap(sortCache.buf) < len(values) {
		sortCache.buf = make([]Value, len(values))
	} else {
		sortCache.buf = sortCache.buf[:len(values)]
	}

	doSort(values, sortCache.buf)
}
