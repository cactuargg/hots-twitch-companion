package main

import "context"

// PredictionAPI is the Twitch-facing boundary of the Prediction lifecycle.
// Injecting a fake lets tests verify write-ahead persistence and one-shot
// creation without making real requests or distributing viewer points.
// The production implementation is *Twitch.
type PredictionAPI interface {
	preflightCreate(context.Context) (TwitchIdentity, error)
	createPrepared(context.Context, string, int, string, string, string) (*Prediction, error)
	create(context.Context, int, string, string, string) (*Prediction, error)
	prediction(context.Context, string, string) (*apiResponse, error)
	end(context.Context, *Prediction, string) error
}

var _ PredictionAPI = (*Twitch)(nil)
