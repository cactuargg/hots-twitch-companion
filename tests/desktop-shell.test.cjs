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
