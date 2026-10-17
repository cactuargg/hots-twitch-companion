'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const root=path.join(__dirname,'..');
const html=fs.readFileSync(path.join(root,'cmd/companion/web/index.html'),'utf8');
const source=fs.readFileSync(path.join(root,'cmd/companion/web/assets/onboarding.js'),'utf8');
const start=source.indexOf('const wiz={step:0,max:2,busy:false};');
assert.ok(start>=0);

function setup(){
  const elements=new Map(), calls=[];
  const make=()=>({
    value:'',textContent:'',hidden:false,disabled:false,open:false,dataset:{},className:'',
    children:[],
    classList:{toggle(){}},
    focus(){this.focused=true;},
    append(child){this.children.push(child);},
    replaceChildren(){this.children=[]}
  });
  const $=id=>{if(!elements.has(id))elements.set(id,make());return elements.get(id)};
  const sections=[0,1,2].map(step=>Object.assign(make(),{dataset:{step:String(step)}}));
  const back=make();
  const doc={
    querySelectorAll:sel=>sel==='.wizard-body section'?sections:[],
    querySelector:sel=>sel==='[data-wiz="back"]'?back:null,
    createElement:()=>make()
  };
  $('wizard').hidden=true;
  $('playerName').value='Old player';
  $('clientId').value='';
  $('realMode').checked=false;
  const state={
    config:{playerName:'Old player',playerToonHandle:'',clientId:'',realMode:false,wizardDone:false},
    status:{running:false,realMode:false,connected:false,twitchPhase:'disconnected'},
    hasUnsaved:false
  };
  const ctx={
    $,document:doc,state,
    t:(key,vars)=>vars?key+':'+JSON.stringify(vars):key,
    confirm:()=>true,
    renderPlayerPickers(){},renderPredictionPreviews(){},render(){},
    toast(msg,err){ctx.lastToast={msg,err}},
    go(page){ctx.page=page},
    async save(over={}){
      calls.push({type:'save',payload:{...over}});
      const conf={...state.config,...over,playerName:$('playerName').value,clientId:$('clientId').value};
      state.config=conf;state.hasUnsaved=false;
      return {config:conf};
    },
    async api(route,method,data){
      calls.push({type:'api',route,data});
      if(route==='config'){state.config={...data};return {config:state.config};}
      if(route==='automation/start'){state.status.running=true;return {};}
      if(route==='auth')return {code:'AUTHCODE',verificationUri:'https://www.twitch.tv/activate'};
      return {};
    },
    async loadStatus(){},async reloadReplays(){},
    code:{authUrl:''},window:{open(){}},
    async copyTwitchCode(){},getInputs:()=>({...state.config})
  };
  vm.createContext(ctx);
  const app=fs.readFileSync(path.join(root,'cmd/companion/web/assets/app.js'),'utf8');
  vm.runInContext(app.slice(app.indexOf('async function startAutomation('),app.indexOf('async function manualPrediction(')),ctx);
  vm.runInContext(source.slice(start),ctx);
  return {ctx,$,calls,state,sections,back};
}

test('Wizard has three focused steps and a single explicit live launch CTA',()=>{
 const wizard=html.slice(html.indexOf('<div class="wizard-backdrop"'));
 assert.deepEqual([...wizard.matchAll(/<section data-step="(\d+)"/g)].map(m=>m[1]),['0','1','2']);
 assert.equal((wizard.match(/data-wiz="next"/g)||[]).length,1);
 assert.equal((wizard.match(/data-wiz="start"/g)||[]).length,0,'No duplicate Start button');
 assert.ok(wizard.includes('id="wz-twitch-details"'));
 assert.ok(wizard.includes('id="wz-safe-note"'));
 assert.ok(wizard.includes('id="wizard-player-picker"'));
 assert.ok(wizard.includes('id="wz-preview-title"'));
 const labels=[...wizard.matchAll(/\bid="([^"]+)"/g)].map(m=>m[1]);
 assert.equal(labels.length,new Set(labels).size,'Duplicate wizard IDs');
});

test('Cancelling launch or missing Twitch never starts Predictions or finishes setup',async()=>{
 for(const connected of [false,true]){
  const {ctx,$,calls,state}=setup();ctx.openWizard();$('wz-player').value='Streamer';
  vm.runInContext('wiz.step=2',ctx);state.status.connected=connected;ctx.confirm=()=>false;
  assert.equal(await ctx.wizNext(),false);
  assert.equal(calls.some(c=>c.route==='automation/start'),false);
  assert.equal(state.config.wizardDone,false);
  assert.equal($('wizard').hidden,false);
 }
});

test('Player selection is required for next, but replay parsing is optional',async()=>{
 const {ctx,$,calls}=setup();
 ctx.openWizard();
 $('wz-player').value='';
 assert.equal($('wizard').hidden,false);
 assert.equal(await ctx.wizNext(),false);
 assert.equal($('wz-player').focused,true);
 assert.equal(calls.length,0);
 $('wz-player').value='HotSStreamer';
 assert.equal(await ctx.wizNext(),true);
 assert.equal($('playerName').value,'HotSStreamer');
 assert.equal(calls[0].type,'save');
 assert.equal(vm.runInContext('wiz.step',ctx),1);
});

test('Twitch connection is required before proceeding to live launch',async()=>{
 const {ctx,$,calls,state}=setup();
 ctx.openWizard();
 $('wz-player').value='HotSStreamer';
 await ctx.wizNext();
 assert.equal(await ctx.wizNext(),false);
 assert.equal(vm.runInContext('wiz.step',ctx),1);
 state.status.connected=true;
 assert.equal(await ctx.wizNext(),true);
 assert.equal(vm.runInContext('wiz.step',ctx),2);
 assert.equal(calls.filter(c=>c.route==='auth').length,0);
 assert.equal($('wiz-next').textContent,'onboarding.simple.start');
 assert.equal($('wiz-skip').textContent,'onboarding.simple.finishLater');
});

test('Twitch connect with no Client ID reveals the exact field, not an API request',async()=>{
 const {ctx,$,calls}=setup();
 ctx.openWizard();
 vm.runInContext('wiz.step=1',ctx);
 assert.equal(await ctx.wizardAction('auth'),false);
 assert.equal($('wz-twitch-details').open,true);
 assert.equal($('wz-client').focused,true);
 assert.equal(calls.length,0);
});

test('Start confirms real Predictions, starts automation and marks setup complete',async()=>{
 const {ctx,$,calls,state}=setup();
 ctx.openWizard();
 $('wz-player').value='HotSStreamer';
 vm.runInContext('wiz.step=2',ctx);
 state.status.connected=true;
 state.config.realMode=true;
 $('realMode').checked=true;
 assert.equal(await ctx.wizNext(),true);
 assert.equal(calls[0].type,'save');
 assert.notEqual(calls[0].payload.realMode,false);
 assert.equal(calls[1].route,'automation/start');
 assert.equal(calls[1].data.confirmedRealMode,true);
 assert.equal(calls[2].route,'config');
 assert.equal(calls[2].data.wizardDone,true);
 assert.equal(calls[2].data.realMode,true);
 assert.equal($('wizard').hidden,true);
 assert.equal(ctx.page,'dashboard');
});

test('A running monitor cannot be started again from wizard',async()=>{
 const {ctx,$,calls,state}=setup();
 ctx.openWizard();
 $('wz-player').value='HotSStreamer';
 vm.runInContext('wiz.step=2',ctx);
 state.status.running=true;
 state.status.realMode=true;
 const ok=await ctx.wizNext();
 assert.equal(ok,false);
 assert.equal(calls.length,0);
 assert.equal($('wizard').hidden,false);
});

test('Dismiss stores wizardDone without saving unrelated advanced Settings drafts',async()=>{
 const {ctx,$,calls,state}=setup();
 ctx.openWizard();
 $('clientId').value='UNSAVED-CLIENT-ID';
 state.hasUnsaved=true;
 assert.equal(await ctx.wizardSkip(),true);
 assert.equal(calls.length,1);
 assert.equal(calls[0].route,'config');
 assert.equal(calls[0].data.clientId,'');
 assert.equal(calls[0].data.wizardDone,true);
 assert.equal($('wizard').hidden,true);
});

test('Failure to start leaves wizard open and never marks setup completed',async()=>{
 const {ctx,$,calls,state}=setup();
 ctx.openWizard();
 $('wz-player').value='HotSStreamer';
 state.status.connected=true;
 vm.runInContext('wiz.step=2',ctx);
 ctx.api=async(route,method,data)=>{
   calls.push({type:'api',route,data});
   if(route==='automation/start')throw new Error('Start failed');
   return {config:data};
 };
 assert.equal(await ctx.wizNext(),false);
 assert.equal($('wizard').hidden,false);
 assert.equal(state.config.wizardDone,false);
 assert.equal(calls.some(c=>c.route==='config'),false);
});
