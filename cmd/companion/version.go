package main

const appVersion = "0.9.10"

// Release builds may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string
