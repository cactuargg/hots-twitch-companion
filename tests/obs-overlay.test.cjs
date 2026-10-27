'use strict';
const test=require('node:test'),assert=require('node:assert/strict'),fs=require('node:fs'),vm=require('node:vm'),path=require('node:path');
const script=fs.readFileSync(path.join(__dirname,'../internal/dashboard/web/assets/overlay.js'),'utf8');
function environment(kind){
 const nodes=new Map(),node=id=>{if(!nodes.has(id))nodes.set(id,{hidden:true,style:{},textContent:'',removeAttribute(name){delete this[name]}});return nodes.get(id)};
 const context={location:{pathname:'/secret/'+kind},document:{getElementById:node},fetch:()=>Promise.reject(Error('offline')),AbortSignal:{timeout:()=>({})},setInterval:()=>{}};
 vm.createContext(context);vm.runInContext(script,context);return{context,node};
}
test('OBS mask starts opaque and returns to opaque if its state server disconnects',async()=>{
 const {context,node}=environment('mask');assert.equal(node('mask').hidden,false);
 await vm.runInContext('poll()',context);assert.equal(node('mask').hidden,false);
 const state={preferences:{enabled:true,style:'nexus',scale:100,backdrop:true,labels:true,winColor:'#72dcff',lossColor:'#ff91b2'},wins:7,losses:3,maskVisible:false,maskImageURL:'image?rev=a'};
 context.value=state;vm.runInContext('draw(value)',context);assert.equal(node('mask').hidden,true);assert.equal(node('score').hidden,true);assert.equal(node('mask-image').src,'image?rev=a');
 await vm.runInContext('poll()',context);assert.equal(node('mask').hidden,false);
});
test('OBS score survives disconnect, applies appearance and disables independently of the mask',async()=>{
 const {context,node}=environment('score');context.value={preferences:{enabled:true,style:'minimal',scale:150,backdrop:false,labels:false,winColor:'#ffffff',lossColor:'#ff0000'},wins:8,losses:2,maskVisible:true};
 vm.runInContext('draw(value)',context);assert.equal(node('score').hidden,false);assert.equal(node('wins').textContent,8);assert.equal(node('score').style.transform,'scale(1.5)');assert.equal(node('win-label').hidden,true);assert.equal(node('mask').hidden,true);
 await vm.runInContext('poll()',context);assert.equal(node('wins').textContent,8);
 context.value.preferences.enabled=false;vm.runInContext('draw(value)',context);assert.equal(node('score').hidden,true);
});

test('custom CSS refreshes only the score stylesheet and clears without a page reload',()=>{
 const {context,node}=environment('score');
 context.value={preferences:{enabled:true,style:'nexus',scale:100,labels:true,customCSS:'#wins{color:gold!important}'},wins:7,losses:3};
 vm.runInContext('draw(value)',context);
 assert.match(node('custom-css').href,/^custom\.css\?rev=/);
 node('custom-css').href='unchanged';vm.runInContext('draw(value)',context);assert.equal(node('custom-css').href,'unchanged');
 context.value.preferences.customCSS='';vm.runInContext('draw(value)',context);assert.match(node('custom-css').href,/^custom\.css\?rev=/);
 const mask=environment('mask');mask.context.value=context.value;vm.runInContext('draw(value)',mask.context);assert.equal(mask.node('custom-css').href,undefined);
});
