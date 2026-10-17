//go:build windows && !wails

package main

import (
	"hots-twitch-companion/internal/dashboard"
	"os"
	"os/exec"
	"path/filepath"
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
		cmd := exec.Command(path, "--app="+url, "--window-size=1024,800", "--no-first-run")
		if err := cmd.Start(); err == nil {
			go cmd.Wait()
			return nil
		}
	}
	// Fallback: default browser opens the same local UI without installing anything.
	return exec.Command("rundll32.exe", "url.dll,FileProtocolHandler", url).Start()
}
func main() {
	runWindowsApplication(func(d *dashboard.Dashboard) error { return d.RunDesktop(launchDesktop) })
}
