// Package clock provides the Engine's time source. Inject a Clock in tests
// to advance retry deadlines and timeout boundaries without sleeping.
package clock

import "time"

// Clock supplies decision timestamps to the Engine.
type Clock interface {
	Now() time.Time
}

// System uses the operating system wall clock in production.
type System struct{}

func (System) Now() time.Time { return time.Now() }

var _ Clock = System{}
