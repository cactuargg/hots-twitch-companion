package main

const appVersion = "0.9.9"

// Release builds may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string
