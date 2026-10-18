'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const app=fs.readFileSync('internal/dashboard/web/assets/app.js','utf8');
function setup(){
 const elements=new Map();const $=id=>{if(!elements.has(id))elements.set(id,{checked:true,disabled:false,hidden:false,textContent:'',value:0});return elements.get(id)};
 const state={config:{playerName:'Saved',disableUpdateChecks:false},hasUnsaved:true,status:{updates:{enabled:true,phase:'available',version:'9.0.0',downloadable:true,url:'https://github.com/cactuargg/hots-twitch-companion/releases/tag/v9.0.0'}}};
 const ctx=vm.createContext({$,state,lang:'en',t:(k)=>k,toast(){},loadStatus:async()=>{},api:async()=>{throw Error('not configured')}});
 vm.runInContext(app.slice(app.indexOf('function renderUpdates('),app.indexOf('function renderSettingsDraft(')),ctx);
 return {ctx,$,state};
}
test('turning off update checks saves only the preference and preserves settings drafts',async()=>{
 const {ctx,$,state}=setup();let sent;
 ctx.api=async(route,method,data)=>{sent={route,method,data};return {config:{...state.config,disableUpdateChecks:true},updates:{enabled:false,phase:'disabled'}}};
 $('updates-enabled').checked=false;await ctx.setUpdateChecks();
 assert.equal(sent.route,'updates/preferences');assert.equal(JSON.stringify(sent.data),''+JSON.stringify({enabled:false}));
 assert.equal(state.hasUnsaved,true);assert.equal(state.config.playerName,'Saved');assert.equal($('update-notice').hidden,true);assert.equal($('updates-check').disabled,true);
});
test('failed preference save restores the switch and preserves the draft',async()=>{
 const {ctx,$,state}=setup();ctx.api=async()=>{throw Error('disk full')};$('updates-enabled').checked=false;
 await ctx.setUpdateChecks();assert.equal($('updates-enabled').checked,true);assert.equal(state.hasUnsaved,true);assert.equal(state.saving,false);
});
test('installation is blocked while monitoring, a prediction, OAuth or unsaved changes are present',()=>{
 const {ctx,$,state}=setup();state.hasUnsaved=false;state.status.updates.phase='ready';
 for(const key of ['running','active','creationUnconfirmed','authPending']){
  state.status[key]=true;ctx.renderUpdates(state.status);assert.equal($('updates-install').disabled,true,key);state.status[key]=false;
 }
 state.hasUnsaved=true;ctx.renderUpdates(state.status);assert.equal($('updates-install').disabled,true);
 state.hasUnsaved=false;ctx.renderUpdates(state.status);assert.equal($('updates-install').disabled,false);
});
test('failed update checks show retry time and details without hiding a known release',()=>{
 const {ctx,$,state}=setup();state.status.updates={...state.status.updates,phase:'error',checkedAt:Date.now(),nextCheckAt:Date.now()+60000,checkAfterAt:Date.now()+60000,error:'GitHub HTTP 403'};
 ctx.renderUpdates(state.status);assert.equal($('updates-status').textContent,'updates.phase.error');assert.equal($('updates-check').disabled,true);
 assert.equal($('updates-error').textContent,'GitHub HTTP 403');assert.equal($('update-notice').hidden,false);assert.equal($('updates-retry').textContent,'updates.retry');
});

test('updater startup is acknowledged only after the frontend renders successfully',async()=>{
 for(const failRender of [false,true])for(const pending of [false,true]){
  const calls=[];let boot;
  const element={value:'en',addEventListener(){}};
  const context={state:{logs:[]},document:{querySelectorAll:()=>[],querySelector:()=>element},$:()=>element,
   window:{addEventListener:(name,fn)=>{boot=fn}},initDesktopShell(){},handleTabKeyDown(){},wizNext(){},wizardSkip(){},setUpdateChecks(){},loadStatus(){},reloadReplays(){},
   renderPlayerPickers(){},renderPredictionPreviews(){},renderSettingsDraft(){},renderHistory(){},applyI18n(){},setInputs(){},showReplays(){calls.push('replays')},
   render(status){if(status){calls.push('render');if(failRender)throw Error('render failed')}},
   api:async(route)=>{calls.push(route);return {config:{wizardDone:true},status:{updateStartupPending:pending},replays:[]}},
   toast(){},t:x=>x,setInterval:()=>1};
  const ctx=vm.createContext(context);
  vm.runInContext(app.slice(app.indexOf("window.addEventListener('DOMContentLoaded'"),app.indexOf("window.addEventListener('pagehide'")),ctx);
  await boot();assert.equal(calls.includes('updates/startup-ready'),pending&&!failRender);
  if(pending&&!failRender)assert.ok(calls.indexOf('updates/startup-ready')>calls.indexOf('replays'));
 }
});
