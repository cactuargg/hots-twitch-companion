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
