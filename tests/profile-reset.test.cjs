'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm');
const app=fs.readFileSync('web/assets/app.js','utf8');
const action=app.slice(app.indexOf('async function action(name)'),app.indexOf('async function checkReplayRoot()'));
function harness(confirmed,failed=false){
 const calls=[];
 const ctx=vm.createContext({state:{busy:false},setActionBusy(){},confirm:()=>confirmed,t:x=>x,
  api:async(...args)=>{calls.push(args);if(failed)throw Error('blocked')},
  localStorage:{clear:()=>calls.push('clear')},document:{body:{replaceChildren:()=>calls.push('message')}},line:x=>x,toast:x=>calls.push(x)});
 vm.runInContext(action,ctx);return {ctx,calls};
}
test('reset cancellation leaves API and local browser data untouched',async()=>{
 const {ctx,calls}=harness(false);await ctx.action('reset-profile');assert.deepEqual(calls,[]);
});
test('confirmed reset clears browser preferences only after server accepts it',async()=>{
 const {ctx,calls}=harness(true);await ctx.action('reset-profile');
 assert.equal(calls[0][0],'profile/reset');assert.equal(calls[0][1],'POST');assert.equal(calls[0][2].confirmed,true);
 assert.deepEqual(calls.slice(1),['clear','message']);
 const refused=harness(true,true);await refused.ctx.action('reset-profile');assert.equal(refused.calls.includes('clear'),false);
});
