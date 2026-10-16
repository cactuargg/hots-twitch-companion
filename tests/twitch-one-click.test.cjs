'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const root=path.join(__dirname,'..');
const read=p=>fs.readFileSync(path.join(root,p),'utf8');
const app=read('web/assets/app.js');
const onboarding=read('web/assets/onboarding.js');
const workflow=read('.github/workflows/windows.yml');
const html=read('web/index.html');
const start=app.indexOf('async function action(name)');
const end=app.indexOf('async function checkReplayRoot()',start);
assert.ok(start>=0&&end>start);

function env(builtIn) {
 const elements=new Map(),calls=[];
 const $=id=>{
  if(!elements.has(id)){
   elements.set(id,{value:'',open:false,hidden:false,textContent:'',focused:false,
     focus(){this.focused=true}});
  }
  return elements.get(id);
 };
 const ctx={
  state:{status:{hasTwitchApp:builtIn},busy:false,hasUnsaved:false},
  $,setActionBusy(){},t:x=>x,
  code:{authUrl:''},
  go:page=>{ctx.page=page},
  toast:(message,isError)=>{ctx.toastValue={message,isError}},
  api:async(route,method,payload)=>{
   calls.push({route,method,payload});
   return {code:'TWITCH123',verificationUri:'https://www.twitch.tv/activate'};
  }
 };
 vm.createContext(ctx);
 vm.runInContext(app.slice(start,end),ctx);
 return {ctx,calls,$};
}
test('one-click Twitch Device Code does not require user Client ID when EXE has its own public app',async()=>{
 const {ctx,calls,$}=env(true);
 await ctx.action('auth');
 assert.equal(calls.length,1);
 assert.equal(calls[0].route,'auth');
 assert.equal(calls[0].method,'POST');
 assert.equal(ctx.page,'settings');
 assert.equal($('auth-progress').hidden,false);
 assert.equal($('auth-code').textContent,'TWITCH123');
 assert.equal($('settings-advanced').open,false);
 assert.equal($('clientId').focused,false);
});
test('standalone builds without configured app still guide the user to Client ID field',async()=>{
 const {ctx,calls,$}=env(false);
 await ctx.action('auth');
 assert.equal(calls.length,0);
 assert.equal($('settings-advanced').open,true);
 assert.equal($('clientId').focused,true);
 assert.equal(ctx.toastValue.isError,true);
});
test('explicit Client ID still allows alternative Twitch app when there is no shared app',async()=>{
 const {ctx,calls,$}=env(false);
 $('clientId').value='custom-client';
 await ctx.action('auth');
 assert.equal(calls.length,1);
 assert.equal($('settings-advanced').open,false);
});
test('wizard hides advanced Client ID when app login works and preserves developer override',()=>{
 assert.match(onboarding,/!id&&!state\.status\?\.hasTwitchApp/);
 assert.match(onboarding,/wz-twitch-details'\)\.hidden=/);
 assert.ok(html.includes('id="twitch-client-hint"'));
});
test('Windows release builds embed only a public repository variable, no secret',()=>{
 assert.ok(workflow.includes('vars.TWITCH_CLIENT_ID'));
 assert.ok(workflow.includes('-X main.defaultTwitchClientID=$publicId'));
 assert.ok(!workflow.includes('TWITCH_CLIENT_SECRET'));
 assert.ok(!workflow.includes('secrets.TWITCH_CLIENT_ID'));
});
