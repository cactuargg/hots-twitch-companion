'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const root=path.join(__dirname,'..');
const html=fs.readFileSync(path.join(root,'web/index.html'),'utf8');
const source=fs.readFileSync(path.join(root,'web/assets/app.js'),'utf8');
const core=fs.readFileSync(path.join(root,'web/assets/core.js'),'utf8');
function slice(start,end){
 const a=source.indexOf(start),b=source.indexOf(end,a+start.length);
 assert.ok(a>=0&&b>a,'Missing source region: '+start);
 return source.slice(a,b);
}
function env(){
 const elements=new Map();
 const $=id=>{
   if(!elements.has(id)){
     const classes=new Set();
     elements.set(id,{
       id,value:'',checked:false,hidden:false,open:false,textContent:'',dataset:{},
       classList:{toggle:(key,on)=>on?classes.add(key):classes.delete(key),contains:key=>classes.has(key)},
       closest:()=>null,
       focus(){this.focused=true}
     });
   }
   return elements.get(id);
 };
 const row=$('row');
 $('realMode').closest=()=>row;
 const calls=[];
 const ctx={
   $,t:(key)=>key,state:{config:{realMode:false},hasUnsaved:false,busy:false},
   api:async(route,method,payload)=>{calls.push({route,method,payload});return {code:'ABCD',verificationUri:'https://www.twitch.tv/activate'};},
   go:page=>{ctx.page=page},toast:(message,isError)=>{ctx.lastToast={message,isError}},
   code:{authUrl:''},window:{}
 };
 vm.createContext(ctx);
 vm.runInContext(slice('function renderSettingsDraft(', 'function render(s){'),ctx);
 vm.runInContext(slice('async function action(name)', 'async function checkReplayRoot()'),ctx);
 return {ctx,$,calls,row};
}
test('Settings shows three simple sections, with technical fields disclosed only on demand',()=>{
 const sections=[...html.matchAll(/class="settings-card" aria-labelledby="([^"]+)"/g)].map(m=>m[1]);
 assert.deepEqual(sections,['settings-twitch-title','settings-player-title','settings-pred-title']);
 const start=html.indexOf('<details class="secondary-details settings-advanced" id="settings-advanced">');
 const finish=html.indexOf('</details>',start);
 assert.ok(start>0&&finish>start);
 const advanced=html.slice(start,finish);
 for(const id of ['clientId','toonHandle','replayRoot','predTitle','predWin','predLoss'])
   assert.ok(advanced.includes('id="'+id+'"'),'Missing advanced control: '+id);
 for(const id of ['playerName','predictionWindow'])
   assert.ok(html.indexOf('id="'+id+'"')<start,'Expected basic visible control: '+id);
 assert.ok(html.includes('id="settings-save-status"'));
 assert.ok(!html.includes('id="realMode"'));
 assert.equal((html.match(/data-action="save"/g)||[]).length,1);
 const ids=[...html.matchAll(/\bid="([^"]+)"/g)].map(m=>m[1]);
 assert.equal(ids.length,new Set(ids).size,'Duplicate IDs in settings view');
});
test('Savebar flags unsaved edits without a simulation/live switch',()=>{
 const {ctx,$,row}=env();
 ctx.renderSettingsDraft();
 assert.equal($('settings-save-status').textContent,'minimal.settings.saved');
 ctx.state.hasUnsaved=true;
 ctx.renderSettingsDraft();
 assert.equal($('settings-save-status').textContent,'minimal.settings.unsaved');
 assert.equal($('settings-save-status').classList.contains('dirty'),true);
 ctx.state.hasUnsaved=false;
 ctx.renderSettingsDraft();
 assert.equal($('settings-save-status').classList.contains('dirty'),false);
});
test('Connecting Twitch without Client ID expands the exact advanced input instead of saving drafts',async()=>{
 const {ctx,$,calls}=env();
 ctx.state.hasUnsaved=true;
 await ctx.action('auth');
 assert.equal(ctx.page,'settings');
 assert.equal($('settings-advanced').open,true);
 assert.equal($('clientId').focused,true);
 assert.equal(ctx.lastToast.isError,true);
 assert.equal(calls.length,0,'No network request or config update without Client ID');
 assert.equal(ctx.state.hasUnsaved,true);
});
test('All Settings fields still match persisted config contract',()=>{
 const keys=['clientId','playerName','toonHandle','replayRoot','lang','predictionWindow','predTitle','predWin','predLoss'];
 for(const key of keys){
   assert.ok(html.includes('id="'+key+'"'),'Configuration field missing: '+key);
   assert.ok(core.includes("$('"+key+"')"),'Config reader missing field: '+key);
 }
});
