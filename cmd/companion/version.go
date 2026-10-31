package main

const appVersion = "0.14.0"

// Release builds may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string
