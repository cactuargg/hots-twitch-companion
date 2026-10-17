package twitch

import (
	"context"
	"errors"
	"io"
	"net/http"
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
