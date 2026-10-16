//go:build windows

package main

import "testing"

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
	d := manualDashboard(t, Config{WizardDone: true, PlayerName: "Previous"})
	if err := writeConfig(d.p, d.config()); err != nil {
		t.Fatal(err)
	}
	if err := d.requestProfileReset(); err != nil {
		t.Fatal(err)
	}
	fresh, release, err := loadDesktopDashboard(d.p.dir)
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if fresh.config().WizardDone || fresh.config().PlayerName != "" || fresh.config().RealMode {
		t.Fatal("profile was restored instead of reset")
	}
}
