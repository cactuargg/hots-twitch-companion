package twitch

import (
	"context"
	"errors"
	"fmt"
	"hots-twitch-companion/internal/i18n"
	"hots-twitch-companion/internal/prediction"
	"hots-twitch-companion/internal/storage/credentials"
	"hots-twitch-companion/internal/twitch/transport"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"
)

const Scope = "channel:manage:predictions"
const AuthURL = "https://id.twitch.tv/oauth2"
const APIURL = "https://api.twitch.tv/helix"

type Client struct {
	// Credential mutations are serialized separately from status snapshots.
	refreshMu            sync.Mutex
	mu                   sync.Mutex
	clientID             string
	lang                 string
	tokens               TokenStore
	request              JSONRequest
	token                Token
	pendingToken         *Token
	broadcasterID, login string
}

// TokenStore owns encrypted persistence; tests can inject an in-memory store.
type TokenStore interface {
	Load(*Token) error
	Save(Token) error
}
type FileTokens struct{ Path string }

func (f FileTokens) Load(t *Token) error { return credentials.LoadJSON(f.Path, t) }
func (f FileTokens) Save(t Token) error  { return credentials.SaveJSON(f.Path, t) }

func New(clientID, lang string, tokens TokenStore, request JSONRequest) (*Client, error) {
	t := &Client{clientID: clientID, lang: i18n.Normalize(lang), tokens: tokens, request: request}
	if err := tokens.Load(&t.token); err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	return t, nil
}

// HTTPRequest binds a transport to one client rather than mutable global state.
func HTTPRequest(client *http.Client) JSONRequest {
	return func(ctx context.Context, method, endpoint string, form url.Values, body any, token, clientID string, dst any) error {
		return transport.RequestJSON(ctx, client, method, endpoint, form, body, token, clientID, dst, transport.ResponseHTTPError)
	}
}

type Connection struct {
	HasToken             bool
	Login, BroadcasterID string
}

func (t *Client) Connection() Connection {
	t.mu.Lock()
	defer t.mu.Unlock()
	return Connection{t.token.AccessToken != "", t.login, t.broadcasterID}
}
func (t *Client) ClientID() string { return t.clientID }
func (t *Client) SetLanguage(lang string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lang = i18n.Normalize(lang)
}

// AcceptValidatedToken commits credentials only after the application has
// checked prediction ownership against the validated identity.
func (t *Client) AcceptValidatedToken(token Token, identity Identity) error {
	t.refreshMu.Lock()
	defer t.refreshMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	if err := t.saveToken(token); err != nil {
		return err
	}
	t.login, t.broadcasterID = identity.Login, identity.UserID
	return nil
}
func (t *Client) ValidateIdentity(ctx context.Context, token string) (Identity, error) {
	return ValidateIdentity(ctx, token, t.clientID, t.language(), t.request)
}
func (t *Client) StartDevice(ctx context.Context) (DeviceResponse, error) {
	return StartDevice(ctx, t.clientID, AuthURL, Scope, t.request)
}
func (t *Client) PollDevice(ctx context.Context, d DeviceResponse, log func(string)) (Token, error) {
	return PollDevice(ctx, t.clientID, t.language(), AuthURL, Scope, d, log, t.request, func(lang, key string) string { return i18n.Text(lang, key) })
}

// language may change while monitoring is running; clientID and token
// ownership remain stable until the OAuth application actually changes.
func (t *Client) language() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lang
}

func (t *Client) saveToken(token Token) error {
	if err := t.tokens.Save(token); err != nil {
		return err
	}
	t.token = token
	t.pendingToken = nil
	return nil
}

// Keep a remotely rotated pair until it is durable. Callers must not use it
// for API operations or retry the old refresh token while saving fails.
func (t *Client) persistPendingToken() error {
	if t.pendingToken == nil {
		return nil
	}
	if err := t.saveToken(*t.pendingToken); err != nil {
		return fmt.Errorf("%s: %w", i18n.Text(t.lang, "twitch.token_save_error"), err)
	}
	return nil
}
func (t *Client) tokenFor(ctx context.Context) (access string, err error) {
	t.refreshMu.Lock()
	defer t.refreshMu.Unlock()
	t.mu.Lock()
	defer t.mu.Unlock()
	defer func() {
		if err != nil {
			// A cached login is not a current connection after a failed refresh.
			t.login, t.broadcasterID = "", ""
		}
	}()
	if err := t.persistPendingToken(); err != nil {
		return "", err
	}
	if t.token.AccessToken == "" {
		return "", errors.New(i18n.Text(t.lang, "twitch.connect_first"))
	}
	if time.Since(time.Unix(t.token.SavedAt, 0)) < time.Duration(t.token.ExpiresIn-300)*time.Second {
		return t.token.AccessToken, nil
	}
	if t.token.RefreshToken == "" {
		return "", errors.New(i18n.Text(t.lang, "twitch.token_expired"))
	}
	form := url.Values{"client_id": {t.clientID}, "grant_type": {"refresh_token"}, "refresh_token": {t.token.RefreshToken}}
	var token Token
	// Keep refresh serialization while allowing Connection and language reads.
	t.mu.Unlock()
	requestErr := t.request(ctx, "POST", AuthURL+"/token", form, nil, "", "", &token)
	t.mu.Lock()
	if e := requestErr; e != nil {
		return "", errors.New(i18n.Text(t.lang, "twitch.refresh_error", e.Error()))
	}
	if token.AccessToken == "" || token.RefreshToken == "" {
		return "", errors.New(i18n.Text(t.lang, "twitch.incomplete_tokens"))
	}
	token.SavedAt = time.Now().Unix()
	t.pendingToken = &token
	if e := t.persistPendingToken(); e != nil {
		return "", e
	}
	return token.AccessToken, nil
}

// validateIdentity verifies candidate credentials without changing
// the persisted token. A device authorization must not replace the currently
// connected broadcaster before this check completes.
func ValidateIdentity(ctx context.Context, accessToken, clientID, lang string, request JSONRequest) (Identity, error) {
	var me Identity
	if err := request(ctx, "GET", AuthURL+"/validate", nil, nil, accessToken, "", &me); err != nil {
		return me, err
	}
	switch IdentityIssue(me, clientID, Scope) {
	case "wrong_client":
		return me, errors.New(i18n.Text(lang, "twitch.wrong_client"))
	case "no_scope":
		return me, errors.New(i18n.Text(lang, "twitch.no_scope"))
	case "missing_broadcaster":
		return me, errors.New("Client validation returned no broadcaster ID")
	}
	return me, nil
}

func (t *Client) Identity(ctx context.Context) (Identity, error) {
	token, err := t.tokenFor(ctx)
	if err != nil {
		return Identity{}, err
	}
	me, err := t.ValidateIdentity(ctx, token)
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

func (t *Client) req(ctx context.Context, method, route string, body any, dst any) error {
	token, e := t.tokenFor(ctx)
	if e != nil {
		return e
	}
	e = t.request(ctx, method, APIURL+route, nil, body, token, t.clientID, dst)
	var httpError *transport.HTTPError
	if errors.As(e, &httpError) && httpError.Status == 401 {
		t.mu.Lock()
		if t.token.AccessToken == token {
			t.token.SavedAt = 0
		}
		t.mu.Unlock()
		token, e = t.tokenFor(ctx)
		if e != nil {
			return e
		}
		return t.request(ctx, method, APIURL+route, nil, body, token, t.clientID, dst)
	}
	return e
}
func (t *Client) Prediction(ctx context.Context, broadcaster, id string) (*APIResponse, error) {
	route := PredictionRoute(broadcaster, id)
	var d APIResponse
	e := t.req(ctx, "GET", route, nil, &d)
	return &d, e
}

// preflightCreate performs all fallible read-only checks before a lobby is
// consumed. A failure here is safe to retry: no creation POST has been sent.
func (t *Client) PreflightCreate(ctx context.Context) (Identity, error) {
	me, err := t.Identity(ctx)
	if err != nil {
		return Identity{}, err
	}
	previous, err := t.Prediction(ctx, me.UserID, "")
	if err != nil {
		return Identity{}, err
	}
	if HasOpenPrediction(previous) {
		return Identity{}, errors.New(i18n.Text(t.language(), "twitch.prediction_exists"))
	}
	return me, nil
}

// createPrepared must only run after the caller durably records the creation
// attempt. A lost POST response must never trigger a second POST.
func (t *Client) CreatePrepared(ctx context.Context, broadcasterID string, windowMinutes int, title, winTitle, lossTitle string) (*prediction.Prediction, error) {
	payload, err := CreationRequest(broadcasterID, windowMinutes, title, winTitle, lossTitle)
	if err != nil {
		return nil, errors.New(i18n.Text(t.language(), "twitch.outcomes_invalid"))
	}
	var response APIResponse
	if err := t.req(ctx, "POST", "/predictions", payload, &response); err != nil {
		return nil, err
	}
	confirmed, err := ParseCreatedPrediction(&response, broadcasterID, winTitle, lossTitle, time.Now().UnixMilli())
	if errors.Is(err, ErrMalformedCreateResponse) {
		return nil, errors.New(i18n.Text(t.language(), "twitch.create_bad_response"))
	}
	if err != nil {
		return nil, errors.New(i18n.Text(t.language(), "twitch.outcomes_invalid"))
	}
	return confirmed, nil
}

func (t *Client) Create(ctx context.Context, windowMinutes int, title, winTitle, lossTitle string) (*prediction.Prediction, error) {
	if winTitle == lossTitle {
		return nil, errors.New(i18n.Text(t.language(), "twitch.outcomes_invalid"))
	}
	me, err := t.PreflightCreate(ctx)
	if err != nil {
		return nil, err
	}
	return t.CreatePrepared(ctx, me.UserID, windowMinutes, title, winTitle, lossTitle)
}

// End reconciles the exact remote prediction before applying an outcome.
func (t *Client) End(ctx context.Context, p *prediction.Prediction, outcome string) error {
	return FinishPrediction(ctx, p, outcome,
		func(ctx context.Context, broadcaster, id string) (*APIResponse, error) {
			return t.Prediction(ctx, broadcaster, id)
		},
		func(ctx context.Context, body map[string]any) (*APIResponse, error) {
			var result APIResponse
			err := t.req(ctx, "PATCH", "/predictions", body, &result)
			return &result, err
		})
}
