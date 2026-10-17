package main

import (
	"errors"
	"os"
	"path/filepath"
)

const resetMarker = ".reset-profile"

// Delete only Companion-owned data, before any token or WebView is loaded.
// Keep the marker until every removal succeeds so a failed reset retries safely.
func applyProfileReset(p paths) error {
	marker := filepath.Join(p.dir, resetMarker)
	if _, err := os.Stat(marker); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	for _, name := range []string{p.config, p.state, p.secret, p.config + ".new", p.state + ".new", p.secret + ".new", marker + ".new"} {
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := os.RemoveAll(filepath.Join(p.dir, "webview2")); err != nil {
		return err
	}
	return os.Remove(marker)
}

func (d *dashboard) requestProfileReset() error {
	e := d.engine
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	// Do not erase the only durable warning for a remote Prediction that
	// may have been created, even if monitoring has already stopped.
	if e.running || e.state.Active != nil || e.state.UnconfirmedCreation != nil {
		return errors.New(tr(e.cfg.lang(), "reset.busy"))
	}
	if err := saveJSON(filepath.Join(d.p.dir, resetMarker), true); err != nil {
		return err
	}
	d.shutdownOnce.Do(func() { close(d.shutdown) })
	return nil
}
