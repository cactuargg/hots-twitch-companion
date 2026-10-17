package main

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

type fakePredictionAPI struct {
	preflightCalls, createCalls, endCalls int
	preflightError error
	onCreate func() error
	onEnd func(*Prediction,string) error
}
func (f *fakePredictionAPI) preflightCreate(context.Context)(TwitchIdentity,error) {
	f.preflightCalls++
	if f.preflightError!=nil {return TwitchIdentity{},f.preflightError}
	return TwitchIdentity{UserID:"channel"},nil
}
func (f *fakePredictionAPI) createPrepared(_ context.Context, broadcaster string, _ int, _,win,loss string)(*Prediction,error) {
	f.createCalls++
	if f.onCreate!=nil {if err:=f.onCreate();err!=nil{return nil,err}}
	return &Prediction{ID:"fake-prediction",BroadcasterID:broadcaster,WinID:"win-01",LossID:"loss-01",StartedAt:time.Now().UnixMilli()},nil
}
func (f *fakePredictionAPI) create(ctx context.Context, _ int, _,_,_ string)(*Prediction,error) {
	me,err:=f.preflightCreate(ctx)
	if err!=nil{return nil,err}
	return f.createPrepared(ctx,me.UserID,2,"","","")
}
func (f *fakePredictionAPI) prediction(context.Context,string,string)(*apiResponse,error){return &apiResponse{},nil}
func (f *fakePredictionAPI) end(_ context.Context,p *Prediction, outcome string)error {
	f.endCalls++
	if f.onEnd!=nil{return f.onEnd(p,outcome)}
	return nil
}

func TestInjectedPredictionAPIRetriesOnlyPreflightAndPersistsBeforePost(t *testing.T) {
	temp:=t.TempDir()
	t.Setenv("TEMP",temp)
	t.Setenv("TMP",temp)
	p:=paths{dir:temp,config:filepath.Join(temp,"config.json"),state:filepath.Join(temp,"state.json"),secret:filepath.Join(temp,"tokens.dpapi")}
	e,err:=newEngine(p,Config{RealMode:true,PlayerName:"Tester",Lang:"en"},func(string){})
	if err!=nil {t.Fatal(err)}
	fake:=&fakePredictionAPI{preflightError:errors.New("temporary validation outage")}
	fake.onCreate=func()error {
		s,err:=loadState(p)
		if err!=nil {return err}
		if s.Active!=nil || s.UnconfirmedCreation==nil || s.LastCreatedLobbySHA=="" || s.PendingLobby!=nil {
			return errors.New("POST attempted before the lobby and uncertainty marker were committed")
		}
		return nil
	}
	e.predictions=fake
	now:=time.Now()
	lobbyFileTest(t,realLobbyTest(t),now.Add(-5*time.Second))
	lastError:=""
	lastLog:=time.Time{}
	started:=now.Add(-time.Minute)
	e.scanTick(context.Background(),started,&lastError,&lastLog)
	if fake.createCalls!=0 || e.state.PendingLobby==nil || e.state.UnconfirmedCreation!=nil {
		t.Fatalf("preflight error consumed lobby: calls=%d state=%+v",fake.createCalls,e.state)
	}
	fake.preflightError=nil
	e.scanTick(context.Background(),started,&lastError,&lastLog)
	if fake.createCalls!=1 || e.state.Active==nil || e.state.UnconfirmedCreation!=nil {
		t.Fatalf("confirmed creation was not durably committed: calls=%d state=%+v",fake.createCalls,e.state)
	}
	e.scanTick(context.Background(),started,&lastError,&lastLog)
	if fake.createCalls!=1 {t.Fatalf("duplicate create: %d",fake.createCalls)}
}

func TestInjectedPredictionAPISeesDurableSettlementIntent(t *testing.T) {
	e,p:=liveLifecycleTest(t)
	fake:=&fakePredictionAPI{}
	fake.onEnd=func(predict *Prediction,outcome string)error {
		s,err:=loadState(p)
		if err!=nil{return err}
		if s.Active==nil || s.Active.Settlement==nil || s.Active.Settlement.Outcome!=outcome ||
			s.Active.ID!=predict.ID {
			return errors.New("remote settlement started without durable intent")
		}
		return nil
	}
	e.predictions=fake
	s:=e.state
	if err:=e.doSettle(context.Background(),&s,"win","manual");err!=nil {t.Fatal(err)}
	if fake.endCalls!=1 || e.state.Active!=nil || s.Active!=nil {
		t.Fatalf("settlement failed: calls=%d state=%+v",fake.endCalls,e.state)
	}
}
