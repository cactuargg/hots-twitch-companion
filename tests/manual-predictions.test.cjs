'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const root=path.join(__dirname,'..');
const app=fs.readFileSync(path.join(root,'cmd/companion/web/assets/app.js'),'utf8');
const html=fs.readFileSync(path.join(root,'cmd/companion/web/index.html'),'utf8');

function harness(status,allow=true){
 const calls=[],confirmations=[],toasts=[];
 let refreshes=0;
 const ctx={
   state:{status},
   t:key=>key,
   confirm:question=>{confirmations.push(question);return allow},
   api:async(route,method,payload)=>{calls.push({route,method,payload});return {ok:true}},
   toast:msg=>toasts.push(msg),
   loadStatus:async()=>{refreshes++}
 };
 vm.createContext(ctx);
 const a=app.indexOf('async function manualPrediction('),b=app.indexOf('async function action(',a);
 assert.ok(a>=0&&b>a);
 vm.runInContext(app.slice(a,b),ctx);
 return {ctx,calls,confirmations,toasts,get refreshes(){return refreshes}};
}
test('Manual buttons are only shown when a Prediction is actually active',()=>{
 for(const name of ['manual-win','manual-loss','manual-cancel','manual-sync']){
  assert.ok(html.includes('data-action="'+name+'"'),'Missing action '+name);
  assert.ok(app.includes("case '"+name+"':"));
 }
 assert.ok(html.includes('id="prediction-manual" hidden'));
 assert.ok(html.includes('id="prediction-sync"'));
 assert.ok(app.includes('manual.hidden=!st.active'));
 assert.ok(app.includes("$('prediction-sync').hidden=!st.active||!!st.activeDryRun"));
});
test('Live Win/Loss/Cancel require positive confirmation before network action',async()=>{
 for(const action of ['win','loss','cancel']){
  const status={active:true,activeDryRun:false,activePredictionId:'live-123',activePredictionStartedAt:1234};
  const denied=harness(status,false);
  assert.equal(await denied.ctx.manualPrediction(action),false);
  assert.equal(denied.confirmations.length,1);
  assert.equal(denied.calls.length,0);
  const approved=harness(status,true);
  assert.equal(await approved.ctx.manualPrediction(action),true);
  assert.equal(approved.confirmations.length,1);
  assert.equal(approved.calls.length,1);
  assert.equal(approved.calls[0].route,'prediction/manual');
  assert.equal(approved.calls[0].method,'POST');
  assert.equal(approved.calls[0].payload.outcome,action);
  assert.equal(approved.calls[0].payload.id,'live-123');
  assert.equal(approved.calls[0].payload.startedAt,1234);
  assert.equal(approved.refreshes,1);
  assert.deepEqual(approved.toasts,['manual.success.updated']);
 }
});
test('Simulation never asks about channel points, but still sends exact active identity',async()=>{
 const h=harness({active:true,activeDryRun:true,activePredictionId:'SIMULATION',activePredictionStartedAt:25});
 assert.equal(await h.ctx.manualPrediction('cancel'),true);
 assert.equal(h.confirmations.length,0);
 assert.equal(h.calls[0].payload.outcome,'cancel');
});
test('Stale buttons and already settled predictions cannot issue requests',async()=>{
 for(const status of [{active:false},{active:true,activePredictionId:'x'},{}]){
  const h=harness(status);
  await assert.rejects(()=>h.ctx.manualPrediction('win'),/manual.stale/);
  assert.equal(h.calls.length,0);
 }
});
test('Sync checks Twitch status without irreversible-outcome confirmation',async()=>{
 const h=harness({active:true,activeDryRun:false,activePredictionId:'live-222',activePredictionStartedAt:2222});
 assert.equal(await h.ctx.manualPrediction('sync'),true);
 assert.equal(h.confirmations.length,0);
 assert.equal(h.calls[0].payload.outcome,'sync');
});

test('Unconfirmed outcome cannot be switched before Twitch is reconciled',async()=>{
 const status={active:true,activeDryRun:false,activePredictionId:'live-pending',activePredictionStartedAt:2222,pendingOutcome:'win'};
 for(const action of ['loss','cancel']){
  const h=harness(status);
  await assert.rejects(()=>h.ctx.manualPrediction(action),/manual.pending/);
  assert.equal(h.calls.length,0);
  assert.equal(h.confirmations.length,0);
 }
 for(const action of ['win','sync']){
  const h=harness(status);
  await h.ctx.manualPrediction(action);
  assert.equal(h.calls[0].payload.outcome,action);
 }
});
