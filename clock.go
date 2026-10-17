package main

import "time"

// Clock supplies decision time for the Engine. Production uses the system
// clock; tests can deterministically advance the logical time between
// lifecycle transitions without sleeping for minutes.
type Clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

var _ Clock = systemClock{}
