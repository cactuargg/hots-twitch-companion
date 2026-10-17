//go:build windows

package main

import (
	"hots-twitch-companion/internal/config"
	"testing"
)

func TestDesktopProfileExclusiveAndReusable(t *testing.T) {
	dir := t.TempDir()
	release, err := lockDesktopProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := lockDesktopProfile(dir)
	if err == nil {
		second()
		release()
		t.Fatal("second process acquired profile")
	}
	release()
	release, err = lockDesktopProfile(dir)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestResetLaunchStartsWithFreshWizard(t *testing.T) {
	p := config.NewPaths(t.TempDir())
	if err := config.Write(p, config.Data{WizardDone: true, PlayerName: "Previous"}); err != nil {
		t.Fatal(err)
	}
	d, releaseFirst, err := loadDesktopDashboard(p.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.RequestProfileReset(); err != nil {
		releaseFirst()
		t.Fatal(err)
	}
	releaseFirst()
	fresh, release, err := loadDesktopDashboard(p.Dir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if fresh.Config().WizardDone || fresh.Config().PlayerName != "" || !fresh.Config().RealMode {
		t.Fatal("profile was restored instead of reset")
	}
}
