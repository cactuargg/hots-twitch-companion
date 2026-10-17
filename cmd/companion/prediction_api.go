package main

import "context"

// PredictionAPI is the Twitch-facing boundary of the Prediction lifecycle.
// Injecting a fake lets tests verify write-ahead persistence and one-shot
// creation without making real requests or distributing viewer points.
// The production implementation is *Twitch.
type PredictionAPI interface {
	PreflightCreate(context.Context) (TwitchIdentity, error)
	CreatePrepared(context.Context, string, int, string, string, string) (*Prediction, error)
	Create(context.Context, int, string, string, string) (*Prediction, error)
	Prediction(context.Context, string, string) (*apiResponse, error)
	End(context.Context, *Prediction, string) error
}

var _ PredictionAPI = (*Twitch)(nil)
