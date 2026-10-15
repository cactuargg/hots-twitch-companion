'use strict';
const base = new URL('./', location.href).pathname;
const $ = id => document.getElementById(id);
const state = { status: null, config: null, replays: [], logs: [], expandedReplays: new Set(), hasUnsaved: false, busy: false };
const code = { authUrl: 'https://www.twitch.tv/activate' };
let toastTimer;

function toast(message, error = false){ const n=$('toast'); n.textContent=message; n.className='toast show'+(error?' error':''); clearTimeout(toastTimer); toastTimer=setTimeout(()=>n.className='toast',4600); }
function safe(v){return v==null?'':String(v)}
async function api(route, method='GET', data){
 const options={method,headers:{'X-Companion':'1'},cache:'no-store'};
 if(data!==undefined){options.headers['Content-Type']='application/json';options.body=JSON.stringify(data)}
 const controller=new AbortController(); const timeout=setTimeout(()=>controller.abort(),45000); options.signal=controller.signal;
 try{const r=await fetch(base+'api/'+route,options);const x=await r.json();if(!r.ok||x.error)throw Error(x.error||'HTTP '+r.status);return x}
 finally{clearTimeout(timeout)}
}
function getInputs(){return {clientId:$('clientId').value.trim(),playerName:$('playerName').value.trim(),playerToonHandle:$('toonHandle').value.trim(),replayRoot:$('replayRoot').value.trim(),realMode:$('realMode').checked,lang:$('lang').value,predictionWindowMin:parseInt($('predictionWindow').value,10)||2,wizardDone:!!(state.config&&state.config.wizardDone),predictionTitle:$('predTitle').value.trim(),predictionWin:$('predWin').value.trim(),predictionLoss:$('predLoss').value.trim()}}
function setInputs(c){$('clientId').value=c.clientId||'';$('playerName').value=c.playerName||'';$('toonHandle').value=c.playerToonHandle||'';$('replayRoot').value=c.replayRoot||'';$('realMode').checked=!!c.realMode;$('lang').value=c.lang||lang;$('predictionWindow').value=String(c.predictionWindowMin||2);$('predTitle').value=c.predictionTitle||'';$('predWin').value=c.predictionWin||'';$('predLoss').value=c.predictionLoss||'';state.hasUnsaved=false;renderPredictionPreviews();renderPlayerPickers();}
function go(view){document.querySelectorAll('.view').forEach(v=>v.classList.toggle('active',v.id==='view-'+view));document.querySelectorAll('.tab').forEach(n=>n.classList.toggle('active',n.dataset.view===view));}
function line(text,cls){const d=document.createElement('div');d.className=cls||'';d.textContent=text;return d}
function relative(iso){if(!iso)return '';const ms=Date.now()-new Date(iso).getTime();if(!Number.isFinite(ms))return '';const min=Math.round(ms/60000);if(min<2)return t('rel.now');if(min<60)return t('rel.min',{n:min});if(min<1440)return t('rel.hour',{n:Math.round(min/60)});return new Date(iso).toLocaleDateString(lang==='ru'?'ru-RU':'en-US')}
function resultBadge(result){const span=document.createElement('span');span.className='outcome-pill '+(result==='win'?'win':result==='loss'?'loss':'unknown');span.textContent=result==='win'?t('badge.win'):result==='loss'?t('badge.loss'):t('badge.unknown');return span}
