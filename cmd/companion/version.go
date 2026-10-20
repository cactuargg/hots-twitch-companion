package main

const appVersion = "0.10.1"

// Release builds may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string
