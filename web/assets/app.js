'use strict';
const base = new URL('./', location.href).pathname;
const $ = id => document.getElementById(id);
const state = { status: null, config: null, replays: [], hasUnsaved: false, busy: false };
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
function getInputs(){return {clientId:$('clientId').value.trim(),playerName:$('playerName').value.trim(),playerToonHandle:$('toonHandle').value.trim(),replayRoot:$('replayRoot').value.trim(),realMode:$('realMode').checked}}
function setInputs(c){$('clientId').value=c.clientId||'';$('playerName').value=c.playerName||'';$('toonHandle').value=c.playerToonHandle||'';$('replayRoot').value=c.replayRoot||'';$('realMode').checked=!!c.realMode;state.hasUnsaved=false;}
function go(view){document.querySelectorAll('.view').forEach(v=>v.classList.toggle('active',v.id==='view-'+view));document.querySelectorAll('.nav-item').forEach(n=>n.classList.toggle('active',n.dataset.view===view));}
function line(text,cls){const d=document.createElement('div');d.className=cls||'';d.textContent=text;return d}
function relative(iso){if(!iso)return '';const ms=Date.now()-new Date(iso).getTime();if(!Number.isFinite(ms))return '';const min=Math.round(ms/60000);if(min<2)return 'только что';if(min<60)return min+' мин назад';if(min<1440)return Math.round(min/60)+' ч назад';return new Date(iso).toLocaleDateString('ru-RU')}
function resultBadge(result){const span=document.createElement('span');span.className='outcome-pill '+(result==='win'?'win':result==='loss'?'loss':'unknown');span.textContent=result==='win'?'ПОБЕДА':result==='loss'?'ПОРАЖЕНИЕ':'НЕИЗВЕСТНО';return span}
const mapPic=i=>['./assets/map-towers.webp','./assets/map-hollow.webp','./assets/map-dragon.webp'][i%3];
function showReplays(items){state.replays=Array.isArray(items)?items:[];const box=$('recent-list'),full=$('full-replays');box.replaceChildren();full.replaceChildren();if(!state.replays.length){box.append(line('Реплеи пока не найдены','empty-mini'));full.append(line('Реплеи не найдены. Проверь путь в настройках.','empty-mini'));return}
 state.replays.slice(0,3).forEach((r,i)=>{const el=document.createElement('div');el.className='recent-item';const img=document.createElement('img');img.className='recent-img';img.src=mapPic(i);img.alt='';const desc=document.createElement('div');desc.className='recent-text';desc.append(line(r.map||r.file,'').cloneNode(true));desc.firstChild.replaceWith(Object.assign(document.createElement('b'),{textContent:r.map||r.file}));desc.append(Object.assign(document.createElement('small'),{textContent:relative(r.modified)}));el.append(img,desc,resultBadge(r.result));box.append(el)});
 state.replays.forEach(r=>{const el=document.createElement('div');el.className='full-replay';const text=document.createElement('div');text.append(Object.assign(document.createElement('div'),{className:'full-replay-name',textContent:r.map||r.file}));text.append(Object.assign(document.createElement('small'),{textContent:(r.file||'')+' · '+relative(r.modified)}));el.append(text,resultBadge(r.result));full.append(el)});
}
function showLogs(logs){const arr=Array.isArray(logs)?logs:[];for(const id of ['log-preview','full-log']){const el=$(id);el.replaceChildren();const filtered=id==='log-preview'?arr.slice(-7):arr;if(!filtered.length){el.append(line('Событий пока нет','empty-mini'));continue}for(const entry of filtered){const row=document.createElement('div');row.className='log-row';const time=Object.assign(document.createElement('span'),{className:'log-time',textContent:(entry.time||'').slice(0,8)});const ico=Object.assign(document.createElement('span'),{className:'log-icon',textContent:entry.message&&/(Ошибка|Не удалось|error)/i.test(entry.message)?'!':'◆'});const msg=Object.assign(document.createElement('span'),{className:'log-msg',textContent:entry.message||''});row.append(time,ico,msg);el.append(row)}}}
function render(s){state.status=s;const connected=!!s.connected;const running=!!s.running;const active=!!s.active;const name=s.channel||'';const simulation=!s.realMode;
 $('top-connection').classList.toggle('connected',connected);$('top-connected-text').textContent=connected?'Twitch подключён':'Нет подключения';$('dash-channel').textContent=connected?(name?'@'+name:'Авторизован'):'Не подключён';$('channel-pill').textContent=connected?'CONNECTED':'OFFLINE';$('channel-pill').classList.toggle('online',connected);
 $('dash-player').textContent=s.playerName||s.playerToonHandle||'Не указан';$('dash-prediction').textContent=active?(s.activeDryRun?'Симуляция':'Активен'):'Нет';$('dash-result').textContent=s.lastResult||'Нет данных';$('match-status').textContent=active?'Матч в процессе':running?'Ожидание нового матча':'Мониторинг выключен';$('match-description').textContent=active?'Prediction открыт. Ожидаем завершения игры и появления реплея.':running?'Программа следит за новыми lobby и готовыми реплеями.':'Нажми «Включить автоматику», чтобы начать отслеживание.';$('match-mini').textContent=active?'Матч активен':running?'Мониторинг работает':'Ожидание матча';$('prediction-state').textContent=active?'Идёт прогноз':'Неактивен';$('prediction-help').textContent=active?'Ожидаем результат игры.':simulation?'Режим симуляции: баллы не задействованы.':'Ожидание нового матча.';
 $('watch-tag').classList.toggle('on',running);$('watch-tag').innerHTML=running?'<span class="pulse-dot"></span> Мониторинг работает':'<span class="pulse-dot"></span> Мониторинг выключен';$('start-label').textContent=running?'Остановить автоматику':'Включить автоматику';$('automation-title').textContent=running?'Автоматизация включена':'Автоматизация выключена';$('automation-detail').textContent=running?'Ожидание событий из HotS. Отслеживаются только игровые файлы.':'Программа не отслеживает новые матчи.';
 $('automation-mode').textContent=simulation?'Режим симуляции':'РЕАЛЬНЫЕ PREDICTIONS';$('automation-mode-detail').textContent=simulation?'Twitch Predictions не создаются и баллы не распределяются.':'Баллы Twitch реально распределяются по итогам матчей.';$('footer-mode').textContent=simulation?'Режим симуляции включён':'ВНИМАНИЕ: реальные Predictions включены';$('footer-detail').textContent=simulation?'Реальные Predictions не создаются. Игровые файлы только читаются.':'Приложение создаёт и рассчитывает настоящие ставки на Twitch.';$('footer-status').textContent=running?'Мониторинг включён':'Ожидание';$('twitch-info').textContent=connected?'Аккаунт подключён: @'+name+'. Разрешение channel:manage:predictions.':'Нет подключённого Twitch аккаунта.';
 if(s.authCode){$('auth-progress').hidden=false;$('auth-code').textContent=s.authCode;$('auth-info').textContent=s.authStatus||'Введи код на странице Twitch.'}else if(s.authPending){$('auth-progress').hidden=false;$('auth-info').textContent='Ожидаем подтверждения Twitch...';}else{$('auth-progress').hidden=true}
 showLogs(s.logs);
}
async function loadStatus(){try{render(await api('status'))}catch(e){/* app closed, keep last UI */}}
async function reloadReplays(){try{const x=await api('replays');showReplays(x.replays||[])}catch(e){toast(e.message,true)}}
async function save(){const data=getInputs();if(data.realMode&&!state.config?.realMode){if(!confirm('Включить НАСТОЯЩИЕ Twitch Predictions?\n\nЭто создаст и автоматически рассчитает реальные прогнозы с баллами зрителей. Рекомендуется сначала проверить реплей в симуляции.')){data.realMode=false;$('realMode').checked=false}}
 const x=await api('config','POST',{...data,confirmedRealMode:data.realMode});state.config=x.config;state.hasUnsaved=false;toast('Настройки сохранены');await loadStatus();return x}
async function action(name){if(state.busy&&name!=='quit')return;state.busy=true;try{switch(name){
 case 'save':await save();break;
 case 'auth':{if(state.hasUnsaved)await save();if(!$('clientId').value.trim()){go('twitch');toast('Сначала введи Client ID Twitch',true);break}const x=await api('auth','POST',{});if(x.code){go('twitch');$('auth-progress').hidden=false;$('auth-code').textContent=x.code;code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';toast('Код Twitch получен. Открой страницу активации.')}break}
 case 'activate':window.open(code.authUrl,'_blank','noopener,noreferrer');break;
 case 'developer':window.open('https://dev.twitch.tv/console/apps','_blank','noopener,noreferrer');break;
 case 'replay':{const x=await api('replay/check','POST',{});go('replays');const el=$('replay-detail');el.hidden=false;el.textContent='Файл: '+x.file+'\nКарта: '+x.map+'\n\n'+(x.players||[]).map(p=>p.name+' · '+p.toonHandle+' · '+p.result).join('\n')+'\n\n'+(x.selected?'ВЫБРАННЫЙ ИГРОК: '+x.selected.name+' → '+x.selected.result:'Нужный игрок не определён: проверь ник/Toon Handle.');toast('Реплей прочитан');await reloadReplays();break}
 case 'start':if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast('Мониторинг запущен');await loadStatus();break;
 case 'stop':await api('automation/stop','POST',{});toast('Мониторинг останавливается');await loadStatus();break;
 case 'toggle':if(state.status?.running){await api('automation/stop','POST',{});toast('Мониторинг останавливается')}else{if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast('Мониторинг запущен')}await loadStatus();break;
 case 'refresh-replays':await reloadReplays();toast('Список обновлён');break;
 case 'quit':if(!confirm('Закрыть HotS Twitch Companion? Мониторинг будет остановлен.'))break;await api('quit','POST',{});window.close();document.body.innerHTML='<div style="font:18px Segoe UI;color:#ccdaee;margin:70px;text-align:center">HotS Twitch Companion завершён. Это окно можно закрыть.</div>';break;
 }}catch(e){toast(e.message||'Ошибка запроса',true)}finally{state.busy=false}}
window.addEventListener('DOMContentLoaded',async()=>{
 document.querySelectorAll('[data-view]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.view)));
 document.querySelectorAll('[data-goto]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.goto)));
 document.querySelectorAll('[data-action]').forEach(el=>el.addEventListener('click',()=>action(el.dataset.action)));
 for(const id of ['clientId','playerName','toonHandle','replayRoot','realMode'])$(id).addEventListener('input',()=>state.hasUnsaved=true);
 try{const b=await api('bootstrap');state.config=b.config;setInputs(b.config);render(b.status);showReplays(b.replays||[])}catch(e){toast('Не удалось подключиться к приложению: '+e.message,true)}
 setInterval(loadStatus,2500);setInterval(reloadReplays,30000);
});
