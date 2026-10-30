package twitch

import (
 "errors"
 "strings"
 "testing"
)

func TestIdentityValidationRules(t *testing.T){
 base:=Identity{ClientID:"app",UserID:"channel",Scopes:[]string{"channel:manage:predictions"}}
 if issue:=IdentityIssue(base,"app","channel:manage:predictions");issue!="" {t.Fatal(issue)}
 wrong:=base;wrong.ClientID="other"
 if issue:=IdentityIssue(wrong,"app","channel:manage:predictions");issue!="wrong_client"{t.Fatal(issue)}
 noScope:=base;noScope.Scopes=nil
 if issue:=IdentityIssue(noScope,"app","channel:manage:predictions");issue!="no_scope"{t.Fatal(issue)}
 missing:=base;missing.UserID=""
 if issue:=IdentityIssue(missing,"app","channel:manage:predictions");issue!="missing_broadcaster"{t.Fatal(issue)}
}

func TestCreateResponseRequiresDistinctVerifiedOutcomes(t *testing.T){
 request,err:=CreationRequest("channel",2,"Match","Win","Lose")
 if err!=nil || request["prediction_window"]!=120 {t.Fatalf("request: %v %v",request,err)}
 if _,err:=CreationRequest("",2,"Match","Win","Lose");!errors.Is(err,ErrInvalidOutcomes){t.Fatal(err)}
 if _,err:=CreationRequest("channel",2,"Match","Same","Same");!errors.Is(err,ErrInvalidOutcomes){t.Fatal(err)}
 response:=&APIResponse{Data:[]APIPrediction{{ID:"prediction-id",Status:"ACTIVE"}}}
 response.Data[0].Outcomes=make([]struct{ID string `json:"id"`;Title string `json:"title"`;ChannelPoints *int64 `json:"channel_points"`},2)
 response.Data[0].Outcomes[0].ID="win-id";response.Data[0].Outcomes[0].Title="Win"
 response.Data[0].Outcomes[1].ID="loss-id";response.Data[0].Outcomes[1].Title="Lose"
 got,err:=ParseCreatedPrediction(response,"channel","Win","Lose",12345)
 if err!=nil || got.ID!="prediction-id" || got.WinID!="win-id" || got.LossID!="loss-id" || got.StartedAt!=12345 {t.Fatalf("confirmation: %+v %v",got,err)}
 wrong:=*response;wrong.Data=append([]APIPrediction(nil),response.Data...)
 wrong.Data[0].Status="LOCKED"
 if _,err:=ParseCreatedPrediction(&wrong,"channel","Win","Lose",0);!errors.Is(err,ErrMalformedCreateResponse){t.Fatal(err)}
 wrong.Data[0].Status="ACTIVE";wrong.Data[0].Outcomes[1].ID="win-id"
 if _,err:=ParseCreatedPrediction(&wrong,"channel","Win","Lose",0);!errors.Is(err,ErrInvalidOutcomes){t.Fatal(err)}
}
func TestPredictionRouteEscapesParametersAndOpenCheck(t *testing.T){
 route:=PredictionRoute("a&b","id ?")
 if !strings.Contains(route,"a%26b")||!strings.Contains(route,"id=id+%3F")||strings.Contains(route,"first=5"){t.Fatal(route)}
 if !HasOpenPrediction(&APIResponse{Data:[]APIPrediction{{Status:"ACTIVE"}}}) {t.Fatal("missed open Prediction")}
 if HasOpenPrediction(&APIResponse{Data:[]APIPrediction{{Status:"RESOLVED"}}}) {t.Fatal("resolved Prediction treated as active")}
}
