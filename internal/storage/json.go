// Package storage implements small, explicit, read-only and durable JSON
// operations for the application's local data files.
package storage

import (
	"encoding/json"
	"errors"
	"os"
)

// SaveJSON writes an atomically renamed, fsynced temporary file. Errors
// return without committing a new state, so irreversible effects stay gated.
func SaveJSON(path string, v any) error {

	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		return e
	}
	tmp := path + ".new"
	f, e := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if e != nil { return e }
	if _, e = f.Write(b); e == nil { e = f.Sync() }
	closeErr := f.Close()
	if e != nil { return e }
	if closeErr != nil { return closeErr }
	return os.Rename(tmp, path)
}
func LoadJSON(path string, v any) error {
	b, e := os.ReadFile(path)
	if errors.Is(e, os.ErrNotExist) {
		return nil
	}
	if e != nil {
		return e
	}
	return json.Unmarshal(b, v)
}
