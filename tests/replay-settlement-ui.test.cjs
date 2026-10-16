'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');
const dir=path.join(__dirname,'..','web');
const app=fs.readFileSync(path.join(dir,'assets','app.js'),'utf8');
const html=fs.readFileSync(path.join(dir,'index.html'),'utf8');
const i18n=fs.readFileSync(path.join(dir,'assets','i18n.js'),'utf8');

function actionContext(status,approved) {
 const records=[];
 const ctx={
  state:{busy:false,status},t:key=>key,
  confirm:prompt=>{records.push({type:'confirm',prompt});return approved},
  api:async (route,method,payload)=>{records.push({type:'api',route,method,payload});return {}},
  loadStatus:async()=>records.push({type:'refresh'}),
  toast:()=>{},setActionBusy:()=>{}
 };
 const start=app.indexOf('async function action(name)');
 const end=app.indexOf('async function checkReplayRoot()',start);
 assert.ok(start>=0&&end>start);
 vm.createContext(ctx);vm.runInContext(app.slice(start,end),ctx);
 return {ctx,records};
}
test('one named recheck action can actively settle a pending Prediction',async()=>{
 const h=actionContext({active:true,activeDryRun:true,activePredictionId:'SIMULATION',activePredictionStartedAt:1234},true);
 await h.ctx.action('recheck-result');
 assert.equal(h.records.filter(x=>x.type==='confirm').length,0);
 const api=h.records.find(x=>x.type==='api');
 assert.equal(api.route,'prediction/recheck');
 assert.equal(api.method,'POST');
 assert.equal(api.payload.id,'SIMULATION');
 assert.equal(api.payload.startedAt,1234);
 assert.equal(h.records.filter(x=>x.type==='refresh').length,1);
});
test('live result recheck requires explicit viewer-points confirmation',async()=>{
 const state={active:true,activeDryRun:false,activePredictionId:'PRED-1',activePredictionStartedAt:7878};
 const denied=actionContext(state,false);
 await denied.ctx.action('recheck-result');
 assert.equal(denied.records.filter(x=>x.type==='confirm').length,1);
 assert.equal(denied.records.filter(x=>x.type==='api').length,0);
 const allowed=actionContext(state,true);
 await allowed.ctx.action('recheck-result');
 assert.equal(allowed.records.filter(x=>x.type==='confirm').length,1);
 assert.equal(allowed.records.filter(x=>x.type==='api').length,1);
});
test('stale button cannot request another Prediction and read-only replay preview remains distinct',async()=>{
 const h=actionContext({active:false},true);
 await h.ctx.action('recheck-result');
 assert.equal(h.records.filter(x=>x.type==='api').length,0);
 assert.ok(html.includes('data-action="recheck-result"'));
 assert.ok(html.includes('id="prediction-scan-message"'));
 assert.ok(app.includes("case 'replay':{"));
 assert.ok(app.includes("api('replay/check','POST',{})"));
 assert.ok(i18n.includes("replays.check':'View latest replay result'"));
 assert.ok(i18n.includes("replays.check':'Показать результат реплея'"));
});
test('UI renders explicit replay diagnosis and alerts if monitoring stops scanning',()=>{
 assert.match(app,/st\.replayScan\|\|\{\}/);
 assert.match(app,/stale\?'stalled'/);
 assert.match(app,/prediction-scan-message/);
 for(const key of ['too_old','player_not_found','settle_error','stalled','paused','correlation_unavailable','correlation_ambiguous','correlation_incomplete','correlation_mismatch']){
  assert.ok(i18n.includes("'replay.scan."+key+"'"),'Missing diagnostic '+key);
 }
});
