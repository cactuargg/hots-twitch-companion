package main

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Config writes must not replace the engine's Twitch client while scanning
// is running. The race test job also exercises concurrent reads and writes.
func TestSafeLiveConfigChangesSerializeWithScanner(t *testing.T) {
	e, _ := scanFixture(t, time.Now())
	e.mu.Lock()
	e.running = true
	e.mu.Unlock()
	e.actionMu.Lock()
	started := make(chan struct{})
	done := make(chan error, 1)
	cfg := e.cfg
	cfg.Lang = "en"
	cfg.WizardDone = true
	go func() {
		close(started)
		done <- e.updateConfig(cfg)
	}()
	<-started
	select {
	case err := <-done:
		e.actionMu.Unlock()
		t.Fatalf("settings changed while scan lock held: %v", err)
	case <-time.After(35 * time.Millisecond):
	}
	e.actionMu.Unlock()
	if err := <-done; err != nil { t.Fatalf("safe live settings update failed: %v", err) }
	e.mu.Lock()
	got := e.cfg
	e.mu.Unlock()
	if got.Lang != "en" || !got.WizardDone { t.Fatalf("settings not updated: %+v", got) }

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		lastError := ""
		lastLog := time.Time{}
		for i := 0; i < 20; i++ { e.scanTick(context.Background(), time.Now().Add(-time.Minute), &lastError, &lastLog) }
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < 20; i++ {
			next := cfg
			if i % 2 == 0 { next.Lang = "ru" } else { next.Lang = "en" }
			next.WizardDone = i%2 == 0
			if err := e.updateConfig(next); err != nil { t.Errorf("safe settings update: %v", err); return }
		}
	}()
	wg.Wait()
}
