//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"
)

func launchDesktop(url string) error {
	// Microsoft Edge app mode gives a separate minimal browser window, no CLI or local install.
	candidates := []string{}
	for _, dir := range []string{os.Getenv("PROGRAMFILES(X86)"), os.Getenv("PROGRAMFILES"), os.Getenv("LOCALAPPDATA")} {
		if dir != "" {
			candidates = append(candidates, filepath.Join(dir, "Microsoft", "Edge", "Application", "msedge.exe"))
		}
	}
	if found, e := exec.LookPath("msedge.exe"); e == nil {
		candidates = append([]string{found}, candidates...)
	}
	for _, path := range candidates {
		if stat, err := os.Stat(path); err != nil || stat.IsDir() {
			continue
		}
		cmd := exec.Command(path, "--app="+url, "--window-size=1440,950", "--no-first-run")
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return nil
		}
	}
	// Fallback: default browser opens the same local UI without installing anything.
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
}
func showError(message string) {
	dll := syscall.NewLazyDLL("user32.dll")
	f := dll.NewProc("MessageBoxW")
	text, _ := syscall.UTF16PtrFromString(message)
	title, _ := syscall.UTF16PtrFromString("HotS Twitch Companion")
	f.Call(0, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(title)), 0x10)
}
func main() {
	p, err := pathsForUser()
	if err != nil {
		showError(err.Error())
		return
	}
	if err = os.MkdirAll(p.dir, 0700); err != nil {
		showError(err.Error())
		return
	}
	cfg, err := readConfig(p)
	if err != nil {
		showError("Ошибка чтения настроек: " + err.Error())
		return
	}
	d, err := newDashboard(p, cfg)
	if err != nil {
		showError("Ошибка запуска: " + err.Error())
		return
	}
	if err = d.runDesktop(launchDesktop); err != nil {
		showError(fmt.Sprintf("Не удалось открыть интерфейс: %v", err))
	}
}
