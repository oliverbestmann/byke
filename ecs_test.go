package ecs

import (
	"github.com/pkg/profile"
	"testing"
)

func TestRunSystem(t *testing.T) {
	type X struct {
		Component[X]
		Value int
	}

	w := NewWorld()

	w.RunSystem(func(c *Commands) {
		c.Spawn(X{Value: 1})
	})

	var found bool
	w.RunSystem(func(q Query[X]) {
		for range q.Items() {
			found = true
		}
	})

	if !found {
		t.FailNow()
	}
}

func TestOptionQueryValue(t *testing.T) {
	type X struct {
		Component[X]
		Value int
	}

	w := NewWorld()

	w.RunSystem(func(c *Commands) {
		c.Spawn(X{Value: 1})
	})

	type Values struct {
		Name Option[Name]
		X    X
	}

	var found bool
	w.RunSystem(func(q Query[Values]) {
		for range q.Items() {
			found = true
		}
	})

	if !found {
		t.FailNow()
	}
}

func BenchmarkWorld_RunSystem(b *testing.B) {
	defer profile.Start(profile.CPUProfile).Stop()

	type X struct {
		Component[X]
		Value int
	}

	type Y struct {
		Component[Y]
		Value int
	}

	w := NewWorld()

	w.RunSystem(func(c *Commands) {
		for range 2000 {
			c.Spawn(X{Value: 1}, Y{Value: 2})
		}
	})

	type Values struct {
		Name Option[Name]
		X    X
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		w.RunSystem(func(q Query[Values]) {
			for range q.Items() {
				// do nothing
			}
		})
	}
}
