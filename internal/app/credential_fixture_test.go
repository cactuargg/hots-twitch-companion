package app

import (
	"hots-twitch-companion/internal/config"
	"hots-twitch-companion/internal/storage/credentials"
	twitchpkg "hots-twitch-companion/internal/twitch"
	"sync"
)

// Credential fixtures inspect the persisted test store rather than client fields.
var testTokenPaths sync.Map

func testToken(t *twitchpkg.Client) twitchpkg.Token {
	p, _ := testTokenPaths.Load(t)
	var token twitchpkg.Token
	_ = credentials.LoadJSON(p.(string), &token)
	return token
}
func seedToken(t *twitchpkg.Client, token twitchpkg.Token) {
	c := t.Connection()
	if err := t.AcceptValidatedToken(token, twitchpkg.Identity{UserID: c.BroadcasterID, Login: c.Login}); err != nil {
		panic(err)
	}
}
func seedIdentity(t *twitchpkg.Client, login, id string) {
	if err := t.AcceptValidatedToken(testToken(t), twitchpkg.Identity{UserID: id, Login: login}); err != nil {
		panic(err)
	}
}

func newTestTwitch(p config.Paths, id, lang string) (*twitchpkg.Client, error) {
	t, err := newTwitch(p, id, lang)
	if err == nil {
		testTokenPaths.Store(t, p.Secret)
	}
	return t, err
}
func registerTestTwitch(e *Engine) { testTokenPaths.Store(e.twitch, e.p.Secret) }
