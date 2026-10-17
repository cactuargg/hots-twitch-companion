// Package twitch owns the synchronized OAuth and Helix client, protocol
// records, Device Flow and transport-independent prediction reconciliation.
package twitch

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"
)

// JSONRequest is injected by the caller, so this package can be tested
// without contacting Twitch or sharing a global HTTP client.
type JSONRequest func(context.Context, string, string, url.Values, any, string, string, any) error

func StartDevice(ctx context.Context, clientID, authURL, scope string, requestJSON JSONRequest) (DeviceResponse, error) {
	var d DeviceResponse
	e := requestJSON(ctx, "POST", authURL+"/device", url.Values{"client_id": {clientID}, "scopes": {scope}}, nil, "", "", &d)
	return d, e
}

// Twitch asks Device Flow clients to increase their polling delay by five
// seconds each time the OAuth endpoint returns slow_down.
func DevicePollRetry(interval time.Duration, err error) (time.Duration, bool) {
	if err == nil {
		return interval, false
	}
	if strings.Contains(err.Error(), "slow_down") {
		return interval + 5*time.Second, true
	}
	if strings.Contains(err.Error(), "authorization_pending") {
		return interval, true
	}
	return interval, false
}

func PollDevice(ctx context.Context, clientID, lang, authURL, scope string, d DeviceResponse, log func(string), requestJSON JSONRequest, translate func(string, string) string) (Token, error) {
	if d.Interval < 5 {
		d.Interval = 5
	}
	interval := time.Duration(d.Interval) * time.Second
	deadline := time.NewTimer(time.Duration(d.ExpiresIn) * time.Second)
	defer deadline.Stop()
	wait := time.NewTimer(interval)
	defer wait.Stop()
	for {
		select {
		case <-ctx.Done():
			return Token{}, ctx.Err()
		case <-deadline.C:
			return Token{}, errors.New(translate(lang, "auth.code_expired"))
		case <-wait.C:
			var t Token
			form := url.Values{"client_id": {clientID}, "scopes": {scope}, "device_code": {d.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}}
			err := requestJSON(ctx, "POST", authURL+"/token", form, nil, "", "", &t)
			if err == nil {
				if t.AccessToken == "" || t.RefreshToken == "" {
					return Token{}, errors.New(translate(lang, "auth.no_tokens"))
				}
				t.SavedAt = time.Now().Unix()
				return t, nil
			}
			next, retry := DevicePollRetry(interval, err)
			if !retry {
				return Token{}, err
			}
			interval = next
			wait.Reset(interval)
		}
	}
}
