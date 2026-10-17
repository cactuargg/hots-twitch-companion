//go:build !windows

package credentials

import "errors"

func Protect(b []byte) ([]byte, error)   { return nil, errors.New("DPAPI requires Windows") }
func Unprotect(b []byte) ([]byte, error) { return nil, errors.New("DPAPI requires Windows") }
