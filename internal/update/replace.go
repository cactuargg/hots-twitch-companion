package update

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Replace runs only in the detached helper after the old process has exited.
// The backup stays on the same filesystem. Launch failure restores the old EXE.
// The profile is never read, reset or copied by this operation.
func Replace(target string, p Prepared, start func(string) error) error {
	if !filepath.IsAbs(target) || !strings.EqualFold(filepath.Ext(target), ".exe") || filepath.Dir(filepath.Dir(p.Path)) != filepath.Dir(target) {
		return errors.New("invalid portable replacement paths")
	}
	hash, err := VerifyExecutable(p.Path)
	if err != nil {
		return err
	}
	if hash != p.SHA256 {
		return errors.New("staged executable changed")
	}
	st, err := os.Lstat(target)
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return errors.New("target is not a regular executable")
	}
	backup := filepath.Join(filepath.Dir(p.Path), "previous.exe")
	if _, err = os.Lstat(backup); !os.IsNotExist(err) {
		return errors.New("update backup already exists")
	}
	// Keep the current pathname intact until the verified replacement is ready.
	// Rename replaces it in one filesystem operation; the synced backup is
	// already durable, rather than creating a gap by moving the old EXE away.
	if err = copyBackup(target, backup); err != nil {
		return err
	}
	if err = os.Rename(p.Path, target); err != nil {
		return err
	}
	if err = start(target); err != nil {
		if restoreErr := os.Rename(backup, target); restoreErr != nil {
			return fmt.Errorf("launch: %v; restore: %w", err, restoreErr)
		}
		if restartErr := start(target); restartErr != nil {
			return fmt.Errorf("restored old executable, but restart failed: %w", restartErr)
		}
		return fmt.Errorf("new executable could not start; restored previous version: %w", err)
	}
	return nil
}

func copyBackup(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0700)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
