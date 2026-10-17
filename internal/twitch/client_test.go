package twitch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

type memoryTokens struct {
	token Token
	fail  bool
}

func (s *memoryTokens) Load(t *Token) error { *t = s.token; return nil }
func (s *memoryTokens) Save(t Token) error {
	if s.fail {
		return errors.New("disk full")
	}
	s.token = t
	return nil
}

type clientTransport func(*http.Request) (*http.Response, error)

func TestClientStatusRemainsAvailableDuringSerializedRefresh(t *testing.T) {
	entered, release := make(chan struct{}), make(chan struct{})
	defer close(release)
	refreshes := 0
	request := func(ctx context.Context, method, endpoint string, form url.Values, body any, token, clientID string, dst any) error {
		refreshes++
		if refreshes != 1 {
			t.Error("concurrent callers refreshed the same credentials twice")
			return errors.New("duplicate refresh")
		}
		close(entered)
		<-release
		*dst.(*Token) = Token{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 3600}
		return nil
	}
	c, err := New("client", "en", &memoryTokens{token: Token{AccessToken: "expired", RefreshToken: "old"}}, request)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 2)
	refresh := func() {
		access, err := c.tokenFor(context.Background())
		if err == nil && access != "fresh" {
			err = errors.New("wrong access token")
		}
		done <- err
	}
	go refresh()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("refresh never started")
	}
	go refresh()
	snapshot := make(chan Connection, 1)
	go func() { c.SetLanguage("ru"); snapshot <- c.Connection() }()
	select {
	case connection := <-snapshot:
		if !connection.HasToken {
			t.Fatal("snapshot lost credentials")
		}
	case <-time.After(time.Second):
		t.Fatal("status or language change waited for the network")
	}
	// Release the request without closing twice in the deferred cleanup.
	release <- struct{}{}
	for range 2 {
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(time.Second):
			t.Fatal("refresh did not finish")
		}
	}
}

func TestClientRetriesSavingRotatedTokensBeforeAPIRequests(t *testing.T) {
	store := &memoryTokens{token: Token{AccessToken: "expired", RefreshToken: "old"}, fail: true}
	refreshes, apiCalls := 0, 0
	request := func(ctx context.Context, method, endpoint string, form url.Values, body any, token, clientID string, dst any) error {
		if endpoint == AuthURL+"/token" {
			refreshes++
			if refreshes != 1 || form.Get("refresh_token") != "old" {
				t.Fatal("reused a refresh token instead of retrying persistence")
			}
			*dst.(*Token) = Token{AccessToken: "fresh", RefreshToken: "rotated", ExpiresIn: 3600}
			return nil
		}
		apiCalls++
		if token != "fresh" || store.token.RefreshToken != "rotated" {
			t.Fatal("API request used credentials before they were saved")
		}
		return nil
	}
	c, err := New("client", "en", store, request)
	if err != nil {
		t.Fatal(err)
	}
	c.login, c.broadcasterID = "channel", "123"
	for range 2 {
		if err := c.req(context.Background(), "POST", "/predictions", nil, nil); err == nil || !strings.Contains(err.Error(), "Could not save refreshed Twitch credentials") {
			t.Fatalf("missing actionable persistence error: %v", err)
		}
	}
	if refreshes != 1 || apiCalls != 0 || c.Connection().Login != "" {
		t.Fatal("failed persistence did not pause access")
	}
	store.fail = false
	if err := c.req(context.Background(), "POST", "/predictions", nil, nil); err != nil {
		t.Fatal(err)
	}
	if refreshes != 1 || apiCalls != 1 {
		t.Fatal("recovery refreshed again or failed to resume API access")
	}
	// A restart must recover the same durable credentials without refreshing.
	restarted, err := New("client", "en", store, request)
	if err != nil {
		t.Fatal(err)
	}
	if access, err := restarted.tokenFor(context.Background()); err != nil || access != "fresh" {
		t.Fatalf("saved credentials did not survive restart: %v", err)
	}
}

func TestClientNewAuthorizationSupersedesPendingRefreshOnlyAfterSave(t *testing.T) {
	store := &memoryTokens{token: Token{AccessToken: "old"}, fail: true}
	c, err := New("client", "en", store, HTTPRequest(http.DefaultClient))
	if err != nil {
		t.Fatal(err)
	}
	pending := Token{AccessToken: "rotated", RefreshToken: "rotated-refresh"}
	c.pendingToken = &pending
	replacement := Token{AccessToken: "new-account", ExpiresIn: 3600, SavedAt: time.Now().Unix()}
	identity := Identity{Login: "new-channel", UserID: "456"}
	if err := c.AcceptValidatedToken(replacement, identity); err == nil || c.pendingToken == nil {
		t.Fatal("failed authorization save discarded pending rotation")
	}
	store.fail = false
	if err := c.AcceptValidatedToken(replacement, identity); err != nil {
		t.Fatal(err)
	}
	access, err := c.tokenFor(context.Background())
	if err != nil || access != replacement.AccessToken || store.token.AccessToken != access || c.Connection().Login != identity.Login {
		t.Fatal("pending rotation overwrote new authorization")
	}
}

func (f clientTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClientKeepsCredentialsWhenCommitFails(t *testing.T) {
	store := &memoryTokens{token: Token{AccessToken: "original"}}
	c, err := New("client", "en", store, HTTPRequest(http.DefaultClient))
	if err != nil {
		t.Fatal(err)
	}
	store.fail = true
	if err := c.AcceptValidatedToken(Token{AccessToken: "replacement"}, Identity{Login: "other"}); err == nil {
		t.Fatal("failed save accepted")
	}
	if c.token.AccessToken != "original" || c.Connection().Login != "" {
		t.Fatal("failed save changed live credentials")
	}
}

func TestClientRefreshUsesInjectedTransportAndPersistsRotation(t *testing.T) {
	store := &memoryTokens{token: Token{AccessToken: "expired", RefreshToken: "one-use"}}
	calls := 0
	transport := clientTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != AuthURL+"/token" {
			t.Fatalf("unexpected endpoint %s", r.URL)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatal(err)
		}
		if r.Form.Get("refresh_token") != "one-use" {
			t.Fatal("wrong refresh token")
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"access_token":"fresh","refresh_token":"rotated","expires_in":3600}`))}, nil
	})
	c, err := New("client", "en", store, HTTPRequest(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.tokenFor(context.Background())
	if err != nil || token != "fresh" || store.token.RefreshToken != "rotated" {
		t.Fatalf("refresh lost rotation: %s %v %+v", token, err, store.token)
	}
	if _, err := c.tokenFor(context.Background()); err != nil || calls != 1 {
		t.Fatal("fresh token refreshed again")
	}
	if time.Since(time.Unix(store.token.SavedAt, 0)) > time.Minute {
		t.Fatal("refresh timestamp not saved")
	}
}
