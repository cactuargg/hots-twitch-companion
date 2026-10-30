package main

const appVersion = "0.13.0"

// Release builds may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string
