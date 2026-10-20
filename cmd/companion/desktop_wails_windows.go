//go:build windows && wails

package main

import (
	"context"
	"errors"
	"hots-twitch-companion/internal/dashboard"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

const updateEdition = "desktop"

func main() { runWindowsApplication(runNativeDesktop) }

func runNativeDesktop(d *dashboard.Dashboard) error {
	var mu sync.Mutex
	var appContext context.Context
	var tray *desktopTray
	d.SetHideToTray(func() error {
		mu.Lock()
		defer mu.Unlock()
		if appContext == nil || tray == nil || !tray.ready.Load() {
			return errors.New(d.Text("shell.tray_unavailable"))
		}
		wailsruntime.WindowHide(appContext)
		return nil
	})
	handler, err := d.NativeAssetHandler()
	if err != nil {
		return err
	}
	// No Go methods are bound to JavaScript. Existing HTTP-shaped API guards,
	// JSON requests and payout confirmations are shared with the browser build.
	return wails.Run(&options.App{
		Title: "HotS Twitch Companion", Width: 1100, Height: 800, MinWidth: 480, MinHeight: 540,
		Frameless:        true,
		BackgroundColour: options.NewRGB(12, 16, 29),
		AssetServer:      &assetserver.Options{Handler: handler},
		Windows:          &windows.Options{WebviewUserDataPath: filepath.Join(d.ProfileDir(), "webview2"), Theme: windows.Dark},
		OnStartup: func(ctx context.Context) {
			mu.Lock()
			appContext = ctx
			tray = startDesktopTray(ctx, d)
			mu.Unlock()
			go func() {
				select {
				case <-d.Done():
					wailsruntime.Quit(ctx)
				case <-ctx.Done():
				}
			}()
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if !d.CloseRequested() {
				return false
			}
			choice, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
				Type: wailsruntime.QuestionDialog, Title: "HotS Twitch Companion", Message: d.Text("shell.close_active"),
				Buttons: []string{"Yes", "No"}, DefaultButton: "No", CancelButton: "No",
			})
			return err != nil || choice != "Yes"
		},
		OnShutdown: func(context.Context) {
			mu.Lock()
			if tray != nil {
				tray.stop()
			}
			mu.Unlock()
			d.Close()
		},
	})
}
