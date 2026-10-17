package app

import (
	"context"
	"errors"
	predmodel "hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/twitch/transport"
	"testing"
	"time"
)

type manualClock struct{ current time.Time }

func (c *manualClock) Now() time.Time          { return c.current }
func (c *manualClock) Advance(d time.Duration) { c.current = c.current.Add(d) }

func TestClockControlsSettlementRetryWithoutSleeping(t *testing.T) {
	e, p := liveLifecycleTest(t)
	now := time.Date(2026, 10, 17, 15, 0, 0, 0, time.UTC)
	clock := &manualClock{current: now}
	e.clock = clock
	fake := &fakePredictionAPI{onEnd: func(*predmodel.Prediction, string) error { return errors.New("offline") }}
	e.predictions = fake
	s := e.state
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil {
		t.Fatal("expected initial Twitch failure")
	}
	retry := s.Active.Settlement
	if retry == nil || retry.Outcome != "win" || retry.RetryAt != now.Add(10*time.Second).UnixMilli() || fake.endCalls != 1 {
		t.Fatalf("first retry not scheduled from injected time: %+v calls=%d", retry, fake.endCalls)
	}
	clock.Advance(9 * time.Second)
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err == nil || fake.endCalls != 1 {
		t.Fatalf("retried before deadline: calls=%d err=%v", fake.endCalls, err)
	}
	clock.Advance(time.Second)
	fake.onEnd = nil
	if err := e.doSettle(context.Background(), &s, "win", "manual"); err != nil {
		t.Fatal(err)
	}
	if fake.endCalls != 2 || s.Active != nil || s.LastSettlementAt != clock.Now().UnixMilli() {
		t.Fatalf("retry did not commit using fake time: state=%+v calls=%d", s, fake.endCalls)
	}
	reloaded, err := loadState(p)
	if err != nil || reloaded.Active != nil || reloaded.LastSettlementAt != clock.Now().UnixMilli() {
		t.Fatalf("retry transition not durable: %+v %v", reloaded, err)
	}
}

func TestClockKeepsTwitchRateLimitRetryDeadline(t *testing.T) {
	e, _ := liveLifecycleTest(t)
	now := time.Date(2026, 10, 17, 16, 0, 0, 0, time.UTC)
	e.clock = &manualClock{current: now}
	limit := now.Add(90 * time.Second).UnixMilli()
	e.predictions = &fakePredictionAPI{onEnd: func(*predmodel.Prediction, string) error {
		return &transport.HTTPError{Status: 429, Message: "slow down", RetryAt: limit}
	}}
	s := e.state
	if err := e.doSettle(context.Background(), &s, "loss", "manual"); err == nil {
		t.Fatal("missing rate limit")
	}
	if s.Active == nil || s.Active.Settlement == nil || s.Active.Settlement.RetryAt != limit ||
		s.Active.Settlement.Outcome != "loss" || s.Active.Settlement.Source != "manual" {
		t.Fatalf("rate limit was not preserved: %+v", s)
	}
}

func TestClockControlsUncertainCreationTimestamp(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	e, p := liveLifecycleTest(t)
	now := time.Now().Add(-time.Second)
	clock := &manualClock{current: now}
	e.clock = clock
	s := e.state
	s.Active = nil
	s.LastCreatedLobbySHA = ""
	s.RecentLobbySHAs = nil
	if err := e.updateAndSave(s); err != nil {
		t.Fatal(err)
	}
	fake := &fakePredictionAPI{onCreate: func() error { return errors.New("lost POST response") }}
	e.predictions = fake
	lobbyFileTest(t, realLobbyTest(t), now.Add(-4*time.Second))
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), now.Add(-time.Minute), &lastError, &lastLog)
	stored, err := loadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if stored.UnconfirmedCreation == nil || stored.UnconfirmedCreation.AttemptedAt != now.UnixMilli() ||
		fake.createCalls != 1 {
		t.Fatalf("creation timestamp not from injected clock: %+v calls=%d", stored, fake.createCalls)
	}
	clock.Advance(time.Second)
	e.scanTick(context.Background(), now.Add(-time.Minute), &lastError, &lastLog)
	if fake.createCalls != 1 {
		t.Fatalf("unconfirmed POST retried: %d", fake.createCalls)
	}
}

func TestClockControlsHundredMinuteTimeoutBoundary(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("TMP", temp)
	t.Setenv("TEMP", temp)
	end := time.Date(2026, 10, 17, 19, 0, 0, 0, time.UTC)
	e, p := scanFixture(t, end.Add(-100*time.Minute))
	clock := &manualClock{current: end}
	e.clock = clock
	lastError := ""
	lastLog := time.Time{}
	e.scanTick(context.Background(), end.Add(-101*time.Minute), &lastError, &lastLog)
	if e.state.Active == nil {
		t.Fatal("Prediction canceled at exactly the 100-minute boundary")
	}
	clock.Advance(time.Millisecond)
	e.scanTick(context.Background(), end.Add(-101*time.Minute), &lastError, &lastLog)
	if e.state.Active != nil || e.state.LastSettlementAt != clock.Now().UnixMilli() {
		t.Fatalf("Prediction was not canceled immediately after timeout: %+v", e.state)
	}
	persisted, err := loadState(p)
	if err != nil || persisted.Active != nil || persisted.LastSettlementAt != clock.Now().UnixMilli() {
		t.Fatalf("timeout cancel was not persisted: %+v %v", persisted, err)
	}
}
