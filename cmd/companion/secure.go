package main

import "hots-twitch-companion/internal/storage/credentials"

// Compatibility functions for the OAuth client until it moves into the
// internal/twitch package. DPAPI is implemented in storage/credentials.
func protect(b []byte) ([]byte,error) {return credentials.Protect(b)}
func unprotect(b []byte) ([]byte,error) {return credentials.Unprotect(b)}
