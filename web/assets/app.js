'use strict';
const base = new URL('./', location.href).pathname;
const $ = id => document.getElementById(id);
const state = { status: null, config: null, replays: [], hasUnsaved: false, busy: false };
const code = { authUrl: 'https://www.twitch.tv/activate' };
let toastTimer;

/* ── i18n ────────────────────────────────────────────────────────────── */
const I18N = {
 en: {
  'nav.dashboard':'Overview','nav.replays':'Matches','nav.automation':'Automation','nav.logs':'Logs','nav.settings':'Settings',
  'dash.eyebrow':'CONTROL CENTER','dash.title':'Overview','dash.sub':'HotS, Twitch connection and automated prediction status.',
  'match.title':'Match status','match.imgalt':'Fantasy battleground',
  'info.player':'Player','info.prediction':'Active prediction','info.result':'Last result',
  'recent.title':'Recent matches','recent.all':'All matches →','recent.loading':'Loading saved replays…',
  'log.title':'Event log','log.all':'Full log →',
  'twitch.eyebrow':'CONNECTIONS','twitch.sub':'Link your channel with official OAuth. Your password is never sent to the app.',
  'twitch.h2':'Connect your Twitch account','twitch.p':'You need a Client ID of a Public app from the Twitch Developer Console. No client secret is required.',
  'twitch.placeholder':'Paste your Client ID','twitch.console':'Twitch Console ↗','twitch.connect':'Connect Twitch','twitch.settings.title':'Twitch settings',
  'auth.label':'Authorization code:','auth.activate':'Open Twitch Activate ↗',
  'replays.eyebrow':'MATCH HISTORY','replays.title':'Matches & replays','replays.sub':'Verify results without starting a match or calling the Twitch API.',
  'replays.refresh':'Refresh','replays.h2':'Recently saved matches','replays.check':'Check latest replay',
  'automation.eyebrow':'AUTOMATION','automation.title':'Prediction control','automation.sub':'Auto-start on lobby detection and auto-resolve on the match result.',
  'automation.start':'Start monitoring','automation.stop':'Stop','automation.how':'How it works',
  'automation.step1.pre':'A new ','automation.step2':'A “HotS win?” prediction opens (betting window {min} min)',
  'automation.step3':'Once the replay appears, the selected player’s result is read',
  'automation.step4':'The prediction resolves with the outcome or is cancelled on timeout',
  'logs.eyebrow':'DIAGNOSTICS','logs.title':'Event log','logs.sub':'Local events and API results only — no secret tokens.',
  'settings.eyebrow':'CONFIGURATION','settings.title':'Settings','settings.sub':'Settings are stored in your Windows profile. Game files are never modified.',
  'settings.interface':'Interface','settings.lang':'Interface language',
  'settings.player':'Player name','settings.player.ph':'Name as it appears in replays',
  'settings.toon':'Toon Handle (optional)','settings.toon.ph':'Exact player identifier',
  'settings.root':'Replays folder (can stay empty)','settings.root.ph':'Searches Documents / Heroes of the Storm',
  'settings.mode':'Operation mode','settings.window':'Prediction duration',
  'settings.window.hint':'Betting window, same as the Twitch Predictions interface (1–30 minutes).',
  'settings.real':'Allow real Twitch Predictions','settings.real.sub':'Only after verifying the parser: real viewer points will be distributed automatically.',
  'settings.save':'Save settings',
  'footer.quit':'Exit','footer.quit.title':'Close the app','lang.switch.title':'Switch language','unit.min':'min',
  'top.connected':'Twitch connected','top.disconnected':'Not connected',
  'dash.channel.authorized':'Authorized','dash.channel.none':'Not connected','dash.player.empty':'Not set',
  'dash.prediction.sim':'Simulation','dash.prediction.active':'Active','dash.prediction.none':'None','dash.result.none':'No data',
  'match.status.active':'Match in progress','match.status.waiting':'Waiting for a new match','match.status.off':'Monitoring is off',
  'match.desc.active':'The prediction is open. Waiting for the game to finish and the replay to appear.',
  'match.desc.waiting':'Watching for new lobbies and finished replays.',
  'match.desc.off':'Click “Enable automation” to start tracking.',
  'match.mini.active':'Match active','match.mini.running':'Monitoring active','match.mini.waiting':'Waiting for a match',
  'pred.state.label':'PREDICTION STATE','pred.state.active':'Prediction running','pred.state.inactive':'Inactive',
  'pred.help.active':'Waiting for the match result.','pred.help.sim':'Simulation mode: no points involved.','pred.help.wait':'Waiting for a new match.',
  'watch.on':'Monitoring active','watch.off':'Monitoring is off','start.stop':'Stop automation','start.start':'Enable automation',
  'automation.h2.on':'Automation enabled','automation.h2.off':'Automation disabled',
  'automation.detail.on':'Waiting for HotS events. Only game files are monitored.','automation.detail.off':'The app is not watching for new matches.',
  'mode.sim':'Simulation mode','mode.real':'REAL PREDICTIONS',
  'mode.sim.detail':'No Twitch Predictions are created and no points are distributed.','mode.real.detail':'Twitch points are actually distributed based on match results.',
  'footer.mode.sim':'Simulation mode enabled','footer.mode.real':'WARNING: real Predictions enabled',
  'footer.detail.sim':'Real Predictions are not created. Game files are read-only.','footer.detail.real':'The app creates and resolves real Twitch predictions.',
  'footer.status.on':'Monitoring on','footer.status.off':'Standby',
  'twitch.info.on':'Account connected: @{name}. Permission channel:manage:predictions.','twitch.info.off':'No Twitch account connected.',
  'auth.info.default':'Enter the code on the Twitch page.','auth.info.pending':'Waiting for Twitch confirmation…',
  'replays.empty.recent':'No replays found yet','replays.empty.full':'No replays found. Check the folder in Settings.',
  'logs.empty':'No events yet','logs.empty.full':'Log is empty',
  'badge.win':'WIN','badge.loss':'LOSS','badge.unknown':'UNKNOWN',
  'rel.now':'just now','rel.min':'{n} min ago','rel.hour':'{n} h ago',
  'toast.saved':'Settings saved','toast.need_client':'Enter your Twitch Client ID first',
  'toast.auth_code':'Twitch code received. Open the activation page.','toast.replay_ok':'Replay parsed',
  'toast.started':'Monitoring started','toast.stopping':'Monitoring is stopping','toast.refreshed':'List refreshed',
  'toast.request_error':'Request failed','toast.connect_fail':'Could not connect to the app: {err}',
  'confirm.real':'Enable REAL Twitch Predictions?\n\nThis will create and automatically resolve real predictions with viewer points. Testing the parser in simulation first is recommended.',
  'confirm.quit':'Close HotS Twitch Companion? Monitoring will be stopped.',
  'replay.file':'File','replay.map':'Map','replay.selected':'SELECTED PLAYER',
  'replay.no_selected':'Player not identified: check the name/Toon Handle.',
  'quit.message':'HotS Twitch Companion has exited. You can close this window.'
 },
 ru: {
  'nav.dashboard':'Обзор','nav.replays':'Матчи','nav.automation':'Автоматизация','nav.logs':'Журнал','nav.settings':'Настройки',
  'dash.eyebrow':'ЦЕНТР УПРАВЛЕНИЯ','dash.title':'Обзор','dash.sub':'Состояние HotS, подключения Twitch и автоматических прогнозов.',
  'match.title':'Состояние матча','match.imgalt':'Фэнтезийное поле боя',
  'info.player':'Игрок','info.prediction':'Активный прогноз','info.result':'Последний результат',
  'recent.title':'Последние матчи','recent.all':'Все матчи →','recent.loading':'Загружаем сохранённые реплеи…',
  'log.title':'Журнал событий','log.all':'Весь журнал →',
  'twitch.eyebrow':'ПОДКЛЮЧЕНИЯ','twitch.sub':'Привяжи канал через официальный OAuth. Пароль не передаётся приложению.',
  'twitch.h2':'Подключение аккаунта Twitch','twitch.p':'Нужен Client ID приложения типа Public из Twitch Developer Console. Секретный ключ не требуется.',
  'twitch.placeholder':'Вставь Client ID','twitch.console':'Консоль Twitch ↗','twitch.connect':'Подключить Twitch','twitch.settings.title':'Настройки Twitch',
  'auth.label':'Код для авторизации:','auth.activate':'Открыть Twitch Activate ↗',
  'replays.eyebrow':'ИСТОРИЯ ИГР','replays.title':'Матчи и реплеи','replays.sub':'Проверка результата без запуска нового матча или обращения к Twitch.',
  'replays.refresh':'Обновить','replays.h2':'Последние сохранённые матчи','replays.check':'Проверить последний реплей',
  'automation.eyebrow':'АВТОМАТИЗАЦИЯ','automation.title':'Управление Predictions','automation.sub':'Автоматический запуск по lobby и завершение по результату матча.',
  'automation.start':'Запустить мониторинг','automation.stop':'Остановить','automation.how':'Как это работает',
  'automation.step1.pre':'Появился новый ','automation.step2':'Создаётся прогноз «Победа в HotS?» (приём ставок — {min} мин)',
  'automation.step3':'После появления реплея считывается результат выбранного игрока',
  'automation.step4':'Prediction завершается нужным исходом либо отменяется при тайм-ауте',
  'logs.eyebrow':'ДИАГНОСТИКА','logs.title':'Журнал событий','logs.sub':'Только локальные события и результаты API без секретных токенов.',
  'settings.eyebrow':'КОНФИГУРАЦИЯ','settings.title':'Настройки','settings.sub':'Все настройки сохраняются в профиле Windows. Файлы игры не изменяются.',
  'settings.interface':'Интерфейс','settings.lang':'Язык интерфейса',
  'settings.player':'Ник игрока','settings.player.ph':'Ник из реплея',
  'settings.toon':'Toon Handle (необязательно)','settings.toon.ph':'Точный идентификатор игрока',
  'settings.root':'Папка с реплеями (можно оставить пустой)','settings.root.ph':'Поиск в Documents / Heroes of the Storm',
  'settings.mode':'Режим работы','settings.window':'Длительность Prediction',
  'settings.window.hint':'Время приёма ставок, как в интерфейсе Twitch Predictions (1–30 минут).',
  'settings.real':'Разрешить реальные Twitch Predictions','settings.real.sub':'Только после проверки парсера: реальные баллы зрителей будут распределяться автоматически.',
  'settings.save':'Сохранить настройки',
  'footer.quit':'Выход','footer.quit.title':'Закрыть приложение','lang.switch.title':'Сменить язык','unit.min':'мин',
  'top.connected':'Twitch подключён','top.disconnected':'Нет подключения',
  'dash.channel.authorized':'Авторизован','dash.channel.none':'Не подключён','dash.player.empty':'Не указан',
  'dash.prediction.sim':'Симуляция','dash.prediction.active':'Активен','dash.prediction.none':'Нет','dash.result.none':'Нет данных',
  'match.status.active':'Матч в процессе','match.status.waiting':'Ожидание нового матча','match.status.off':'Мониторинг выключен',
  'match.desc.active':'Prediction открыт. Ожидаем завершения игры и появления реплея.',
  'match.desc.waiting':'Программа следит за новыми lobby и готовыми реплеями.',
  'match.desc.off':'Нажми «Включить автоматику», чтобы начать отслеживание.',
  'match.mini.active':'Матч активен','match.mini.running':'Мониторинг работает','match.mini.waiting':'Ожидание матча',
  'pred.state.label':'СОСТОЯНИЕ PREDICTION','pred.state.active':'Идёт прогноз','pred.state.inactive':'Неактивен',
  'pred.help.active':'Ожидаем результат игры.','pred.help.sim':'Режим симуляции: баллы не задействованы.','pred.help.wait':'Ожидание нового матча.',
  'watch.on':'Мониторинг работает','watch.off':'Мониторинг выключен','start.stop':'Остановить автоматику','start.start':'Включить автоматику',
  'automation.h2.on':'Автоматизация включена','automation.h2.off':'Автоматизация выключена',
  'automation.detail.on':'Ожидание событий из HotS. Отслеживаются только игровые файлы.','automation.detail.off':'Программа не отслеживает новые матчи.',
  'mode.sim':'Режим симуляции','mode.real':'РЕАЛЬНЫЕ PREDICTIONS',
  'mode.sim.detail':'Twitch Predictions не создаются и баллы не распределяются.','mode.real.detail':'Баллы Twitch реально распределяются по итогам матчей.',
  'footer.mode.sim':'Режим симуляции включён','footer.mode.real':'ВНИМАНИЕ: реальные Predictions включены',
  'footer.detail.sim':'Реальные Predictions не создаются. Игровые файлы только читаются.','footer.detail.real':'Приложение создаёт и рассчитывает настоящие ставки на Twitch.',
  'footer.status.on':'Мониторинг включён','footer.status.off':'Ожидание',
  'twitch.info.on':'Аккаунт подключён: @{name}. Разрешение channel:manage:predictions.','twitch.info.off':'Нет подключённого Twitch аккаунта.',
  'auth.info.default':'Введи код на странице Twitch.','auth.info.pending':'Ожидаем подтверждения Twitch…',
  'replays.empty.recent':'Реплеи пока не найдены','replays.empty.full':'Реплеи не найдены. Проверь путь в настройках.',
  'logs.empty':'Событий пока нет','logs.empty.full':'Журнал пуст',
  'badge.win':'ПОБЕДА','badge.loss':'ПОРАЖЕНИЕ','badge.unknown':'НЕИЗВЕСТНО',
  'rel.now':'только что','rel.min':'{n} мин назад','rel.hour':'{n} ч назад',
  'toast.saved':'Настройки сохранены','toast.need_client':'Сначала введи Client ID Twitch',
  'toast.auth_code':'Код Twitch получен. Открой страницу активации.','toast.replay_ok':'Реплей прочитан',
  'toast.started':'Мониторинг запущен','toast.stopping':'Мониторинг останавливается','toast.refreshed':'Список обновлён',
  'toast.request_error':'Ошибка запроса','toast.connect_fail':'Не удалось подключиться к приложению: {err}',
  'confirm.real':'Включить НАСТОЯЩИЕ Twitch Predictions?\n\nЭто создаст и автоматически рассчитает реальные прогнозы с баллами зрителей. Рекомендуется сначала проверить реплей в симуляции.',
  'confirm.quit':'Закрыть HotS Twitch Companion? Мониторинг будет остановлен.',
  'replay.file':'Файл','replay.map':'Карта','replay.selected':'ВЫБРАННЫЙ ИГРОК',
  'replay.no_selected':'Нужный игрок не определён: проверь ник/Toon Handle.',
  'quit.message':'HotS Twitch Companion завершён. Это окно можно закрыть.'
 }
};
let lang = localStorage.getItem('hots-lang') || ((navigator.language || '').toLowerCase().startsWith('ru') ? 'ru' : 'en');
function t(key, vars){
  let s = I18N[lang] && I18N[lang][key] != null ? I18N[lang][key] : I18N.en[key];
  if (s == null) return key;
  if (vars) for (const [k, v] of Object.entries(vars)) s = String(s).split('{'+k+'}').join(v);
  return String(s);
}
function windowMinutes(){ return state.config ? (state.config.predictionWindowMin || 2) : 2; }
function applyI18n(){
  document.documentElement.lang = lang;
  for (const el of document.querySelectorAll('[data-i18n]')) el.textContent = t(el.dataset.i18n, {min: windowMinutes()});
  for (const el of document.querySelectorAll('[data-i18n-placeholder]')) el.placeholder = t(el.dataset.i18nPlaceholder);
  for (const el of document.querySelectorAll('[data-i18n-title]')) el.title = t(el.dataset.i18nTitle);
  for (const el of document.querySelectorAll('[data-i18n-alt]')) el.alt = t(el.dataset.i18nAlt);
  for (const o of $('predictionWindow').querySelectorAll('option')) o.textContent = o.value + ' ' + t('unit.min');
  for (const b of $('lang-switch').querySelectorAll('button')) b.classList.toggle('active', b.dataset.lang === lang);
  $('lang').value = lang;
}
async function setLang(l){
  if (l === lang) return;
  lang = l;
  try { localStorage.setItem('hots-lang', l); } catch (e) { /* private mode */ }
  applyI18n();
  render(state.status);
  showReplays(state.replays);
  // Persist the language so backend logs and errors follow the UI language.
  if (state.config) {
    try {
      const c = state.config;
      const body = {clientId: c.clientId || '', playerName: c.playerName || '', playerToonHandle: c.playerToonHandle || '',
        replayRoot: c.replayRoot || '', realMode: !!c.realMode, confirmedRealMode: !!c.realMode,
        lang: l, predictionWindowMin: c.predictionWindowMin || 0};
      const x = await api('config', 'POST', body);
      state.config = x.config;
      $('predictionWindow').value = String(x.config.predictionWindowMin || 2);
    } catch (e) { toast(e.message, true); }
  }
}

/* ── helpers ─────────────────────────────────────────────────────────── */
function toast(message, error = false){ const n=$('toast'); n.textContent=message; n.className='toast show'+(error?' error':''); clearTimeout(toastTimer); toastTimer=setTimeout(()=>n.className='toast',4600); }
function safe(v){return v==null?'':String(v)}
async function api(route, method='GET', data){
 const options={method,headers:{'X-Companion':'1'},cache:'no-store'};
 if(data!==undefined){options.headers['Content-Type']='application/json';options.body=JSON.stringify(data)}
 const controller=new AbortController(); const timeout=setTimeout(()=>controller.abort(),45000); options.signal=controller.signal;
 try{const r=await fetch(base+'api/'+route,options);const x=await r.json();if(!r.ok||x.error)throw Error(x.error||'HTTP '+r.status);return x}
 finally{clearTimeout(timeout)}
}
function getInputs(){return {clientId:$('clientId').value.trim(),playerName:$('playerName').value.trim(),playerToonHandle:$('toonHandle').value.trim(),replayRoot:$('replayRoot').value.trim(),realMode:$('realMode').checked,lang:$('lang').value,predictionWindowMin:parseInt($('predictionWindow').value,10)||2}}
function setInputs(c){$('clientId').value=c.clientId||'';$('playerName').value=c.playerName||'';$('toonHandle').value=c.playerToonHandle||'';$('replayRoot').value=c.replayRoot||'';$('realMode').checked=!!c.realMode;$('lang').value=c.lang||lang;$('predictionWindow').value=String(c.predictionWindowMin||2);state.hasUnsaved=false;}
function go(view){document.querySelectorAll('.view').forEach(v=>v.classList.toggle('active',v.id==='view-'+view));document.querySelectorAll('.nav-item').forEach(n=>n.classList.toggle('active',n.dataset.view===view));}
function line(text,cls){const d=document.createElement('div');d.className=cls||'';d.textContent=text;return d}
function relative(iso){if(!iso)return '';const ms=Date.now()-new Date(iso).getTime();if(!Number.isFinite(ms))return '';const min=Math.round(ms/60000);if(min<2)return t('rel.now');if(min<60)return t('rel.min',{n:min});if(min<1440)return t('rel.hour',{n:Math.round(min/60)});return new Date(iso).toLocaleDateString(lang==='ru'?'ru-RU':'en-US')}
function resultBadge(result){const span=document.createElement('span');span.className='outcome-pill '+(result==='win'?'win':result==='loss'?'loss':'unknown');span.textContent=result==='win'?t('badge.win'):result==='loss'?t('badge.loss'):t('badge.unknown');return span}
const mapPic=i=>['./assets/map-towers.webp','./assets/map-hollow.webp','./assets/map-dragon.webp'][i%3];
function showReplays(items){state.replays=Array.isArray(items)?items:[];const box=$('recent-list'),full=$('full-replays');box.replaceChildren();full.replaceChildren();if(!state.replays.length){box.append(line(t('replays.empty.recent'),'empty-mini'));full.append(line(t('replays.empty.full'),'empty-mini'));return}
 state.replays.slice(0,3).forEach((r,i)=>{const el=document.createElement('div');el.className='recent-item';const img=document.createElement('img');img.className='recent-img';img.src=mapPic(i);img.alt='';const desc=document.createElement('div');desc.className='recent-text';desc.append(Object.assign(document.createElement('b'),{textContent:r.map||r.file}));desc.append(Object.assign(document.createElement('small'),{textContent:relative(r.modified)}));el.append(img,desc,resultBadge(r.result));box.append(el)});
 state.replays.forEach(r=>{const el=document.createElement('div');el.className='full-replay';const text=document.createElement('div');text.append(Object.assign(document.createElement('div'),{className:'full-replay-name',textContent:r.map||r.file}));text.append(Object.assign(document.createElement('small'),{textContent:(r.file||'')+' · '+relative(r.modified)}));el.append(text,resultBadge(r.result));full.append(el)});
}
function showLogs(logs){const arr=Array.isArray(logs)?logs:[];for(const id of ['log-preview','full-log']){const el=$(id);el.replaceChildren();const filtered=id==='log-preview'?arr.slice(-7):arr;if(!filtered.length){el.append(line(id==='log-preview'?t('logs.empty'):t('logs.empty.full'),'empty-mini'));continue}for(const entry of filtered){const row=document.createElement('div');row.className='log-row';const time=Object.assign(document.createElement('span'),{className:'log-time',textContent:(entry.time||'').slice(0,8)});const ico=Object.assign(document.createElement('span'),{className:'log-icon',textContent:entry.message&&/(Ошибка|Не удалось|error|fail|expired|invalid)/i.test(entry.message)?'!':'◆'});const msg=Object.assign(document.createElement('span'),{className:'log-msg',textContent:entry.message||''});row.append(time,ico,msg);el.append(row)}}}
function render(s){
 if(s)state.status=s;
 const st=state.status||{connected:false,running:false,active:false,realMode:false,logs:[]};
 const connected=!!st.connected;const running=!!st.running;const active=!!st.active;const name=st.channel||'';const simulation=!st.realMode;
 $('top-connection').classList.toggle('connected',connected);$('top-connected-text').textContent=connected?t('top.connected'):t('top.disconnected');
 $('dash-channel').textContent=connected?(name?'@'+name:t('dash.channel.authorized')):t('dash.channel.none');$('channel-pill').textContent=connected?'CONNECTED':'OFFLINE';$('channel-pill').classList.toggle('online',connected);
 $('dash-player').textContent=st.playerName||st.playerToonHandle||t('dash.player.empty');
 $('dash-prediction').textContent=active?(st.activeDryRun?t('dash.prediction.sim'):t('dash.prediction.active')):t('dash.prediction.none');
 $('dash-result').textContent=st.lastResult||t('dash.result.none');
 $('match-status').textContent=active?t('match.status.active'):running?t('match.status.waiting'):t('match.status.off');
 $('match-description').textContent=active?t('match.desc.active'):running?t('match.desc.waiting'):t('match.desc.off');
 $('match-mini').textContent=active?t('match.mini.active'):running?t('match.mini.running'):t('match.mini.waiting');
 $('prediction-state').textContent=active?t('pred.state.active'):t('pred.state.inactive');
 $('prediction-help').textContent=active?t('pred.help.active'):simulation?t('pred.help.sim'):t('pred.help.wait');
 $('watch-tag').classList.toggle('on',running);$('watch-tag').innerHTML='<span class="pulse-dot"></span> '+t(running?'watch.on':'watch.off');
 $('start-label').textContent=running?t('start.stop'):t('start.start');
 $('automation-title').textContent=running?t('automation.h2.on'):t('automation.h2.off');
 $('automation-detail').textContent=running?t('automation.detail.on'):t('automation.detail.off');
 $('automation-mode').textContent=simulation?t('mode.sim'):t('mode.real');
 $('automation-mode-detail').textContent=simulation?t('mode.sim.detail'):t('mode.real.detail');
 $('footer-mode').textContent=simulation?t('footer.mode.sim'):t('footer.mode.real');
 $('footer-detail').textContent=simulation?t('footer.detail.sim'):t('footer.detail.real');
 $('footer-status').textContent=running?t('footer.status.on'):t('footer.status.off');
 $('twitch-info').textContent=connected?t('twitch.info.on',{name}):t('twitch.info.off');
 if(st.authCode){$('auth-progress').hidden=false;$('auth-code').textContent=st.authCode;$('auth-info').textContent=st.authStatus||t('auth.info.default')}else if(st.authPending){$('auth-progress').hidden=false;$('auth-info').textContent=t('auth.info.pending');}else{$('auth-progress').hidden=true}
 showLogs(st.logs);
}
async function loadStatus(){try{render(await api('status'))}catch(e){/* app closed, keep last UI */}}
async function reloadReplays(){try{const x=await api('replays');showReplays(x.replays||[])}catch(e){toast(e.message,true)}}
async function save(){const data=getInputs();if(data.realMode&&!state.config?.realMode){if(!confirm(t('confirm.real'))){data.realMode=false;$('realMode').checked=false}}
 const x=await api('config','POST',{...data,confirmedRealMode:data.realMode});state.config=x.config;state.hasUnsaved=false;lang=x.config.lang||lang;applyI18n();toast(t('toast.saved'));await loadStatus();return x}
async function action(name){if(state.busy&&name!=='quit')return;state.busy=true;try{switch(name){
 case 'save':await save();break;
 case 'auth':{if(state.hasUnsaved)await save();if(!$('clientId').value.trim()){go('twitch');toast(t('toast.need_client'),true);break}const x=await api('auth','POST',{});if(x.code){go('twitch');$('auth-progress').hidden=false;$('auth-code').textContent=x.code;code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';toast(t('toast.auth_code'))}break}
 case 'activate':window.open(code.authUrl,'_blank','noopener,noreferrer');break;
 case 'developer':window.open('https://dev.twitch.tv/console/apps','_blank','noopener,noreferrer');break;
 case 'replay':{const x=await api('replay/check','POST',{});go('replays');const el=$('replay-detail');el.hidden=false;el.textContent=t('replay.file')+': '+x.file+'\n'+t('replay.map')+': '+x.map+'\n\n'+(x.players||[]).map(p=>p.name+' · '+p.toonHandle+' · '+p.result).join('\n')+'\n\n'+(x.selected?t('replay.selected')+': '+x.selected.name+' → '+x.selected.result:t('replay.no_selected'));toast(t('toast.replay_ok'));await reloadReplays();break}
 case 'start':if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast(t('toast.started'));await loadStatus();break;
 case 'stop':await api('automation/stop','POST',{});toast(t('toast.stopping'));await loadStatus();break;
 case 'toggle':if(state.status?.running){await api('automation/stop','POST',{});toast(t('toast.stopping'))}else{if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast(t('toast.started'))}await loadStatus();break;
 case 'refresh-replays':await reloadReplays();toast(t('toast.refreshed'));break;
 case 'quit':if(!confirm(t('confirm.quit')))break;await api('quit','POST',{});window.close();document.body.innerHTML='<div style="font:18px Segoe UI;color:#ccdaee;margin:70px;text-align:center">'+safe(t('quit.message'))+'</div>';break;
 }}catch(e){toast(e.message||t('toast.request_error'),true)}finally{state.busy=false}}
window.addEventListener('DOMContentLoaded',async()=>{
 document.querySelectorAll('[data-view]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.view)));
 document.querySelectorAll('[data-goto]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.goto)));
 document.querySelectorAll('[data-action]').forEach(el=>el.addEventListener('click',()=>action(el.dataset.action)));
 for (const b of $('lang-switch').querySelectorAll('button')) b.addEventListener('click',()=>setLang(b.dataset.lang));
 for (const id of ['clientId','playerName','toonHandle','replayRoot','realMode','lang','predictionWindow']){$(id).addEventListener('input',()=>state.hasUnsaved=true);$(id).addEventListener('change',()=>state.hasUnsaved=true);}
 applyI18n();
 render(null);
 try{const b=await api('bootstrap');state.config=b.config;setInputs(b.config);if(b.config.lang){lang=b.config.lang;try{localStorage.setItem('hots-lang',lang)}catch(e){}applyI18n();}render(b.status);showReplays(b.replays||[])}catch(e){toast(t('toast.connect_fail',{err:e.message}),true)}
 setInterval(loadStatus,2500);setInterval(reloadReplays,30000);
});
