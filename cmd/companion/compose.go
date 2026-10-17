package main

import (
	"hots-twitch-companion/internal/app"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/dashboard"
	"hots-twitch-companion/internal/storage"
)

func migrateProfile(p config.Paths, c config.Data) error {
	if err := storage.MigrateLegacySimulation(p.State); err != nil {
		return err
	}
	return config.Write(p, c)
}

func composeDashboard(p config.Paths, c config.Data) (*dashboard.Dashboard, error) {
	var d *dashboard.Dashboard
	engine, err := app.New(p, c, app.Options{
		DefaultClientID: defaultTwitchClientID,
		Log:             func(message string) { d.LogEvent("info", "", message) },
		LogEvent:        func(level, code, message string) { d.LogEvent(level, code, message) },
	})
	if err != nil {
		return nil, err
	}
	d = dashboard.New(engine, p, appVersion)
	return d, nil
}
