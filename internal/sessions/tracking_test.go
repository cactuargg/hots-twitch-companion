package sessions

import "testing"

func TestTrackingPauseAndRestartAreDifferent(t *testing.T) {
	s := State{Tracking: []Period{{From: 100, To: 200}, {From: 300}}}
	for _, tc := range []struct {
		at   int64
		want bool
	}{{99, false}, {100, true}, {199, true}, {200, false}, {250, false}, {300, true}, {10000, true}} {
		if got := TrackingAt(s, tc.at); got != tc.want {
			t.Fatalf("at %d: %v", tc.at, got)
		}
	}
	// No process-start timestamp participates in the decision.
}
