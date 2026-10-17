//go:build !windows

package main

import "errors"

func protect(b []byte) ([]byte, error)   { return nil, errors.New("DPAPI requires Windows") }
func unprotect(b []byte) ([]byte, error) { return nil, errors.New("DPAPI requires Windows") }
