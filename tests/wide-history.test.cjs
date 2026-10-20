'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),vm=require('node:vm'),fs=require('node:fs'),path=require('node:path');
function helpers(){const context={I18N:{ru:{},en:{}},window:{addEventListener(){}},renderObs(){},setInputs(){},t:key=>key};vm.createContext(context);vm.runInContext(fs.readFileSync(path.join(__dirname,'../internal/dashboard/web/assets/wide.js'),'utf8'),context);return context}
test('pagination changes visible rows without changing aggregate win rate',()=>{
 const c=helpers(),items=Array.from({length:35},(_,i)=>({id:String(i),hero:i%2?'Jaina':'Valla',map:'Hanamura',result:i%3?'win':'loss',recordedAt:i+1,mode:50091}));
 const first=c.widePaginate(items,1,10),last=c.widePaginate(items,99,10);assert.equal(first.items.length,10);assert.equal(last.current,4);assert.equal(last.items.length,5);assert.equal(last.offset,30);
 const stats=c.wideStatistics(items,'hero');assert.equal(stats.wins+stats.losses,35);assert.equal(stats.groups.reduce((n,g)=>n+g.wins+g.losses,0),35);
});
test('time, mode, hero and session filters compose and imported history cannot become a current session',()=>{
 const c=helpers(),now=2000000000000,items=[{id:'active',sessionId:'s',hero:'Jaina',map:'A',mode:50091,result:'win',recordedAt:now-1},{id:'pending',sessionId:'',hero:'Jaina',map:'B',mode:50091,result:'loss',needsAssignment:true,recordedAt:now-2},{id:'old',hero:'Valla',map:'A',mode:50001,result:'win',historical:true,recordedAt:now-40*86400000}];
 const filter={period:'session',mode:'all',hero:'',map:'',query:''};assert.equal(c.wideFilterMatches(items,filter,{id:'s'},now).length,1);assert.equal(c.wideFilterMatches(items,filter,null,now).length,0);
 filter.period='all';assert.equal(c.wideFilterMatches(items,filter,null,now).length,3);filter.period='week';assert.equal(c.wideFilterMatches(items,filter,null,now).length,2);filter.hero='Valla';assert.equal(c.wideFilterMatches(items,filter,null,now).length,0);filter.period='all';filter.mode='50001';assert.equal(c.wideFilterMatches(items,filter,null,now)[0].id,'old');
});
test('excluded and unknown outcomes never enter the win-rate denominator',()=>{
 const c=helpers(),stats=c.wideStatistics([{hero:'A',result:'win'},{hero:'A',result:'loss'},{hero:'A',result:'unknown'},{hero:'A',result:'win',excluded:true}],'hero');assert.equal(stats.wins,1);assert.equal(stats.losses,1);assert.equal(stats.rate,50);assert.equal(c.wideStatistics([],'hero').rate,null);
});
