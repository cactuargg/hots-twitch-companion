package app

import (
	"context"
	"hots-twitch-companion/internal/prediction"
	twitchpkg "hots-twitch-companion/internal/twitch"
)

// PredictionAPI is the Twitch-facing boundary of the Prediction lifecycle.
// Injecting a fake lets tests verify write-ahead persistence and one-shot
// creation without making real requests or distributing viewer points.
// The production implementation is *Twitch.
type PredictionAPI interface {
	PreflightCreate(context.Context) (twitchpkg.Identity, error)
	CreatePrepared(context.Context, string, int, string, string, string) (*prediction.Prediction, error)
	Create(context.Context, int, string, string, string) (*prediction.Prediction, error)
	Prediction(context.Context, string, string) (*twitchpkg.APIResponse, error)
	End(context.Context, *prediction.Prediction, string) error
}

var _ PredictionAPI = (*twitchpkg.Client)(nil)
