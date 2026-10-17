'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const app=fs.readFileSync('cmd/companion/web/assets/app.js','utf8');
const i18n=fs.readFileSync('cmd/companion/web/assets/i18n.js','utf8');
const html=fs.readFileSync('cmd/companion/web/index.html','utf8');
const action=app.slice(app.indexOf('async function action(name)'),app.indexOf('async function checkReplayRoot()'));

test('footer badge exists and is wired to the update action',()=>{
 assert.match(html,/id="update-badge"[^>]*data-action="update"/);
 assert.match(html,/id="update-badge"[^>]*hidden/);
});

test('render shows the badge only when both version and URL are known',()=>{
 const badge={hidden:true,textContent:''};
 const generic=()=>({textContent:'',hidden:true,value:'',classList:{toggle(){}},dataset:{}});
 const ctx=vm.createContext({
  $:id=>id==='update-badge'?badge:generic(),
  document:{querySelector:()=>({dataset:{},classList:{toggle(){}}})},
  t:(k,v)=>k==='update.available'?'Update available: v'+v.version:'',
  state:{replays:[]},
  monitoringView:()=>({phase:'off',prediction:'none',twitch:'disconnected'}),
  renderSettingsDraft(){},showLogs(){},
 });
 const render=app.slice(app.indexOf('function render(s)'),app.indexOf('async function loadStatus()'));
 vm.runInContext(render,ctx);
 vm.runInContext('render({version:"0.9.3",updateVersion:"0.9.4",updateURL:"https://example.com"})',ctx);
 assert.equal(badge.hidden,false);
 assert.equal(badge.textContent,'Update available: v0.9.4');
 vm.runInContext('render({version:"0.9.3"})',ctx);
 assert.equal(badge.hidden,true,'badge must hide without update info');
});

test('update action opens the release page in the system browser',async()=>{
 const opened=[];
 const ctx=vm.createContext({state:{status:{updateURL:'https://example.com/release'}},setActionBusy(){},
  openExternalLink:url=>opened.push(url),toast(){},confirm:()=>true});
 vm.runInContext(action,ctx);
 await ctx.action('update');
 assert.deepEqual(opened,['https://example.com/release']);
});

test('localized badge labels exist for both languages',()=>{
 assert.match(i18n,/'update\.available':'Update available: v\{version\} ↗'/);
 assert.match(i18n,/'update\.available':'Доступна версия: v\{version\} ↗'/);
});
