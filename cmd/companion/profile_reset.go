package main

import (
 "errors"
 "path/filepath"
 "hots-twitch-companion/internal/storage"
)

const resetMarker = storage.ResetMarker

func applyProfileReset(p paths) error {
 return storage.ApplyProfileReset(p.dir,p.config,p.state,p.secret)
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
