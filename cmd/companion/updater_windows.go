//go:build windows

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"hots-twitch-companion/internal/update"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type portableUpdatePlan struct {
	ParentPID int             `json:"parentPid"`
	Target    string          `json:"target"`
	Profile   string          `json:"profile"`
	Prepared  update.Prepared `json:"prepared"`
}

func startHidden(path string, args ...string) (*exec.Cmd, error) {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
func startPortableUpdate(target, profile string, p update.Prepared) error {
	if filepath.Dir(filepath.Dir(p.Path)) != filepath.Dir(target) {
		return errors.New("update staging must be beside the executable")
	}
	hash, err := update.VerifyExecutable(p.Path)
	if err != nil {
		return err
	}
	if hash != p.SHA256 {
		return errors.New("downloaded executable changed")
	}
	dir := filepath.Dir(p.Path)
	helper := filepath.Join(dir, "updater.exe")
	src, err := os.Open(target)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(helper, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	syncErr := dst.Sync()
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	if closeErr != nil {
		return closeErr
	}
	plan := portableUpdatePlan{os.Getpid(), target, profile, p}
	data, err := json.Marshal(plan)
	if err != nil {
		return err
	}
	planPath := filepath.Join(dir, "plan.json")
	if err = os.WriteFile(planPath, data, 0600); err != nil {
		return err
	}
	ready := filepath.Join(dir, "ready")
	if err = os.Remove(ready); err != nil && !os.IsNotExist(err) {
		return err
	}
	cmd, err := startHidden(helper, "--apply-update", planPath)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timeout := time.NewTimer(5 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			return fmt.Errorf("update helper exited before ready: %v", err)
		case <-timeout.C:
			_ = cmd.Process.Kill()
			return errors.New("update helper did not become ready; application remains open")
		case <-ticker.C:
			if _, err := os.Stat(ready); err == nil {
				return nil
			}
		}
	}
}
func applyPortableUpdate(planPath string) error {
	f, err := os.Open(planPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var plan portableUpdatePlan
	decoder := json.NewDecoder(io.LimitReader(f, 64<<10))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&plan); err != nil {
		return err
	}
	dir := filepath.Dir(planPath)
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if !filepath.IsAbs(planPath) || self != filepath.Join(dir, "updater.exe") || plan.ParentPID <= 0 || !filepath.IsAbs(plan.Profile) || plan.Prepared.Path != filepath.Join(dir, "next.exe") || filepath.Dir(plan.Target) != filepath.Dir(dir) || !strings.EqualFold(filepath.Ext(plan.Target), ".exe") {
		return errors.New("invalid update helper plan")
	}
	hash, err := update.VerifyExecutable(plan.Prepared.Path)
	if err != nil {
		return err
	}
	if hash != plan.Prepared.SHA256 {
		return errors.New("staged update checksum changed")
	}
	parent, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(plan.ParentPID))
	if err != nil {
		return err
	}
	defer syscall.CloseHandle(parent)
	if err = os.WriteFile(filepath.Join(dir, "ready"), []byte("ready"), 0600); err != nil {
		return err
	}
	event, err := syscall.WaitForSingleObject(parent, 60000)
	if err != nil {
		return err
	}
	if event != syscall.WAIT_OBJECT_0 {
		return errors.New("old application has not exited; executable was not replaced")
	}
	var release func()
	until := time.Now().Add(5 * time.Second)
	for {
		release, err = lockDesktopProfile(plan.Profile)
		if err == nil {
			break
		}
		if time.Now().After(until) {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	defer func() {
		if release != nil {
			release()
		}
	}()
	attempt := 0
	return update.Replace(plan.Target, plan.Prepared, func(path string) error {
		if release != nil {
			release()
			release = nil
		}
		attempt++
		if attempt > 1 {
			cmd, err := startHidden(path, "--data-dir", plan.Profile)
			if err == nil {
				go cmd.Wait()
			}
			return err
		}
		return startAndConfirmUpdate(path, plan.Profile, filepath.Join(dir, "started"))
	})
}

func startAndConfirmUpdate(path, profile, marker string) error {
	if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
		return err
	}
	cmd, err := startHidden(path, "--data-dir", profile, "--update-ready-file", marker)
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-done:
			if _, statErr := os.Stat(marker); statErr == nil {
				return nil
			}
			return fmt.Errorf("new application exited before startup acknowledgement: %v", err)
		case <-ticker.C:
			if _, err := os.Stat(marker); err == nil {
				return nil
			}
		case <-timer.C:
			_ = cmd.Process.Kill()
			<-done
			return errors.New("new application did not confirm startup; restoring previous version")
		}
	}
}
