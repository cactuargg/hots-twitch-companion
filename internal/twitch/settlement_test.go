package twitch

import (
 "context"
 "errors"
 "testing"
 "hots-twitch-companion/internal/prediction"
)

func TestFinishPredictionVerifiesExactRemoteOutcome(t *testing.T) {
 p:=&prediction.Prediction{ID:"wager",BroadcasterID:"channel",WinID:"win",LossID:"loss"}
 for _,tc:=range []struct{outcome,status,winner string}{
  {"win","RESOLVED","win"},{"loss","RESOLVED","loss"},{"cancel","CANCELED",""},
 }{
  t.Run(tc.outcome,func(t *testing.T){
   reads,patches:=0,0
   read:=func(context.Context,string,string)(*APIResponse,error){reads++;return &APIResponse{Data:[]APIPrediction{{ID:"wager",Status:"ACTIVE"}}},nil}
   patch:=func(_ context.Context,body map[string]any)(*APIResponse,error){
    patches++
    if body["status"]!=tc.status||body["broadcaster_id"]!="channel"||body["id"]!="wager"{t.Fatalf("bad settlement body: %+v",body)}
    if tc.winner!=""&&body["winning_outcome_id"]!=tc.winner{t.Fatalf("wrong winner: %+v",body)}
    if tc.winner==""&&body["winning_outcome_id"]!=nil{t.Fatal("cancel included winner")}
    return &APIResponse{Data:[]APIPrediction{{ID:"wager",Status:tc.status,WinningOutcomeID:tc.winner}}},nil
   }
   if err:=FinishPrediction(context.Background(),p,tc.outcome,read,patch);err!=nil{t.Fatal(err)}
   if reads!=1||patches!=1{t.Fatalf("unexpected read/patch %d/%d",reads,patches)}
  })
 }
}

func TestFinishPredictionLostPatchResponseDoesNotRepay(t *testing.T) {
 p:=&prediction.Prediction{ID:"one",BroadcasterID:"owner",WinID:"yes",LossID:"no"}
 reads,patches:=0,0
 read:=func(context.Context,string,string)(*APIResponse,error){
  reads++
  status:="ACTIVE";winner:=""
  if reads>1{status="RESOLVED";winner="yes"}
  return &APIResponse{Data:[]APIPrediction{{ID:"one",Status:status,WinningOutcomeID:winner}}},nil
 }
 patch:=func(context.Context,map[string]any)(*APIResponse,error){patches++;return nil,errors.New("lost response")}
 if err:=FinishPrediction(context.Background(),p,"win",read,patch);err!=nil{t.Fatal(err)}
 if patches!=1||reads!=2{t.Fatalf("lost-response recovery: %d GET and %d PATCH",reads,patches)}
}

func TestFinishPredictionRefusesRemoteIDMismatchAndConflictingResolution(t *testing.T) {
 p:=&prediction.Prediction{ID:"one",BroadcasterID:"owner",WinID:"yes",LossID:"no"}
 patches:=0
 patch:=func(context.Context,map[string]any)(*APIResponse,error){patches++;return nil,nil}
 readWrong:=func(context.Context,string,string)(*APIResponse,error){return &APIResponse{Data:[]APIPrediction{{ID:"other",Status:"ACTIVE"}}},nil}
 if err:=FinishPrediction(context.Background(),p,"win",readWrong,patch);err==nil||patches!=0 {t.Fatalf("mismatched remote id: %v, patches=%d",err,patches)}
 readCanceled:=func(context.Context,string,string)(*APIResponse,error){return &APIResponse{Data:[]APIPrediction{{ID:"one",Status:"CANCELED"}}},nil}
 err:=FinishPrediction(context.Background(),p,"win",readCanceled,patch)
 var finished *RemoteFinishedError
 if !errors.As(err,&finished)||patches!=0||RemoteOutcome(finished.Prediction,p)!="cancel"{t.Fatalf("conflicting finalization %v, patches=%d",err,patches)}
}
