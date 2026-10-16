//go:build windows && wails

package main

import (
	"context"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

func main() { runWindowsApplication(runNativeDesktop) }

func runNativeDesktop(d *dashboard) error {
	handler, err := d.nativeAssetHandler()
	if err != nil {
		return err
	}
	// No Go methods are bound to JavaScript. Existing HTTP-shaped API guards,
	// JSON requests and payout confirmations are shared with the browser build.
	return wails.Run(&options.App{
		Title: "HotS Twitch Companion — Desktop Pilot", Width: 1024, Height: 800, MinWidth: 800, MinHeight: 600,
		BackgroundColour: options.NewRGB(12, 16, 29),
		AssetServer:      &assetserver.Options{Handler: handler},
		Windows:          &windows.Options{WebviewUserDataPath: filepath.Join(d.p.dir, "webview2"), Theme: windows.Dark},
		OnStartup: func(ctx context.Context) {
			go func() {
				select {
				case <-d.shutdown:
					wailsruntime.Quit(ctx)
				case <-ctx.Done():
				}
			}()
		},
		OnBeforeClose: func(ctx context.Context) bool {
			if !d.closeRequested() {
				return false
			}
			choice, err := wailsruntime.MessageDialog(ctx, wailsruntime.MessageDialogOptions{
				Type: wailsruntime.QuestionDialog, Title: "HotS Twitch Companion", Message: d.tr("shell.close_active"),
				Buttons: []string{"Yes", "No"}, DefaultButton: "No", CancelButton: "No",
			})
			return err != nil || choice != "Yes"
		},
		OnShutdown: func(context.Context) { d.engine.stopWatch(); d.shutdownOnce.Do(func() { close(d.shutdown) }) },
	})
}
