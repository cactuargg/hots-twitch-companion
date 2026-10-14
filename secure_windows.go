package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

type dataBlob struct {
	Size uint32
	Data *byte
}

var crypt32 = syscall.NewLazyDLL("crypt32.dll")
var protectProc = crypt32.NewProc("CryptProtectData")
var unprotectProc = crypt32.NewProc("CryptUnprotectData")
var localFree = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")

func blobFor(b []byte) dataBlob {
	if len(b) == 0 {
		return dataBlob{}
	}
	return dataBlob{uint32(len(b)), &b[0]}
}
func dpapi(b []byte, encrypt bool) ([]byte, error) {
	src := blobFor(b)
	var dst dataBlob
	proc := unprotectProc
	if encrypt {
		proc = protectProc
	}
	var ok uintptr
	var err error
	if encrypt {
		ok, _, err = proc.Call(uintptr(unsafe.Pointer(&src)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&dst)))
	} else {
		ok, _, err = proc.Call(uintptr(unsafe.Pointer(&src)), 0, 0, 0, 0, 1, uintptr(unsafe.Pointer(&dst)))
	}
	if ok == 0 {
		return nil, fmt.Errorf("Windows DPAPI: %v", err)
	}
	defer localFree.Call(uintptr(unsafe.Pointer(dst.Data)))
	if dst.Size == 0 {
		return []byte{}, nil
	}
	bs := unsafe.Slice(dst.Data, int(dst.Size))
	out := append([]byte(nil), bs...)
	for i := range bs {
		bs[i] = 0
	}
	return out, nil
}
func protect(b []byte) ([]byte, error)   { return dpapi(b, true) }
func unprotect(b []byte) ([]byte, error) { return dpapi(b, false) }
