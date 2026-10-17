'use strict';
const base = new URL('./', location.href).pathname;
const $ = id => document.getElementById(id);
const state = { status: null, config: null, replays: [], logs: [], expandedReplays: new Set(), hasUnsaved: false, busy: false };
const code = { authUrl: 'https://www.twitch.tv/activate' };
let toastTimer;

// Keep authentication pages outside the embedded application origin.
function openExternalLink(raw){
 const url=new URL(raw);
 if(url.protocol!=='https:'||url.username||url.password||url.port||
   !['www.twitch.tv','twitch.tv','dev.twitch.tv'].includes(url.hostname)&&
    !(url.hostname==='github.com'&&url.pathname.startsWith('/cactuargg/hots-twitch-companion/releases/')))throw Error('External URL refused');
 if(window.runtime?.BrowserOpenURL)window.runtime.BrowserOpenURL(url.href);
 else window.open(url.href,'_blank','noopener,noreferrer');
}

function toast(message, error = false){ const n=$('toast'); n.textContent=message; n.className='toast show'+(error?' error':''); clearTimeout(toastTimer); toastTimer=setTimeout(()=>n.className='toast',4600); }
function safe(v){return v==null?'':String(v)}
async function api(route, method='GET', data){
 const options={method,headers:{'X-Companion':'1'},cache:'no-store'};
 if(data!==undefined){options.headers['Content-Type']='application/json';options.body=JSON.stringify(data)}
 const controller=new AbortController(); const timeout=setTimeout(()=>controller.abort(),45000); options.signal=controller.signal;
 try{const r=await fetch(base+'api/'+route,options);const x=await r.json();if(!r.ok||x.error)throw Error(x.error||'HTTP '+r.status);return x}
 finally{clearTimeout(timeout)}
}
function getInputs(){return {clientId:$('clientId').value.trim(),playerName:$('playerName').value.trim(),playerToonHandle:$('toonHandle').value.trim(),replayRoot:$('replayRoot').value.trim(),realMode:true,lang:$('lang').value,predictionWindowMin:parseInt($('predictionWindow').value,10)||2,wizardDone:!!(state.config&&state.config.wizardDone),predictionTitle:$('predTitle').value.trim(),predictionWin:$('predWin').value.trim(),predictionLoss:$('predLoss').value.trim()}}
function setInputs(c){$('clientId').value=c.clientId||'';$('playerName').value=c.playerName||'';$('toonHandle').value=c.playerToonHandle||'';$('replayRoot').value=c.replayRoot||'';$('lang').value=c.lang||lang;$('predictionWindow').value=String(c.predictionWindowMin||2);$('predTitle').value=c.predictionTitle||'';$('predWin').value=c.predictionWin||'';$('predLoss').value=c.predictionLoss||'';state.hasUnsaved=false;renderPredictionPreviews();renderPlayerPickers();renderSettingsDraft();}
function go(view, options={}){
 const tabs=Array.from(document.querySelectorAll('.tab'));
 if(!tabs.some(tab=>tab.dataset.view===view))return;
 document.querySelectorAll('.view').forEach(panel=>panel.classList.toggle('active',panel.id==='view-'+view));
 for(const tab of tabs){
   const active=tab.dataset.view===view;
   tab.classList.toggle('active',active);
   tab.setAttribute('aria-selected',String(active));
   tab.tabIndex=active?0:-1;
 }
 if(options.focus)tabs.find(tab=>tab.dataset.view===view).focus();
}
function handleTabKeyDown(event){
 const keys=['ArrowRight','ArrowLeft','Home','End'];
 if(!keys.includes(event.key))return;
 const tabs=Array.from(document.querySelectorAll('.tab'));
 const index=tabs.indexOf(event.target);
 if(index<0)return;
 event.preventDefault();
 const target=event.key==='Home'?0:event.key==='End'?tabs.length-1:
   (index+(event.key==='ArrowRight'?1:-1)+tabs.length)%tabs.length;
 go(tabs[target].dataset.view,{focus:true});
}
function line(text,cls){const d=document.createElement('div');d.className=cls||'';d.textContent=text;return d}
function relative(iso){if(!iso)return '';const ms=Date.now()-new Date(iso).getTime();if(!Number.isFinite(ms))return '';const min=Math.round(ms/60000);if(min<2)return t('rel.now');if(min<60)return t('rel.min',{n:min});if(min<1440)return t('rel.hour',{n:Math.round(min/60)});return new Date(iso).toLocaleDateString(lang==='ru'?'ru-RU':'en-US')}
function resultBadge(result){const span=document.createElement('span');span.className='outcome-pill '+(result==='win'?'win':result==='loss'?'loss':'unknown');span.textContent=result==='win'?t('badge.win'):result==='loss'?t('badge.loss'):t('badge.unknown');return span}
