//go:build windows

package main

import (
	"bytes"
	"encoding/json"
	"hots-twitch-companion/internal/update"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestPortableHelperSubprocess(t *testing.T) {
	plan := os.Getenv("COMPANION_TEST_UPDATE_PLAN")
	if plan == "" {
		return
	}
	if err := applyPortableUpdate(plan); err != nil {
		t.Fatal(err)
	}
}
func TestPortableHelperWaitsReplacesRestartsAndPreservesProfile(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Companion.exe")
	build := exec.Command("go", "build", "-o", target, "./testdata/portable")
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("fixture build: %v %s", err, b)
	}
	old, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	profile := filepath.Join(root, "profile")
	os.Mkdir(profile, 0700)
	config := []byte(`{"playerName":"unchanged"}`)
	os.WriteFile(filepath.Join(profile, "config.json"), config, 0600)
	stage := filepath.Join(root, ".companion-update-test")
	os.Mkdir(stage, 0700)
	payload := filepath.Join(stage, "next.exe")
	newBytes := append(append([]byte{}, old...), []byte("new version fixture")...)
	os.WriteFile(payload, newBytes, 0700)
	hash, err := update.VerifyExecutable(payload)
	if err != nil {
		t.Fatal(err)
	}
	stop := filepath.Join(root, "stop-parent")
	parent, err := startHidden(target, "--wait", stop)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = parent.Process.Kill(); _ = parent.Wait() }()
	self, _ := os.Executable()
	testExe, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(stage, "updater.exe")
	os.WriteFile(helper, testExe, 0700)
	plan := portableUpdatePlan{parent.Process.Pid, target, profile, update.Prepared{Path: payload, SHA256: hash, Version: "9.0.0"}}
	data, _ := json.Marshal(plan)
	planPath := filepath.Join(stage, "plan.json")
	os.WriteFile(planPath, data, 0600)
	process, err := startHiddenTestHelper(helper, planPath)
	if err != nil {
		t.Fatal(err)
	}
	defer process.Process.Kill()
	waitFile(t, filepath.Join(stage, "ready"))
	before, _ := os.ReadFile(target)
	if !bytes.Equal(before, old) {
		t.Fatal("replaced executable before parent exited")
	}
	os.WriteFile(stop, []byte("stop"), 0600)
	done := make(chan error, 1)
	go func() { done <- process.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("helper did not finish")
	}
	waitFile(t, filepath.Join(profile, "restarted"))
	until := time.Now().Add(10 * time.Second)
	for {
		file, err := os.OpenFile(target, os.O_WRONLY, 0)
		if err == nil {
			file.Close()
			break
		}
		if time.Now().After(until) {
			t.Fatal("fixture executable remained locked")
		}
		time.Sleep(25 * time.Millisecond)
	}
	after, _ := os.ReadFile(target)
	backup, _ := os.ReadFile(filepath.Join(stage, "previous.exe"))
	preserved, _ := os.ReadFile(filepath.Join(profile, "config.json"))
	if !bytes.Equal(after, newBytes) || !bytes.Equal(backup, old) || !bytes.Equal(preserved, config) {
		t.Fatal("replacement, backup or profile preservation failed")
	}
}
func startHiddenTestHelper(path, plan string) (*exec.Cmd, error) {
	cmd := exec.Command(path, "-test.run=^TestPortableHelperSubprocess$")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	cmd.Env = append(os.Environ(), "COMPANION_TEST_UPDATE_PLAN="+plan)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
func waitFile(t *testing.T, path string) {
	t.Helper()
	until := time.Now().Add(10 * time.Second)
	for time.Now().Before(until) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("file never appeared: %s", path)
}

func TestFailedStartupRestoresAndRestartsPreviousExecutable(t *testing.T) {
	t.Setenv("COMPANION_TEST_FAIL_START", "1")
	root := t.TempDir()
	target := filepath.Join(root, "Companion.exe")
	build := exec.Command("go", "build", "-o", target, "./testdata/portable")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("fixture build: %v %s", err, out)
	}
	old, _ := os.ReadFile(target)
	stage := filepath.Join(root, ".companion-update-test")
	os.Mkdir(stage, 0700)
	profile := filepath.Join(root, "profile")
	os.Mkdir(profile, 0700)
	payload := filepath.Join(stage, "next.exe")
	os.WriteFile(payload, append(append([]byte{}, old...), []byte("new")...), 0700)
	hash, err := update.VerifyExecutable(payload)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	err = update.Replace(target, update.Prepared{Path: payload, SHA256: hash, Version: "9.0.0"}, func(path string) error {
		calls++
		if calls == 1 {
			return startAndConfirmUpdate(path, profile, filepath.Join(stage, "started"))
		}
		cmd, err := startHidden(path, "--data-dir", profile)
		if err != nil {
			return err
		}
		return cmd.Wait()
	})
	if err == nil || calls != 2 {
		t.Fatalf("startup failure did not trigger recovery: %v (%d starts)", err, calls)
	}
	actual, _ := os.ReadFile(target)
	if !bytes.Equal(old, actual) {
		t.Fatal("previous executable not restored")
	}
	waitFile(t, filepath.Join(profile, "restarted"))
}
