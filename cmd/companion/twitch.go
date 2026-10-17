package main

import (
 "net/url"
 twitchpkg "hots-twitch-companion/internal/twitch"
	transport "hots-twitch-companion/internal/twitch/transport"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"
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
func startDevice(ctx context.Context,clientID string)(DeviceResponse,error){
 return twitchpkg.StartDevice(ctx,clientID,authURL,scope,requestJSON)
}
func devicePollRetry(interval time.Duration,err error)(time.Duration,bool){
 return twitchpkg.DevicePollRetry(interval,err)
}
func pollDevice(ctx context.Context,clientID,lang string,d DeviceResponse,log func(string))(Token,error){
 return twitchpkg.PollDevice(ctx,clientID,lang,authURL,scope,d,log,requestJSON,
  func(language,key string) string {return tr(language,key)})
}

type Twitch struct {
	mu                   sync.Mutex
	clientID             string
	lang                 string
	p                    paths
	token                Token
	broadcasterID, login string
}

func newTwitch(p paths, clientID, lang string) (*Twitch, error) {
	t := &Twitch{clientID: clientID, lang: normalizeLang(lang), p: p}
	b, e := os.ReadFile(p.secret)
	if errors.Is(e, os.ErrNotExist) {
		return t, nil
	}
	if e != nil {
		return nil, e
	}
	clear, e := unprotect(b)
	if e != nil {
		return nil, fmt.Errorf("Windows DPAPI: %w", e)
	}
	if e = json.Unmarshal(clear, &t.token); e != nil {
		return nil, e
	}
	return t, nil
}
// language may change while monitoring is running; clientID and token
// ownership remain stable until the OAuth application actually changes.
func (t *Twitch) language() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lang
}

func (t *Twitch) saveToken(token Token) error {
	b, e := json.Marshal(token)
	if e != nil {
		return e
	}
	sealed, e := protect(b)
	if e != nil {
		return e
	}
	if e = os.WriteFile(t.p.secret+".new", sealed, 0600); e != nil {
		return e
	}
	if e = os.Rename(t.p.secret+".new", t.p.secret); e != nil {
		return e
	}
	t.token = token
	return nil
}
func (t *Twitch) tokenFor(ctx context.Context) (string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.token.AccessToken == "" {
		return "", errors.New(tr(t.lang, "twitch.connect_first"))
	}
	if time.Since(time.Unix(t.token.SavedAt, 0)) < time.Duration(t.token.ExpiresIn-300)*time.Second {
		return t.token.AccessToken, nil
	}
	if t.token.RefreshToken == "" {
		return "", errors.New(tr(t.lang, "twitch.token_expired"))
	}
	form := url.Values{"client_id": {t.clientID}, "grant_type": {"refresh_token"}, "refresh_token": {t.token.RefreshToken}}
	var token Token
	if e := requestJSON(ctx, "POST", authURL+"/token", form, nil, "", "", &token); e != nil {
		return "", errors.New(tr(t.lang, "twitch.refresh_error", e.Error()))
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		return "", errors.New(tr(t.lang, "twitch.incomplete_tokens"))
	}
	token.SavedAt = time.Now().Unix()
	if e := t.saveToken(token); e != nil {
		return "", e
	}
	return token.AccessToken, nil
}
// validateTwitchIdentity verifies candidate credentials without changing
// the persisted token. A device authorization must not replace the currently
// connected broadcaster before this check completes.
func validateTwitchIdentity(ctx context.Context, accessToken, clientID, lang string) (TwitchIdentity, error) {
	var me TwitchIdentity
	if err := requestJSON(ctx, "GET", authURL+"/validate", nil, nil, accessToken, "", &me); err != nil {
		return me, err
	}
	if me.ClientID != clientID {
		return me, errors.New(tr(lang, "twitch.wrong_client"))
	}
	valid := false
	for _, v := range me.Scopes {
		if v == scope {
			valid = true
		}
	}
	if !valid {
		return me, errors.New(tr(lang, "twitch.no_scope"))
	}
	if me.UserID == "" {
		return me, errors.New("Twitch validation returned no broadcaster ID")
	}
	return me, nil
}

func (t *Twitch) identity(ctx context.Context) (TwitchIdentity, error) {
	token, err := t.tokenFor(ctx)
	if err != nil {
		return TwitchIdentity{}, err
	}
	me, err := validateTwitchIdentity(ctx, token, t.clientID, t.language())
	if err != nil {
		// Failed validation must not leave a stale "connected" identity
		// visible after revoked credentials or a different OAuth client.
		t.mu.Lock()
		if t.token.AccessToken == token {
			t.login, t.broadcasterID = "", ""
		}
		t.mu.Unlock()
		return me, err
	}
	t.mu.Lock()
	if t.token.AccessToken == token {
		t.broadcasterID = me.UserID
		t.login = me.Login
	}
	t.mu.Unlock()
	return me, nil
}

func (t *Twitch) req(ctx context.Context, method, route string, body any, dst any) error {
	token, e := t.tokenFor(ctx)
	if e != nil {
		return e
	}
	e = requestJSON(ctx, method, apiURL+route, nil, body, token, t.clientID, dst)
	var httpError *twitchHTTPError
	if errors.As(e, &httpError) && httpError.Status == 401 {
		t.mu.Lock()
		t.token.SavedAt = 0
		t.mu.Unlock()
		token, e = t.tokenFor(ctx)
		if e != nil {
			return e
		}
		return requestJSON(ctx, method, apiURL+route, nil, body, token, t.clientID, dst)
	}
	return e
}
func (t *Twitch) prediction(ctx context.Context, broadcaster, id string) (*apiResponse, error) {
	route := "/predictions?broadcaster_id=" + url.QueryEscape(broadcaster) + "&first=5"
	if id != "" {
		route = "/predictions?broadcaster_id=" + url.QueryEscape(broadcaster) + "&id=" + url.QueryEscape(id)
	}
	var d apiResponse
	e := t.req(ctx, "GET", route, nil, &d)
	return &d, e
}
// preflightCreate performs all fallible read-only checks before a lobby is
// consumed. A failure here is safe to retry: no creation POST has been sent.
func (t *Twitch) preflightCreate(ctx context.Context) (TwitchIdentity, error) {
	me, err := t.identity(ctx)
	if err != nil {
		return TwitchIdentity{}, err
	}
	previous, err := t.prediction(ctx, me.UserID, "")
	if err != nil {
		return TwitchIdentity{}, err
	}
	for _, v := range previous.Data {
		if v.Status == "ACTIVE" || v.Status == "LOCKED" {
			return TwitchIdentity{}, errors.New(tr(t.language(), "twitch.prediction_exists"))
		}
	}
	return me, nil
}

// createPrepared must only run after the caller durably records the creation
// attempt. A lost POST response must never trigger a second POST.
func (t *Twitch) createPrepared(ctx context.Context, broadcasterID string, windowMinutes int, title, winTitle, lossTitle string) (*Prediction, error) {
	if broadcasterID == "" || winTitle == lossTitle {
		return nil, errors.New(tr(t.language(), "twitch.outcomes_invalid"))
	}
	var d apiResponse
	err := t.req(ctx, "POST", "/predictions", map[string]any{
		"broadcaster_id": broadcasterID,
		"title": title,
		"outcomes": []map[string]string{{"title": winTitle}, {"title": lossTitle}},
		"prediction_window": windowMinutes * 60,
	}, &d)
	if err != nil {
		return nil, err
	}
	if len(d.Data) != 1 || len(d.Data[0].Outcomes) != 2 || d.Data[0].Status != "ACTIVE" {
		return nil, errors.New(tr(t.language(), "twitch.create_bad_response"))
	}
	p := &Prediction{ID: d.Data[0].ID, BroadcasterID: broadcasterID, StartedAt: time.Now().UnixMilli()}
	for _, outcome := range d.Data[0].Outcomes {
		if outcome.Title == winTitle {
			p.WinID = outcome.ID
		}
		if outcome.Title == lossTitle {
			p.LossID = outcome.ID
		}
	}
	if p.ID == "" || p.WinID == "" || p.LossID == "" || p.WinID == p.LossID {
		return nil, errors.New(tr(t.language(), "twitch.outcomes_invalid"))
	}
	return p, nil
}

func (t *Twitch) create(ctx context.Context, windowMinutes int, title, winTitle, lossTitle string) (*Prediction, error) {
	if winTitle == lossTitle {
		return nil, errors.New(tr(t.language(), "twitch.outcomes_invalid"))
	}
	me, err := t.preflightCreate(ctx)
	if err != nil {
		return nil, err
	}
	return t.createPrepared(ctx, me.UserID, windowMinutes, title, winTitle, lossTitle)
}
func (t *Twitch) end(ctx context.Context, p *Prediction, outcome string) error {
 return t.finishPrediction(ctx, p, outcome)
}

