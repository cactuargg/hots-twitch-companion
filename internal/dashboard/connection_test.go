package dashboard

import (
	"context"
	"errors"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/twitch"
	"sync"
	"testing"
	"time"
)

type connectionApplication struct {
	Application
	mu       sync.Mutex
	calls    int
	login    string
	verified chan struct{}
}

func (a *connectionApplication) Config() config.Data { return config.Data{Lang: "en"} }
func (a *connectionApplication) StopWatch()          {}
func (a *connectionApplication) Connection() twitch.Connection {
	a.mu.Lock()
	defer a.mu.Unlock()
	return twitch.Connection{HasToken: true, Login: a.login}
}
func (a *connectionApplication) Identity(context.Context) (twitch.Identity, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.calls++
	if a.calls == 1 {
		return twitch.Identity{}, errors.New("temporarily offline")
	}
	a.login = "demo"
	select {
	case a.verified <- struct{}{}:
	default:
	}
	return twitch.Identity{Login: "demo"}, nil
}
func TestConnectionRecoversWithoutAuthorizationOrStartingMonitoring(t *testing.T) {
	a := &connectionApplication{verified: make(chan struct{}, 1)}
	d := New(a, config.Paths{}, "test")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	d.backgroundCancel = cancel
	done := make(chan struct{})
	go func() { defer close(done); d.watchConnection(ctx, time.Millisecond) }()
	select {
	case <-a.verified:
	case <-time.After(time.Second):
		t.Fatal("connection was not retried")
	}
	d.Close()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("connection watcher ignored cancellation")
	}
	if a.Connection().Login != "demo" {
		t.Fatal("connection was not restored")
	}
	// All other operations remain nil: starting wagers or OAuth would panic.
}
