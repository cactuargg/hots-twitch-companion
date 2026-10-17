//go:build windows && wails

package main

import (
	"context"
	appassets "hots-twitch-companion/assets"
	"runtime"
	"sync"
	"sync/atomic"

	"fyne.io/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

var desktopIcon = appassets.Icon

type desktopTray struct {
	ready    atomic.Bool
	stopping chan struct{}
	once     sync.Once
}

func (t *desktopTray) stop() {
	t.once.Do(func() {
		close(t.stopping)
		if t.ready.Load() {
			systray.Quit()
		}
	})
}

func startDesktopTray(ctx context.Context, d *dashboard) *desktopTray {
	t := &desktopTray{stopping: make(chan struct{})}
	// The Windows tray owns a separate message loop; Wails keeps the main thread.
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		systray.Run(func() {
			systray.SetIcon(desktopIcon)
			systray.SetTooltip("HotS Twitch Companion")
			show := func() { wailsruntime.WindowShow(ctx); wailsruntime.WindowUnminimise(ctx) }
			systray.SetOnTapped(show)
			open := systray.AddMenuItem(d.tr("shell.open"), "")
			exit := systray.AddMenuItem(d.tr("shell.exit"), "")
			t.ready.Store(true)
			select {
			case <-t.stopping:
				systray.Quit()
				return
			default:
			}
			go func() {
				for {
					select {
					case <-open.ClickedCh:
						show()
					case <-exit.ClickedCh:
						show()
						// Quit passes through the same OnBeforeClose confirmation as the X.
						wailsruntime.Quit(ctx)
					case <-t.stopping:
						return
					case <-ctx.Done():
						return
					}
				}
			}()
		}, func() { t.ready.Store(false) })
	}()
	return t
}
