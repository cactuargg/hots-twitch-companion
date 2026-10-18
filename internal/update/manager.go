package update

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"
)

var ErrDisabled = errors.New("update checks disabled")
var ErrBusy = errors.New("update operation in progress")
var ErrCooldown = errors.New("wait before checking again")

type State struct {
	Enabled      bool   `json:"enabled"`
	Phase        string `json:"phase"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	CheckedAt    int64  `json:"checkedAt"`
	NextCheckAt  int64  `json:"nextCheckAt"`
	CheckAfterAt int64  `json:"checkAfterAt"`
	Error        string `json:"error"`
	Downloadable bool   `json:"downloadable"`
	Received     int64  `json:"received"`
	Total        int64  `json:"total"`
}
type Manager struct {
	mu                     sync.Mutex
	client                 *Client
	current, edition, root string
	now                    func() time.Time
	ctx                    context.Context
	cancel                 context.CancelFunc
	operation              context.CancelFunc
	generation             uint64
	busy                   bool
	wake                   chan struct{}
	state                  State
	release                Release
	prepared               Prepared
	failures               int
	notBefore              time.Time
}

func NewManager(client *Client, current, edition, root string, enabled bool) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	phase := "idle"
	if !enabled {
		phase = "disabled"
	}
	return &Manager{client: client, current: current, edition: edition, root: root, now: time.Now, ctx: ctx, cancel: cancel, wake: make(chan struct{}, 1), state: State{Enabled: enabled, Phase: phase}}
}
func (m *Manager) Snapshot() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.state
	if !m.notBefore.IsZero() {
		s.CheckAfterAt = m.notBefore.UnixMilli()
	}
	return s
}
func (m *Manager) Close() {
	m.cancel()
	m.mu.Lock()
	defer m.mu.Unlock()
	m.generation++
	if m.operation != nil {
		m.operation()
	}
}
func (m *Manager) SetEnabled(enabled bool) {
	m.mu.Lock()
	if m.state.Enabled == enabled {
		m.mu.Unlock()
		return
	}
	m.generation++
	if m.operation != nil {
		m.operation()
	}
	m.operation = nil
	m.busy = false
	m.state.Enabled = enabled
	if !enabled {
		m.state.Phase = "disabled"
		m.state.Error = ""
	} else if m.prepared.Path != "" {
		m.state.Phase = "ready"
	} else if m.state.Version != "" {
		m.state.Phase = "available"
	} else {
		m.state.Phase = "idle"
	}
	if enabled {
		m.state.NextCheckAt = 0
	}
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
}
func (m *Manager) begin(phase string) (context.Context, uint64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx.Err() != nil {
		return nil, 0, m.ctx.Err()
	}
	if !m.state.Enabled {
		return nil, 0, ErrDisabled
	}
	if m.busy {
		return nil, 0, ErrBusy
	}
	if phase == "checking" && m.now().Before(m.notBefore) {
		return nil, 0, ErrCooldown
	}
	if phase == "checking" && m.prepared.Path != "" {
		return nil, 0, ErrBusy
	}
	if phase == "downloading" && (!m.state.Downloadable || m.prepared.Path != "") {
		return nil, 0, ErrBusy
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.operation = cancel
	m.generation++
	m.busy = true
	m.state.Phase = phase
	m.state.Error = ""
	return ctx, m.generation, nil
}
func (m *Manager) Check() error {
	ctx, g, err := m.begin("checking")
	if err != nil {
		return err
	}
	m.check(ctx, g)
	return nil
}
func (m *Manager) RequestCheck() error {
	ctx, g, err := m.begin("checking")
	if err != nil {
		return err
	}
	go m.check(ctx, g)
	return nil
}
func (m *Manager) check(ctx context.Context, g uint64) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	r, err := m.client.Fetch(ctx)
	m.mu.Lock()
	defer m.mu.Unlock()
	if g != m.generation {
		return
	}
	m.busy = false
	if m.operation != nil {
		m.operation()
	}
	m.operation = nil
	defer func() {
		select {
		case m.wake <- struct{}{}:
		default:
		}
	}()
	m.state.CheckedAt = m.now().UnixMilli()
	m.notBefore = m.now().Add(30 * time.Second)
	if err != nil {
		m.failures++
		delay := time.Minute * time.Duration(1<<min(m.failures-1, 6))
		next := m.now().Add(delay)
		var api *APIError
		if errors.As(err, &api) && api.RetryAt.After(next) {
			next = api.RetryAt
		}
		m.notBefore = next
		m.state.NextCheckAt = next.UnixMilli()
		m.state.Phase = "error"
		m.state.Error = err.Error()
		return
	}
	m.failures = 0
	m.state.NextCheckAt = m.now().Add(24 * time.Hour).UnixMilli()
	m.state.Phase = "current"
	if IsNewerRelease(r.Tag, m.current) {
		m.release = r
		m.state.Version = strings.TrimPrefix(r.Tag, "v")
		m.state.URL = r.HTMLURL
		m.state.Phase = "available"
		_, packageErr := r.Package(m.edition)
		m.state.Downloadable = m.root != "" && packageErr == nil
	} else {
		m.release = Release{}
		m.state.Version = ""
		m.state.URL = ""
		m.state.Downloadable = false
	}
}
func (m *Manager) Run() {
	for {
		s := m.Snapshot()
		wait := 24 * time.Hour
		if s.Enabled && s.Phase != "ready" && s.Phase != "installing" && s.Phase != "checking" && s.Phase != "downloading" {
			m.mu.Lock()
			due := max(s.NextCheckAt, m.notBefore.UnixMilli())
			m.mu.Unlock()
			if due <= m.now().UnixMilli() {
				_ = m.Check()
				s = m.Snapshot()
				m.mu.Lock()
				due = max(s.NextCheckAt, m.notBefore.UnixMilli())
				m.mu.Unlock()
			}
			if due > 0 {
				wait = time.Until(time.UnixMilli(due))
				if wait < time.Second {
					wait = time.Second
				}
			}
		}
		timer := time.NewTimer(wait)
		select {
		case <-m.ctx.Done():
			timer.Stop()
			return
		case <-m.wake:
			timer.Stop()
		case <-timer.C:
		}
	}
}
func (m *Manager) RequestDownload() error {
	ctx, g, err := m.begin("downloading")
	if err != nil {
		return err
	}
	m.mu.Lock()
	r := m.release
	m.state.Received = 0
	m.state.Total = 0
	m.mu.Unlock()
	go func() {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
		dir, err := os.MkdirTemp(m.root, ".companion-update-")
		var p Prepared
		if err == nil {
			p, err = m.client.Download(ctx, r, m.edition, dir, func(n, total int64) {
				m.mu.Lock()
				defer m.mu.Unlock()
				if g == m.generation {
					m.state.Received = n
					m.state.Total = total
				}
			})
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		if g != m.generation {
			if p.Path != "" {
				os.Remove(p.Path)
			}
			return
		}
		m.busy = false
		if m.operation != nil {
			m.operation()
		}
		m.operation = nil
		if err != nil {
			m.state.Phase = "error"
			m.state.Error = err.Error()
			return
		}
		m.prepared = p
		m.state.Phase = "ready"
	}()
	return nil
}

// Install serializes with preference changes. The callback must acquire the
// application's idle gate before starting any replacement helper.
func (m *Manager) Install(start func(Prepared) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.ctx.Err() != nil {
		return m.ctx.Err()
	}
	if !m.state.Enabled {
		return ErrDisabled
	}
	if m.busy || m.state.Phase != "ready" || m.prepared.Path == "" {
		return ErrBusy
	}
	if err := start(m.prepared); err != nil {
		return err
	}
	m.state.Phase = "installing"
	return nil
}
