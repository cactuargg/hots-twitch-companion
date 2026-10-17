package main

import (
 correlation "hots-twitch-companion/internal/hots/correlation"
	predstate "hots-twitch-companion/internal/prediction"
	"time"
)

// Evidence, not a global Match ID. Never rebind an existing Prediction.
type MatchBinding = predstate.MatchBinding

func bindPredictionMatch(lobby *LobbySnapshot,key,toon string)*MatchBinding {
 return correlation.BindPredictionMatch(lobby,key,toon)
}
func observeActiveLobby(active *Prediction,next *LobbySnapshot,key string)*Prediction {
 return correlation.ObserveActiveLobby(active,next,key)
}
func ambiguousMatch(active *Prediction)*Prediction {
 return correlation.AmbiguousMatch(active)
}

type correlatedReplay = correlation.CorrelatedReplay

func correlationReplayFiles(cfg Config,after time.Time)([]ReplayFile,error){
 return correlation.CorrelationReplayFiles(cfg.ReplayRoot,after)
}
func readStableReplay(expected ReplayFile)([]byte,error){
 return correlation.ReadStableReplay(expected)
}
func contentSHA(raw []byte) string{return correlation.ContentSHA(raw)}

// Deliberately narrow automatic support: 5v5 lobbies. Matchmaking games
// must field ten human participants; custom lobbies may substitute AI
// participants, while every human slot is still verified by ToonHandle and
// observers stay excluded by explicit role, never by slot position.
func participantToons(lobby *BattleLobby)(map[string]bool,error) {return correlation.ParticipantToons(lobby)}
func isHeroHandle(toon string)bool{return correlation.IsHeroHandle(toon)}
func correlatedOutcome(replay Replay,lobby *BattleLobby,toon string)(Player,error) {
 return correlation.CorrelatedOutcome(replay,lobby,toon)
}

func correlatePrediction(cfg Config,p *Prediction,now time.Time)(*correlatedReplay,string,string,string){
 return correlation.CorrelatePrediction(cfg.ReplayRoot,p,now)
}
