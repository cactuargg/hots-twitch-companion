'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const path=require('node:path');
const vm=require('node:vm');

const root=path.join(__dirname,'..');
const read=p=>fs.readFileSync(path.join(root,p),'utf8');
const app=read('web/assets/app.js');
const base=read('web/assets/style.css');
const overview=read('web/assets/overview.css');
const settings=read('web/assets/settings.css');
const wizard=read('web/assets/wizard.css');
const onboarding=read('web/assets/onboarding.css');
const diagnostics=read('web/assets/diagnostics.css');
const windows=read('main_windows.go');

function slice(start,end) {
 const a=app.indexOf(start),b=app.indexOf(end,a+start.length);
 assert.ok(a>=0&&b>a,'missing '+start);
 return app.slice(a,b);
}
function fakeButton(action,primary=true){
 const attrs=new Map();
 return {
   dataset:{action},disabled:false,
   attrs,classList:{contains:cls=>primary&&cls==='btn'},
   setAttribute:(key,val)=>attrs.set(key,val),
   removeAttribute:key=>attrs.delete(key)
 };
}
function busyHarness(){
 const buttons=[fakeButton('toggle'),fakeButton('save'),fakeButton('quit')];
 const document={querySelectorAll:()=>buttons};
 const ctx={document};
 vm.createContext(ctx);
 vm.runInContext(slice('function setActionBusy(', 'function render(s){'),ctx);
 return {ctx,buttons};
}

test('busy feedback marks only the activated button and disables competing actions',()=>{
 const {ctx,buttons}=busyHarness();
 ctx.setActionBusy('toggle',true);
 assert.equal(buttons[0].disabled,true);
 assert.equal(buttons[0].attrs.get('aria-busy'),'true');
 assert.equal(buttons[1].disabled,true);
 assert.equal(buttons[1].attrs.has('aria-busy'),false);
 assert.equal(buttons[2].disabled,false,'Quit remains operable');
 // A status refresh may change action=toggle to action=wizard during the request.
 buttons[0].dataset.action='wizard';
 ctx.setActionBusy('toggle',false);
 assert.equal(buttons[0].attrs.has('aria-busy'),false);
 assert.equal(buttons[0].disabled,false);
 assert.equal(buttons[1].disabled,false);
});

test('failed action always restores button state, including loading indicator',async()=>{
 const {ctx,buttons}=busyHarness();
 let rejectRequest;
 const pending=new Promise((_,reject)=>{rejectRequest=reject;});
 ctx.state={busy:false};
 ctx.save=()=>pending;
 ctx.toast=(message,err)=>{ctx.feedback={message,err}};
 ctx.t=k=>k;
 ctx.api=async()=>({});
 ctx.$=()=>({value:''});
 vm.runInContext(slice('async function action(name)', 'async function checkReplayRoot()'),ctx);
 const task=ctx.action('save');
 assert.equal(ctx.state.busy,true);
 assert.equal(buttons[1].disabled,true);
 assert.equal(buttons[1].attrs.get('aria-busy'),'true');
 rejectRequest(new Error('Network unavailable'));
 await task;
 assert.equal(ctx.state.busy,false);
 assert.equal(buttons[1].disabled,false);
 assert.equal(buttons[1].attrs.has('aria-busy'),false);
 assert.equal(ctx.feedback.err,true);
});

test('compact Windows window replaces the original oversized 1440 by 950 launcher',()=>{
 assert.match(windows,/--window-size=1024,800/);
 assert.doesNotMatch(windows,/--window-size=1440,950/);
});

test('all key surfaces reflow at effective widths for Windows 125-150 percent scaling',()=>{
 for(const [filename,css] of [
 ['base',base],['overview',overview],['settings',settings],['wizard',wizard],
 ['onboarding',onboarding],['diagnostics',diagnostics]
 ]) {
   assert.ok(css.includes('@media(max-width:'),'Responsive rules missing in '+filename);
 }
 assert.match(base,/@media\(max-width:520px\)/);
 assert.match(base,/@media\(max-width:360px\)/);
 assert.match(overview,/\.overview-command-actions\{align-items:stretch;flex-direction:column\}/);
 assert.match(settings,/\.settings-savebar \.form-buttons\{width:100%;display:grid/);
 assert.match(wizard,/\.wizard-head\{align-items:flex-start;flex-wrap:wrap\}/);
 assert.match(onboarding,/minmax\(min\(140px,100%\),1fr\)/);
 assert.match(diagnostics,/overflow-wrap:anywhere/);
});

test('controls have focus, pressed, disabled, busy and reduced-motion treatments',()=>{
 assert.match(base,/:focus-visible/);
 assert.match(base,/\.btn:active:not\(:disabled\)/);
 assert.match(base,/\.btn:disabled/);
 assert.match(base,/\.btn\[aria-busy="true"\]::after/);
 assert.match(base,/@media\(prefers-reduced-motion:reduce\)/);
 assert.match(overview,/\.overview-toggle\.is-stop/);
 assert.match(app,/mainButton\.classList\.toggle\('is-stop',running\)/);
});
