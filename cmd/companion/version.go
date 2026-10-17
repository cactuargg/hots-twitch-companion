package main

const appVersion = "0.9.6"

// Release builds may embed a public OAuth Client ID via -ldflags.
var defaultTwitchClientID string
