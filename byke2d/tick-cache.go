package byke2d

type Releaser interface {
	Release()
}

type tickCache[K comparable, V Releaser] struct {
	entries map[K]*tickCacheEntry[V]
}

type tickCacheEntry[V Releaser] struct {
	Value V
	InUse bool
}

func (c *tickCache[K, V]) Tick() {
	for key, entry := range c.entries {
		if !entry.InUse {
			delete(c.entries, key)
			entry.Value.Release()
		}
	}
}

func (c *tickCache[K, V]) Add(key K, value V) {
	if c.entries == nil {
		c.entries = map[K]*tickCacheEntry[V]{}
	}

	c.entries[key] = &tickCacheEntry[V]{
		Value: value,
		InUse: true,
	}
}

func (c *tickCache[K, V]) Get(key K) (V, bool) {
	entry, ok := c.entries[key]
	if !ok {
		var vZero V
		return vZero, false
	}

	// mark entry to be in use
	entry.InUse = true
	return entry.Value, ok
}
