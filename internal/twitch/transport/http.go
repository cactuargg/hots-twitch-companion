// Package transport handles bounded Twitch OAuth and Helix HTTP requests.
// It has no dependency on the application's token storage, Engine or UI.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func RequestJSON(ctx context.Context, client *http.Client, method, urlPath string, form url.Values, body any, token, clientID string, dst any, onHTTPError func(*http.Response,string) error) error {
	var r io.Reader
	if form != nil {
		r = strings.NewReader(form.Encode())
	} else if body != nil {
		b, e := json.Marshal(body)
		if e != nil {
			return e
		}
		r = bytes.NewReader(b)
	}
	req, e := http.NewRequestWithContext(ctx, method, urlPath, r)
	if e != nil {
		return e
	}
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		if strings.HasSuffix(urlPath, "/validate") {
			req.Header.Set("Authorization", "OAuth "+token)
		} else {
			req.Header.Set("Authorization", "Bearer "+token)
		}
	}
	if clientID != "" {
		req.Header.Set("Client-Id", clientID)
	}
	res, e := client.Do(req)
	if e != nil {
		return e
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if e != nil {
		return e
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		var errBody struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		_ = json.Unmarshal(raw, &errBody)
		msg := errBody.Message
		if msg == "" {
			msg = errBody.Error
		}
		if msg == "" {
			msg = res.Status
		}
		return onHTTPError(res, msg)
	}
	if dst != nil && len(raw) > 0 {
		return json.Unmarshal(raw, dst)
	}
	return nil
}
