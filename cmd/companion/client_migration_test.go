package main

import (
	"hots-twitch-companion/internal/storage/credentials"
	"hots-twitch-companion/internal/twitch"
	"sync"
)

var testTokenPaths sync.Map

func testToken(t *twitch.Client) Token {
	p, _ := testTokenPaths.Load(t)
	var token Token
	_ = credentials.LoadJSON(p.(string), &token)
	return token
}
func seedToken(t *twitch.Client, token Token) {
	c := t.Connection()
	if err := t.AcceptValidatedToken(token, TwitchIdentity{UserID: c.BroadcasterID, Login: c.Login}); err != nil {
		panic(err)
	}
}
func seedIdentity(t *twitch.Client, login, id string) {
	if err := t.AcceptValidatedToken(testToken(t), TwitchIdentity{UserID: id, Login: login}); err != nil {
		panic(err)
	}
}

func newTestTwitch(p paths, id, lang string) (*Twitch, error) {
	t, err := newTwitch(p, id, lang)
	if err == nil {
		testTokenPaths.Store(t, p.secret)
	}
	return t, err
}
func registerTestTwitch(e *Engine) { testTokenPaths.Store(e.twitch, e.p.secret) }
