'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),vm=require('node:vm'),{execFileSync}=require('node:child_process');
const source=fs.readFileSync('internal/dashboard/web/assets/app.js','utf8');
const region=(start,end)=>source.slice(source.indexOf(start),source.indexOf(end,source.indexOf(start)+start.length));

test('real Go status feeds the actual JS prediction view with compatible phase keys',()=>{
 const status=JSON.parse(execFileSync('go',['run','./internal/dashboard/testdata/statusfixture'],{encoding:'utf8',timeout:30000}));
 assert.equal(status.monitorPhase,'off');
 assert.equal(status.predictionPhase,'window_estimate');
 assert.ok(status.predictionWindowSecondsLeft>85&&status.predictionWindowSecondsLeft<=90);
 const ctx=vm.createContext({t:key=>key});
 vm.runInContext(region('function monitoringView(', 'function renderSettingsDraft('),ctx);
 assert.equal(ctx.monitoringView(status).prediction,'window_estimate');
 assert.equal(ctx.monitoringView(status).detail,'overview.detail.window');
});

function savingContext(){
 let finish,title='Original';
 const pending=new Promise(resolve=>finish=resolve);
 const ctx=vm.createContext({state:{hasUnsaved:true,status:{connected:true}},getInputs:()=>({predictionTitle:title}),api:()=>pending,applyI18n(){},renderSettingsDraft(){},toast(){},t:x=>x,loadStatus:async()=>{},lang:'en'});
 vm.runInContext(region('async function save(', 'async function manualPrediction('),ctx);
 return {ctx,edit:value=>{title=value;ctx.state.hasUnsaved=true},finish:()=>finish({config:{predictionTitle:'Original',lang:'en'}})};
}
test('edits made during save remain dirty and cannot start with the old settings',async()=>{
 const {ctx,edit,finish}=savingContext();
 const pending=ctx.save();edit('New draft');finish();await pending;
 assert.equal(ctx.state.config.predictionTitle,'Original');
 assert.equal(ctx.state.hasUnsaved,true);
 assert.equal(await ctx.startAutomation(),false);
 assert.equal(ctx.state.saving,false);
});
test('unchanged save clears dirty state; overlapping config writes are refused',async()=>{
 const {ctx,finish}=savingContext();
 const pending=ctx.save();
 await assert.rejects(ctx.save(),/settings.wait/);
 finish();await pending;
 assert.equal(ctx.state.hasUnsaved,false);
});
test('status polling exposes lost connection and automatically clears it after recovery',async()=>{
 const states=[];let fail=true;
 const ctx=vm.createContext({state:{status:{running:true}},api:async()=>{if(fail)throw Error('offline');return {running:false}},render:()=>states.push(ctx.state.connectionLost)});
 vm.runInContext(region('async function loadStatus(', 'async function reloadReplays('),ctx);
 await ctx.loadStatus();assert.equal(ctx.state.connectionLost,true);
 fail=false;await ctx.loadStatus();assert.equal(ctx.state.connectionLost,false);
 assert.deepEqual(states,[true,false]);
});
test('offline actions cannot use a stale active prediction',async()=>{
 let calls=0;
 const ctx=vm.createContext({state:{connectionLost:true},toast(){},t:x=>x,api:()=>calls++});
 vm.runInContext(region('async function action(name)', 'async function checkReplayRoot('),ctx);
 await ctx.action('manual-win');assert.equal(calls,0);
});
test('language change cannot overwrite a settings save in flight',async()=>{
 let calls=0;const field={value:'ru'};
 const ctx=vm.createContext({state:{saving:true},lang:'en',$:()=>field,toast(){},t:x=>x,api:()=>calls++});
 const translations=fs.readFileSync('internal/dashboard/web/assets/i18n.js','utf8');
 vm.runInContext(translations.slice(translations.indexOf('async function setLang(')),ctx);
 await ctx.setLang('ru');assert.equal(calls,0);assert.equal(field.value,'en');
});
test('failed save leaves the draft dirty and releases the save guard',async()=>{
 const {ctx}=savingContext();ctx.api=async()=>{throw Error('disk full')};
 await assert.rejects(ctx.save(),/disk full/);
 assert.equal(ctx.state.hasUnsaved,true);assert.equal(ctx.state.saving,false);
});
test('slow status request does not queue overlapping polls',async()=>{
 let calls=0,finish;
 const ctx=vm.createContext({state:{},render(){},api:()=>{calls++;return new Promise(resolve=>finish=resolve)}});
 vm.runInContext(region('async function loadStatus(', 'async function reloadReplays('),ctx);
 const first=ctx.loadStatus();await ctx.loadStatus();assert.equal(calls,1);
 finish({});await first;assert.equal(ctx.state.statusPending,false);
});
