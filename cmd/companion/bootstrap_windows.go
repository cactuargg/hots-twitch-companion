//go:build windows

package main

import (
	"errors"
	"flag"
	"fmt"
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/dashboard"
	"hots-twitch-companion/internal/i18n"
	"hots-twitch-companion/internal/storage"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func showError(message string) {
	dll := syscall.NewLazyDLL("user32.dll")
	f := dll.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(message)
	title, _ := syscall.UTF16PtrFromString("HotS Twitch Companion")
	f.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10)
}

// Both launchers use the same profile lock, including the encrypted token file.
// Windows releases it automatically if a process crashes; file existence alone
// never prevents the next launch.
func lockDesktopProfile(dir string) (func(), error) {
	name, err := syscall.UTF16PtrFromString(filepath.Join(dir, ".instance.lock"))
	if err != nil {
		return nil, err
	}
	handle, err := syscall.CreateFile(name, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_ALWAYS, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, fmt.Errorf("Cannot open this profile; another Companion may already be using it: %w", err)
	}
	return func() { syscall.CloseHandle(handle) }, nil
}

func loadDesktopDashboard(dataDir string) (*dashboard.Dashboard, func(), error) {
	p, err := config.PathsForUser()
	if err != nil {
		return nil, nil, err
	}
	if dataDir != "" {
		dir, err := filepath.Abs(dataDir)
		if err != nil {
			return nil, nil, err
		}
		p = config.NewPaths(dir)
	}
	if err := os.MkdirAll(p.Dir, 0700); err != nil {
		return nil, nil, err
	}
	release, err := lockDesktopProfile(p.Dir)
	if err != nil {
		return nil, nil, err
	}
	if err := storage.ApplyProfileReset(p.Dir, p.Config, p.State, p.Secret); err != nil {
		release()
		return nil, nil, err
	}
	cfg, err := config.Read(p)
	if err != nil {
		release()
		return nil, nil, errors.New(i18n.Text(i18n.En, "error.read_config", err.Error()))
	}
	if err := migrateProfile(p, cfg); err != nil {
		release()
		return nil, nil, err
	}
	d, err := composeDashboard(p, cfg)
	if err != nil {
		release()
		return nil, nil, errors.New(i18n.Text(cfg.Language(), "error.startup", err.Error()))
	}
	return d, release, nil
}

func runWindowsApplication(run func(*dashboard.Dashboard) error) {
	flags := flag.NewFlagSet("HotS Twitch Companion", flag.ContinueOnError)
	dataDir := flags.String("data-dir", "", "Use a separate profile directory (for testing)")
	if err := flags.Parse(os.Args[1:]); err != nil {
		showError(err.Error())
		return
	}
	if flags.NArg() != 0 {
		showError("Unexpected command-line argument")
		return
	}
	d, release, err := loadDesktopDashboard(*dataDir)
	if err != nil {
		showError(err.Error())
		return
	}
	defer release()
	defer d.Close()
	d.StartBackground()
	if err := run(d); err != nil {
		showError(d.Text("error.open_ui", err.Error()))
	}
}
