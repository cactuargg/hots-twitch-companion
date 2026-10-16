'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const core=fs.readFileSync('web/assets/core.js','utf8');
const helper=core.slice(core.indexOf('function openExternalLink('),core.indexOf('\nfunction toast('));
test('Twitch links use native system browser or safe browser fallback',()=>{
 for(const native of [true,false]){
  const calls=[];const window=native?{runtime:{BrowserOpenURL:url=>calls.push([url])}}:{open:(...args)=>calls.push(args)};
  const ctx=vm.createContext({URL,window});vm.runInContext(helper,ctx);
  ctx.openExternalLink('https://www.twitch.tv/activate');
  assert.equal(calls[0][0],'https://www.twitch.tv/activate');
  if(!native)assert.deepEqual(calls[0].slice(1),['_blank','noopener,noreferrer']);
  for(const url of ['javascript:alert(1)','http://www.twitch.tv/activate','https://evil.test','https://www.twitch.tv.evil.test','https://user@www.twitch.tv','https://www.twitch.tv:444/'])assert.throws(()=>ctx.openExternalLink(url));
  assert.equal(calls.length,1);
 }
});

const desktop=fs.readFileSync('web/assets/desktop.js','utf8');
function shellHarness(native=true){
 const calls=[],classes=[];
 const titlebar={hidden:true,addEventListener:(name,fn)=>{titlebar.click=fn}};
 const window=native?{runtime:{WindowMinimise:()=>calls.push('minimize'),Quit:()=>calls.push('quit')}}:{};
 const ctx=vm.createContext({window,document:{documentElement:{classList:{add:x=>classes.push(x)}}},$:()=>titlebar,api:async(...args)=>calls.push(args),toast:(...args)=>calls.push(args)});
 vm.runInContext(desktop,ctx);ctx.initDesktopShell();return {ctx,titlebar,calls,classes};
}
test('native controls minimize, close through Wails, and hide only through guarded API',async()=>{
 const {titlebar,calls,classes}=shellHarness();
 assert.equal(titlebar.hidden,false);assert.deepEqual(classes,['desktop']);
 for(const name of ['minimize','close','tray'])await titlebar.click({target:{closest:()=>({dataset:{window:name}})}});
 assert.deepEqual(calls.slice(0,2),['minimize','quit']);assert.deepEqual(Array.from(calls[2]),['window/hide','POST']);
});
test('browser keeps native controls hidden, failed tray request reports error',async()=>{
 const browser=shellHarness(false);assert.equal(browser.titlebar.hidden,true);assert.deepEqual(browser.classes,[]);
 const {ctx,titlebar,calls}=shellHarness();ctx.api=async()=>{throw Error('Tray unavailable')};
 await titlebar.click({target:{closest:()=>({dataset:{window:'tray'}})}});
 assert.deepEqual(calls,[['Tray unavailable',true]]);
});
