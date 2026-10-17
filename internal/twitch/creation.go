package twitch

import (
 "errors"
 "net/url"
 "hots-twitch-companion/internal/prediction"
)

// IdentityIssue preserves the ordered validation rules used before accepting
// OAuth credentials. Localization is intentionally left to the caller.
func IdentityIssue(me Identity, clientID, requiredScope string) string {
 if me.ClientID != clientID {return "wrong_client"}
 valid:=false
 for _,s:=range me.Scopes {if s==requiredScope {valid=true}}
 if !valid {return "no_scope"}
 if me.UserID=="" {return "missing_broadcaster"}
 return ""
}

// PredictionRoute always URL-escapes the broadcaster and optional Prediction ID.
func PredictionRoute(broadcaster,id string) string {
 route:="/predictions?broadcaster_id="+url.QueryEscape(broadcaster)+"&first=5"
 if id!="" {route="/predictions?broadcaster_id="+url.QueryEscape(broadcaster)+"&id="+url.QueryEscape(id)}
 return route
}

func HasOpenPrediction(response *APIResponse) bool {
 if response==nil {return false}
 for _,p:=range response.Data {if p.Status=="ACTIVE"||p.Status=="LOCKED"{return true}}
 return false
}

var ErrInvalidOutcomes = errors.New("invalid Twitch Prediction outcome identifiers")
var ErrMalformedCreateResponse = errors.New("invalid Twitch Prediction creation response")

// CreationRequest validates the same limited parameters as the old client.
// The caller must have persisted the uncertain POST marker before sending it.
func CreationRequest(broadcasterID string, windowMinutes int, title,winTitle,lossTitle string)(map[string]any,error) {
 if broadcasterID==""||winTitle==lossTitle {return nil,ErrInvalidOutcomes}
 return map[string]any{
  "broadcaster_id":broadcasterID,
  "title":title,
  "outcomes":[]map[string]string{{"title":winTitle},{"title":lossTitle}},
  "prediction_window":windowMinutes*60,
 },nil
}

// ParseCreatedPrediction converts a *confirmed* ACTIVE Helix response to the
// durable domain model. It rejects any ambiguous or incomplete outcome IDs.
func ParseCreatedPrediction(d *APIResponse,broadcasterID,winTitle,lossTitle string,startedAt int64)(*prediction.Prediction,error){
 if d==nil||len(d.Data)!=1||len(d.Data[0].Outcomes)!=2||d.Data[0].Status!="ACTIVE"{
  return nil,ErrMalformedCreateResponse
 }
 p:=&prediction.Prediction{ID:d.Data[0].ID,BroadcasterID:broadcasterID,StartedAt:startedAt}
 for _,outcome:=range d.Data[0].Outcomes {
  if outcome.Title==winTitle {p.WinID=outcome.ID}
  if outcome.Title==lossTitle {p.LossID=outcome.ID}
 }
 if p.ID==""||p.WinID==""||p.LossID==""||p.WinID==p.LossID {return nil,ErrInvalidOutcomes}
 return p,nil
}
