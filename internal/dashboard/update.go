package dashboard

import (
	"context"
	"hots-twitch-companion/internal/update"
)

type ReleaseInfo = update.Release

func FetchLatestRelease(ctx context.Context, endpoint string) (ReleaseInfo, error) {
	client := update.NewClient()
	client.Endpoint = endpoint
	return client.Fetch(ctx)
}
func IsNewerRelease(latest, current string) bool { return update.IsNewerRelease(latest, current) }
