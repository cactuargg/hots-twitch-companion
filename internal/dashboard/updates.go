package dashboard

import (
	"errors"
	"hots-twitch-companion/internal/update"
)

const releasesLatestURL = update.LatestURL

// ConfigurePortableUpdates is set by the Windows composition root before any
// background work starts. Tests can supply a manager/installer without processes.
func (d *Dashboard) ConfigurePortableUpdates(edition, root string, install func(update.Prepared) error) {
	d.updates.Close()
	d.updates = update.NewManager(update.NewClient(), d.version, edition, root, !d.Config().DisableUpdateChecks)
	d.installUpdate = install
}
func (d *Dashboard) watchUpdates() { d.updates.Run() }

func (d *Dashboard) installPreparedUpdate() error {
	if d.updateStartupPending() {
		return errors.New(d.Text("update.startup_wait"))
	}
	d.mu.Lock()
	authPending := d.authPending
	d.mu.Unlock()
	if authPending {
		return errors.New(d.Text("update.auth_busy"))
	}
	if d.installUpdate == nil {
		return errors.New(d.Text("update.unavailable"))
	}
	gate, ok := d.engine.(interface{ BeginUpdate(func() error) error })
	if !ok {
		return errors.New(d.Text("update.unavailable"))
	}
	err := d.updates.Install(func(p update.Prepared) error { return gate.BeginUpdate(func() error { return d.installUpdate(p) }) })
	if err == nil {
		d.Close()
	}
	return err
}
