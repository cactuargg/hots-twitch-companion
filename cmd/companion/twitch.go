package main

import (
	"context"
	twitchpkg "hots-twitch-companion/internal/twitch"
	transport "hots-twitch-companion/internal/twitch/transport"
	"net/http"
	"net/url"
	"time"
)

// Twitch OAuth, Helix transport and credential handling live here.
// Engine, configuration and replay scanning remain in core.go.

var httpClient = &http.Client{Timeout: 20 * time.Second}

// The application owns its HTTP client and typed retryable Twitch errors.
// The transport package only performs bounded request/response serialization.
func requestJSON(ctx context.Context, method, urlPath string, form url.Values, body any, token, clientID string, dst any) error {
	return transport.RequestJSON(ctx, httpClient, method, urlPath, form, body, token, clientID, dst, responseHTTPError)
}
func startDevice(ctx context.Context, clientID string) (DeviceResponse, error) {
	return twitchpkg.StartDevice(ctx, clientID, authURL, scope, requestJSON)
}
func devicePollRetry(interval time.Duration, err error) (time.Duration, bool) {
	return twitchpkg.DevicePollRetry(interval, err)
}
func pollDevice(ctx context.Context, clientID, lang string, d DeviceResponse, log func(string)) (Token, error) {
	return twitchpkg.PollDevice(ctx, clientID, lang, authURL, scope, d, log, requestJSON,
		func(language, key string) string { return tr(language, key) })
}

type Twitch = twitchpkg.Client

func newTwitch(p paths, clientID, lang string) (*Twitch, error) {
	return twitchpkg.New(clientID, lang, twitchpkg.FileTokens{Path: p.secret}, requestJSON)
}
func validateTwitchIdentity(ctx context.Context, token, clientID, lang string) (TwitchIdentity, error) {
	return twitchpkg.ValidateIdentity(ctx, token, clientID, lang, requestJSON)
}
