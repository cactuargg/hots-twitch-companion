'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const dir=path.join(__dirname,'..','web');
const html=fs.readFileSync(path.join(dir,'index.html'),'utf8');
const names=['core.js','i18n.js','replays.js','diagnostics.js','onboarding.js','app.js'];
const js=names.map(name=>fs.readFileSync(path.join(dir,'assets',name),'utf8')).join('\n');
const ids=new Set([...html.matchAll(/\bid="([^"]+)"/g)].map(match=>match[1]));

test('all static UI ID dependencies remain after deleting old dashboard',()=>{
 const selectors=[...js.matchAll(/\$\(['"]([\w-]+)['"]\)/g)].map(match=>match[1]);
 for(const id of selectors)assert.ok(ids.has(id),'Missing live UI element: '+id);
 for(const old of ['recent-list','log-preview','dashboard-grid','watch-tag','dash-channel','footer-mode'])
   assert.ok(!ids.has(old),'Unused game dashboard element: '+old);
 assert.equal((html.match(/role="tab"\s/g)||[]).length,3);
 assert.ok(html.includes('id="diagnostics-details"'));
 assert.ok(html.includes('id="overview-alert"'));
 assert.ok(html.includes('id="overview-last-map"'));
 assert.ok(!html.includes('topbar-art'));
});

function harness(){
 const elements=new Map();
 const get=id=>{
  if(!elements.has(id))elements.set(id,{
   id, textContent:'',hidden:false,dataset:{},attrs:{},value:'',
   classList:{classes:new Set(),toggle(key,on){ if(on)this.classes.add(key);else this.classes.delete(key); }},
   setAttribute(key,value){this.attrs[key]=value;}
  });
  return elements.get(id);
 };
 const btn=get('main-button');
 const t=(key,vars)=>key==='minimal.lastEvent'?'Latest: '+vars.message:key;
 const context={
  $:get,document:{querySelector:(sel)=>sel==='.overview-toggle'?btn:null},
  state:{config:{playerName:'Demo'},status:null,replays:[]},
  t,showLogs:()=>{},renderWizard:()=>{}
 };
 vm.createContext(context);
 const source=fs.readFileSync(path.join(dir,'assets','app.js'),'utf8');
 const from=source.indexOf('function monitoringView(');
 const end=source.indexOf('async function loadStatus()',from);
 assert.ok(from>=0&&end>from);
 vm.runInContext(source.slice(from,end),context);
 return {context,get,btn};
}
test('simulation shows one action, latest match and no live warning',()=>{
 const {context,get,btn}=harness();
 context.state.replays=[{map:'Dragon Shire',result:'win'}];
 context.render({running:true,active:false,realMode:false,connected:true,
   twitchPhase:'connected',channel:'demo',predictionPhase:'none',monitorPhase:'watching',
   logs:[{time:'15:00:00',message:'Watching for lobby'}]});
 assert.equal(get('overview-state-title').textContent,'overview.state.watching');
 assert.equal(get('overview-twitch').textContent,'@demo');
 assert.equal(get('overview-last-map').textContent,'Dragon Shire');
 assert.equal(get('overview-last-outcome').textContent,'badge.win');
 assert.equal(get('overview-alert').hidden,true);
 assert.equal(btn.dataset.action,'toggle');
 assert.equal(get('start-label').textContent,'start.stop');
});
test('real viewer-point mode remains visibly warned; missing player opens setup',()=>{
 const {context,get,btn}=harness();
 context.render({running:true,active:true,realMode:true,connected:false,
   twitchPhase:'unverified',predictionPhase:'window_estimate',monitorPhase:'tracking',
   predictionWindowSecondsLeft:45,logs:[]});
 assert.equal(get('overview-alert').hidden,false);
 assert.equal(get('overview-alert').textContent,'minimal.twitchWarning');
 assert.equal(get('automation-mode').textContent,'minimal.realMode');
 context.state.config={playerName:'',playerToonHandle:''};
 context.render({running:false,active:false,realMode:false,connected:false,
   twitchPhase:'disconnected',monitorPhase:'off',predictionPhase:'none',logs:[]});
 assert.equal(get('overview-state-title').textContent,'minimal.setup.title');
 assert.equal(btn.dataset.action,'wizard');
 assert.equal(get('start-label').textContent,'minimal.configure');
});
test('removed game-themed layout does not leak into base CSS',()=>{
 const css=fs.readFileSync(path.join(dir,'assets','style.css'),'utf8');
 assert.ok(!css.includes('clip-path'));
 assert.ok(!css.includes('url(\'./hero.webp\')'));
 assert.ok(!css.includes('grid-template-rows:132px'));
 assert.match(css,/--accent:\s*#9a9bf8/);
});
