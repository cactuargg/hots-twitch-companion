'use strict';
const base = new URL('./', location.href).pathname;
const $ = id => document.getElementById(id);
const state = { status: null, config: null, replays: [], hasUnsaved: false, busy: false };
const code = { authUrl: 'https://www.twitch.tv/activate' };
let toastTimer;

/* ── i18n ────────────────────────────────────────────────────────────── */
const I18N = {
 en: {
  'nav.dashboard':'Overview','nav.history':'Match History','nav.settings':'Settings','nav.logs':'Logs',
  'dash.eyebrow':'CONTROL CENTER','dash.title':'Overview','dash.sub':'HotS, Twitch connection and automated prediction status.',
  'match.title':'Match status','match.imgalt':'Fantasy battleground',
  'info.player':'Player','info.prediction':'Active prediction','info.result':'Last result',
  'recent.title':'Recent matches','recent.all':'Full history →','recent.loading':'Loading saved replays…',
  'log.title':'Event log','log.all':'Full log →',
  'twitch.placeholder':'Paste your Client ID','twitch.console':'Twitch Console ↗','twitch.connect':'Connect Twitch','twitch.settings.title':'Settings',
  'auth.label':'Authorization code:','auth.activate':'Open Twitch Activate ↗',
  'replays.eyebrow':'MATCH HISTORY','replays.title':'Match History','replays.sub':'Click a match to see its details and players.',
  'replays.refresh':'Refresh','replays.h2':'Saved matches','replays.check':'Check latest replay',
  'match.players':'Players','match.you':'YOU','match.error':'Could not parse this replay',
  'automation.start':'Start monitoring','automation.stop':'Stop','automation.how':'How it works',
  'automation.step1.pre':'A new ','automation.step2':'A “HotS win?” prediction opens (betting window {min} min)',
  'automation.step3':'Once the replay appears, the selected player’s result is read',
  'automation.step4':'The prediction resolves with the outcome or is cancelled on timeout',
  'logs.eyebrow':'DIAGNOSTICS','logs.title':'Event log','logs.sub':'Local events and API results only — no secret tokens.',
  'settings.eyebrow':'CONFIGURATION','settings.title':'Settings','settings.sub':'Settings are stored in your Windows profile. Game files are never modified.',
  'settings.interface':'Interface','settings.lang':'Interface language','settings.twitch':'Twitch connection',
  'settings.player':'Player name','settings.player.ph':'Name as it appears in replays',
  'settings.toon':'Toon Handle (optional)','settings.toon.ph':'Exact player identifier',
  'settings.root':'Replays folder (can stay empty)','settings.root.ph':'Searches Documents / Heroes of the Storm',
  'settings.mode':'Operation mode','settings.window':'Prediction duration',
  'settings.window.hint':'Betting window, same as the Twitch Predictions interface (1–30 minutes).',
  'settings.real':'Allow real Twitch Predictions','settings.real.sub':'Only after verifying the parser: real viewer points will be distributed automatically.',
  'settings.save':'Save settings','settings.wizard':'Setup wizard',
  'footer.quit':'Exit','footer.quit.title':'Close the app','unit.min':'min',
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
  'quit.message':'HotS Twitch Companion has exited. You can close this window.',
  'wiz.eyebrow':'INITIAL SETUP','wiz.title':'Setup wizard',
  'wiz.intro':'This wizard connects the app to your game and channel in a few steps: player name, Twitch and automation. Everything can be changed later in Settings.',
  'wiz.lang.h':'Choose language','wiz.lang.sub':'The interface and app messages switch immediately.',
  'wiz.p.h':'Your HotS identity','wiz.p.sub':'Enter the exact nickname you play with — it must match the name stored in replays. If the nickname is not unique, set a Toon Handle later in Settings.',
  'wiz.p.btn':'Check last replay','wiz.p.ok':'Recognized player: {res} (map: {map})','wiz.p.fail':'{err}',
  'wiz.p.hint':'Expected result: your nickname and “Win” or “Loss”. If there are no replays yet, play one game and retry.',
  'wiz.t.h':'Connect Twitch','wiz.t.sub':'In the Twitch Developer Console create a “Public” application, copy the Client ID (no secret needed) and paste it here.',
  'wiz.t.console':'Open Twitch Console ↗','wiz.t.pending':'Waiting for Twitch confirmation…',
  'wiz.t.hint':'Expected: a code appears, you approve access on the Twitch page, and the status becomes “Twitch connected”.',
  'wiz.a.h':'Automation','wiz.a.sub':'During a game the app watches HotS files, creates a “HotS win?” prediction and resolves it by the match result. Simulation mode is on by default — real viewer points are untouched.',
  'wiz.a.duration':'The betting window is configured in Settings (1–30 minutes).',
  'wiz.a.btn':'Start monitoring (simulation)',
  'wiz.f.h':'Ready!','wiz.f.player':'Player','wiz.f.twitch':'Twitch','wiz.f.monitor':'Monitoring',
  'wiz.f.sub':'Everything can be changed later in Settings. Real Predictions are enabled there too — only after verifying the parser on your own replays.',
  'wiz.finish':'Finish','wiz.next':'Next','wiz.back':'Back','wiz.skip':'Skip setup',
  'wiz.ok':'✓ Done','wiz.pending':'— Not yet','wiz.fail':'✗ Check failed'
 },
 ru: {
  'nav.dashboard':'Обзор','nav.history':'История матчей','nav.settings':'Настройки','nav.logs':'Журнал',
  'dash.eyebrow':'ЦЕНТР УПРАВЛЕНИЯ','dash.title':'Обзор','dash.sub':'Состояние HotS, подключения Twitch и автоматических прогнозов.',
  'match.title':'Состояние матча','match.imgalt':'Фэнтезийное поле боя',
  'info.player':'Игрок','info.prediction':'Активный прогноз','info.result':'Последний результат',
  'recent.title':'Последние матчи','recent.all':'Вся история →','recent.loading':'Загружаем сохранённые реплеи…',
  'log.title':'Журнал событий','log.all':'Весь журнал →',
  'twitch.placeholder':'Вставь Client ID','twitch.console':'Консоль Twitch ↗','twitch.connect':'Подключить Twitch','twitch.settings.title':'Настройки',
  'auth.label':'Код для авторизации:','auth.activate':'Открыть Twitch Activate ↗',
  'replays.eyebrow':'ИСТОРИЯ МАТЧЕЙ','replays.title':'История матчей','replays.sub':'Нажми на матч, чтобы увидеть данные игры и игроков.',
  'replays.refresh':'Обновить','replays.h2':'Сохранённые матчи','replays.check':'Проверить последний реплей',
  'match.players':'Игроки','match.you':'ВЫ','match.error':'Не удалось разобрать этот реплей',
  'automation.start':'Запустить мониторинг','automation.stop':'Остановить','automation.how':'Как это работает',
  'automation.step1.pre':'Появился новый ','automation.step2':'Создаётся прогноз «Победа в HotS?» (приём ставок — {min} мин)',
  'automation.step3':'После появления реплея считывается результат выбранного игрока',
  'automation.step4':'Prediction завершается нужным исходом либо отменяется при тайм-ауте',
  'logs.eyebrow':'ДИАГНОСТИКА','logs.title':'Журнал событий','logs.sub':'Только локальные события и результаты API без секретных токенов.',
  'settings.eyebrow':'КОНФИГУРАЦИЯ','settings.title':'Настройки','settings.sub':'Все настройки сохраняются в профиле Windows. Файлы игры не изменяются.',
  'settings.interface':'Интерфейс','settings.lang':'Язык интерфейса','settings.twitch':'Подключение Twitch',
  'settings.player':'Ник игрока','settings.player.ph':'Ник из реплея',
  'settings.toon':'Toon Handle (необязательно)','settings.toon.ph':'Точный идентификатор игрока',
  'settings.root':'Папка с реплеями (можно оставить пустой)','settings.root.ph':'Поиск в Documents / Heroes of the Storm',
  'settings.mode':'Режим работы','settings.window':'Длительность Prediction',
  'settings.window.hint':'Время приёма ставок, как в интерфейсе Twitch Predictions (1–30 минут).',
  'settings.real':'Разрешить реальные Twitch Predictions','settings.real.sub':'Только после проверки парсера: реальные баллы зрителей будут распределяться автоматически.',
  'settings.save':'Сохранить настройки','settings.wizard':'Мастер настройки',
  'footer.quit':'Выход','footer.quit.title':'Закрыть приложение','unit.min':'мин',
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
  'quit.message':'HotS Twitch Companion завершён. Это окно можно закрыть.',
  'wiz.eyebrow':'ПЕРВОНАЧАЛЬНАЯ НАСТРОЙКА','wiz.title':'Мастер настройки',
  'wiz.intro':'Мастер за несколько шагов подключит приложение к игре и твоему каналу: ник игрока, Twitch и автоматизация. Всё это позже можно изменить в «Настройках».',
  'wiz.lang.h':'Выбери язык','wiz.lang.sub':'Интерфейс и сообщения приложения переключаются сразу.',
  'wiz.p.h':'Твой игрок в HotS','wiz.p.sub':'Введи точный ник, под которым играешь — он должен совпадать с именем в реплеях. Если ник не уникален, позже укажи Toon Handle в настройках.',
  'wiz.p.btn':'Проверить последний реплей','wiz.p.ok':'Игрок распознан: {res} (карта: {map})','wiz.p.fail':'{err}',
  'wiz.p.hint':'Ожидаемый результат: твой ник и «Победа» или «Поражение». Если реплеев нет — сыграй одну игру и повтори.',
  'wiz.t.h':'Подключение Twitch','wiz.t.sub':'В Twitch Developer Console создай приложение типа «Public», скопируй Client ID (секрет не нужен) и вставь сюда.',
  'wiz.t.console':'Открыть консоль Twitch ↗','wiz.t.pending':'Ожидаем подтверждения Twitch…',
  'wiz.t.hint':'Ожидаемый результат: появится код, ты подтвердишь доступ на странице Twitch, и статус станет «Twitch подключён».',
  'wiz.a.h':'Автоматизация','wiz.a.sub':'Во время игры приложение следит за файлами HotS, создаёт Prediction «Победа в HotS?» и завершает его по результату матча. По умолчанию включён режим симуляции — реальные баллы зрителей не затрагиваются.',
  'wiz.a.duration':'Длительность приёма ставок настраивается в «Настройках» (1–30 минут).',
  'wiz.a.btn':'Запустить мониторинг (симуляция)',
  'wiz.f.h':'Готово!','wiz.f.player':'Игрок','wiz.f.twitch':'Twitch','wiz.f.monitor':'Мониторинг',
  'wiz.f.sub':'Всё можно изменить позже в «Настройках». Реальные Predictions включаются там же и только после проверки парсера на своих реплеях.',
  'wiz.finish':'Готово','wiz.next':'Далее','wiz.back':'Назад','wiz.skip':'Пропустить настройку',
  'wiz.ok':'✓ Готово','wiz.pending':'— Ещё нет','wiz.fail':'✗ Не удалось проверить'
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
  for (const b of document.querySelectorAll('[data-wiz-lang]')) b.classList.toggle('active', b.dataset.wizLang === lang);
  $('lang').value = lang;
}
async function setLang(l){
  if (l === lang) return;
  lang = l;
  try { localStorage.setItem('hots-lang', l); } catch (e) { /* private mode */ }
  applyI18n();
  render(state.status);
  showReplays(state.replays);
  if (state.config) {
    try {
      const x = await save({lang: l});
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
function getInputs(){return {clientId:$('clientId').value.trim(),playerName:$('playerName').value.trim(),playerToonHandle:$('toonHandle').value.trim(),replayRoot:$('replayRoot').value.trim(),realMode:$('realMode').checked,lang:$('lang').value,predictionWindowMin:parseInt($('predictionWindow').value,10)||2,wizardDone:!!(state.config&&state.config.wizardDone)}}
function setInputs(c){$('clientId').value=c.clientId||'';$('playerName').value=c.playerName||'';$('toonHandle').value=c.playerToonHandle||'';$('replayRoot').value=c.replayRoot||'';$('realMode').checked=!!c.realMode;$('lang').value=c.lang||lang;$('predictionWindow').value=String(c.predictionWindowMin||2);state.hasUnsaved=false;}
function go(view){document.querySelectorAll('.view').forEach(v=>v.classList.toggle('active',v.id==='view-'+view));document.querySelectorAll('.tab').forEach(n=>n.classList.toggle('active',n.dataset.view===view));}
function line(text,cls){const d=document.createElement('div');d.className=cls||'';d.textContent=text;return d}
function relative(iso){if(!iso)return '';const ms=Date.now()-new Date(iso).getTime();if(!Number.isFinite(ms))return '';const min=Math.round(ms/60000);if(min<2)return t('rel.now');if(min<60)return t('rel.min',{n:min});if(min<1440)return t('rel.hour',{n:Math.round(min/60)});return new Date(iso).toLocaleDateString(lang==='ru'?'ru-RU':'en-US')}
function resultBadge(result){const span=document.createElement('span');span.className='outcome-pill '+(result==='win'?'win':result==='loss'?'loss':'unknown');span.textContent=result==='win'?t('badge.win'):result==='loss'?t('badge.loss'):t('badge.unknown');return span}
const mapPic=i=>['./assets/map-towers.webp','./assets/map-hollow.webp','./assets/map-dragon.webp'][i%3];

function playerRow(p, selected){
  const row=document.createElement('div');row.className='player-row'+(selected?' you':'');
  const name=document.createElement('span');name.className='player-name';name.textContent=p.name||'—';
  if(selected){const tag=Object.assign(document.createElement('em'),{textContent:t('match.you')});name.append(tag);}
  const toon=document.createElement('small');toon.textContent=p.toonHandle||'';
  row.append(name,toon,resultBadge(p.result));
  return row;
}
function matchDetail(r){
  const box=document.createElement('div');box.className='match-detail';
  if(r.error){box.append(line(t('match.error')+': '+r.error,'empty-mini'));return box;}
  const head=Object.assign(document.createElement('div'),{className:'match-detail-head'});
  head.append(Object.assign(document.createElement('b'),{textContent:r.map||r.file}));
  head.append(Object.assign(document.createElement('small'),{textContent:(r.file||'')+' · '+(r.modified?new Date(r.modified).toLocaleString(lang==='ru'?'ru-RU':'en-US'):'')}));
  box.append(head);
  const list=document.createElement('div');list.className='player-list';
  (r.players&&r.players.length?r.players:[]).forEach(p=>list.append(playerRow(p,!!r.selected&&r.selected===(p.name||''))));
  if(!list.childNodes.length)list.append(line(t('match.error'),'empty-mini'));
  box.append(list);
  return box;
}
function showReplays(items){
  state.replays=Array.isArray(items)?items:[];
  const box=$('recent-list'),full=$('full-replays');box.replaceChildren();full.replaceChildren();
  if(!state.replays.length){box.append(line(t('replays.empty.recent'),'empty-mini'));full.append(line(t('replays.empty.full'),'empty-mini'));return}
  state.replays.slice(0,3).forEach((r,i)=>{
    const el=document.createElement('div');el.className='recent-item';
    const img=document.createElement('img');img.className='recent-img';img.src=mapPic(i);img.alt='';
    const desc=document.createElement('div');desc.className='recent-text';
    desc.append(Object.assign(document.createElement('b'),{textContent:r.map||r.file}));
    desc.append(Object.assign(document.createElement('small'),{textContent:relative(r.modified)}));
    el.append(img,desc,resultBadge(r.result));box.append(el);
  });
  state.replays.forEach(r=>{
    const item=document.createElement('div');item.className='match-item';
    const head=document.createElement('button');head.type='button';head.className='match-head';
    const title=document.createElement('span');title.className='match-title';title.textContent=r.map||r.file;
    const sub=document.createElement('small');sub.textContent=(r.file||'')+' · '+relative(r.modified);
    head.append(title,sub,resultBadge(r.result),Object.assign(document.createElement('span'),{className:'chev'}));
    head.querySelector('.chev').innerHTML='<svg><use href="#i-chev"/></svg>';
    head.addEventListener('click',()=>{const open=item.classList.toggle('open');item.querySelector('.match-detail').hidden=!open;});
    item.append(head,matchDetail(r));
    full.append(item);
  });
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
 $('automation-mode').textContent=simulation?t('mode.sim'):t('mode.real');
 $('automation-mode-detail').textContent=simulation?t('mode.sim.detail'):t('mode.real.detail');
 $('footer-mode').textContent=simulation?t('footer.mode.sim'):t('footer.mode.real');
 $('footer-detail').textContent=simulation?t('footer.detail.sim'):t('footer.detail.real');
 $('footer-status').textContent=running?t('footer.status.on'):t('footer.status.off');
 $('twitch-info').textContent=connected?t('twitch.info.on',{name}):t('twitch.info.off');
 if(st.authCode){$('auth-progress').hidden=false;$('auth-code').textContent=st.authCode;$('auth-info').textContent=st.authStatus||t('auth.info.default')}else if(st.authPending){$('auth-progress').hidden=false;$('auth-info').textContent=t('auth.info.pending');}else{$('auth-progress').hidden=true}
 showLogs(st.logs);
 if(!$('wizard').hidden)renderWizard();
}
async function loadStatus(){try{render(await api('status'))}catch(e){/* app closed, keep last UI */}}
async function reloadReplays(){try{const x=await api('replays');showReplays(x.replays||[])}catch(e){toast(e.message,true)}}
async function save(over={}){
 const data={...getInputs(),...over};
 if(data.realMode&&!state.config?.realMode){if(!confirm(t('confirm.real'))){data.realMode=false;$('realMode').checked=false}}
 const x=await api('config','POST',{...data,confirmedRealMode:data.realMode});
 state.config=x.config;state.hasUnsaved=false;lang=x.config.lang||lang;applyI18n();toast(t('toast.saved'));await loadStatus();return x;
}
async function action(name){if(state.busy&&name!=='quit')return;state.busy=true;try{switch(name){
 case 'save':await save();break;
 case 'auth':{if(state.hasUnsaved)await save();if(!$('clientId').value.trim()){go('settings');toast(t('toast.need_client'),true);break}const x=await api('auth','POST',{});if(x.code){go('settings');$('auth-progress').hidden=false;$('auth-code').textContent=x.code;code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';toast(t('toast.auth_code'))}break}
 case 'activate':window.open(code.authUrl,'_blank','noopener,noreferrer');break;
 case 'developer':window.open('https://dev.twitch.tv/console/apps','_blank','noopener,noreferrer');break;
 case 'replay':{await api('replay/check','POST',{});toast(t('toast.replay_ok'));await reloadReplays();const first=document.querySelector('#full-replays .match-item');if(first){first.classList.add('open');const d=first.querySelector('.match-detail');if(d)d.hidden=false;}break}
 case 'start':if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast(t('toast.started'));await loadStatus();break;
 case 'stop':await api('automation/stop','POST',{});toast(t('toast.stopping'));await loadStatus();break;
 case 'toggle':if(state.status?.running){await api('automation/stop','POST',{});toast(t('toast.stopping'))}else{if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast(t('toast.started'))}await loadStatus();break;
 case 'refresh-replays':await reloadReplays();toast(t('toast.refreshed'));break;
 case 'wizard':openWizard();break;
 case 'quit':if(!confirm(t('confirm.quit')))break;await api('quit','POST',{});window.close();document.body.innerHTML='<div style="font:18px Segoe UI;color:#ccdaee;margin:70px;text-align:center">'+safe(t('quit.message'))+'</div>';break;
 }}catch(e){toast(e.message||t('toast.request_error'),true)}finally{state.busy=false}}

/* ── setup wizard ────────────────────────────────────────────────────── */
const wiz={step:0,max:4};
function openWizard(){
  wiz.step=0;
  $('wz-player').value=$('playerName').value;
  $('wz-client').value=$('clientId').value;
  $('wz-test-result').hidden=true;$('wz-auth-box').hidden=true;
  renderWizard();
  $('wizard').hidden=false;
}
function closeWizard(){$('wizard').hidden=true;}
function renderWizard(){
  const steps=document.querySelectorAll('.wizard-body section');
  steps.forEach(sec=>sec.hidden=Number(sec.dataset.step)!==wiz.step);
  const dots=$('wiz-dots');dots.replaceChildren();
  for(let i=0;i<=wiz.max;i++){const d=document.createElement('span');d.className='dot'+(i===wiz.step?' on':'')+(i<wiz.step?' done':'');dots.append(d);}
  document.querySelector('[data-wiz="back"]').disabled=wiz.step===0;
  $('wiz-next').textContent=wiz.step===wiz.max?t('wiz.finish'):t('wiz.next');
  const st=state.status||{};
  const set=(id,ok,text)=>{
    const li=$(id);
    li.classList.toggle('ok',!!ok);
    li.querySelector('b').textContent=text!=null&&text!==''?text:(ok?t('wiz.ok'):t('wiz.pending'));
  };
  set('wz-ch-player',!!(state.config&&state.config.playerName),(state.config&&state.config.playerName)||null);
  set('wz-ch-twitch',!!st.connected);
  set('wz-ch-monitor',!!st.running);
}
async function wizNext(){
  if(wiz.step===1){$('playerName').value=$('wz-player').value.trim();try{await save();}catch(e){toast(e.message,true);return false;}}
  if(wiz.step===2){$('clientId').value=$('wz-client').value.trim();try{await save();}catch(e){toast(e.message,true);return false;}}
  if(wiz.step===wiz.max){try{await save({wizardDone:true});}catch(e){toast(e.message,true);}closeWizard();return true;}
  wiz.step++;renderWizard();return true;
}
async function wizardAction(kind){
  try{
    if(kind==='test'){
      const res=$('wz-test-result');res.hidden=false;res.textContent='…';
      $('playerName').value=$('wz-player').value.trim();await save();
      const x=await api('replay/check','POST',{});
      res.className='wiz-result ok';
      res.textContent=x.selected?t('wiz.p.ok',{res:x.selected.name+' → '+x.selected.result,map:x.map}):t('replay.no_selected');
      await reloadReplays();
    }else if(kind==='console'){window.open('https://dev.twitch.tv/console/apps','_blank','noopener,noreferrer');}
    else if(kind==='auth'){
      $('clientId').value=$('wz-client').value.trim();await save();
      if(!$('clientId').value.trim()){toast(t('toast.need_client'),true);return;}
      const x=await api('auth','POST',{});
      if(x.code){const box=$('wz-auth-box');box.hidden=false;$('wz-auth-code').textContent=x.code;code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';toast(t('toast.auth_code'));}
    }else if(kind==='activate'){window.open(code.authUrl,'_blank','noopener,noreferrer');}
    else if(kind==='start'){
      $('playerName').value=$('wz-player').value.trim();await save();
      await api('automation/start','POST',{});toast(t('toast.started'));await loadStatus();
    }
  }catch(e){
    if(kind==='test'){const res=$('wz-test-result');res.hidden=false;res.className='wiz-result err';res.textContent=t('wiz.p.fail',{err:e.message});}
    else toast(e.message||t('toast.request_error'),true);
  }
}
window.addEventListener('DOMContentLoaded',async()=>{
 document.querySelectorAll('[data-view]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.view)));
 document.querySelectorAll('[data-goto]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.goto)));
 document.querySelectorAll('[data-action]').forEach(el=>el.addEventListener('click',()=>action(el.dataset.action)));
 document.querySelectorAll('[data-wiz]').forEach(el=>el.addEventListener('click',()=>wizardAction(el.dataset.wiz)));
 document.querySelectorAll('[data-wiz-lang]').forEach(el=>el.addEventListener('click',()=>setLang(el.dataset.wizLang)));
 document.querySelector('[data-wiz="next"]').addEventListener('click',wizNext);
 document.querySelector('[data-wiz="back"]').addEventListener('click',()=>{if(wiz.step>0){wiz.step--;renderWizard();}});
 document.querySelector('[data-wiz="skip"]').addEventListener('click',async()=>{try{await save({wizardDone:true});}catch(e){/* keep wizard re-runnable */}closeWizard();});
 for (const id of ['clientId','playerName','toonHandle','replayRoot','realMode','predictionWindow']){$(id).addEventListener('input',()=>state.hasUnsaved=true);$(id).addEventListener('change',()=>state.hasUnsaved=true);}
 $('lang').addEventListener('change',()=>setLang($('lang').value));
 applyI18n();
 render(null);
 try{
   const b=await api('bootstrap');state.config=b.config;setInputs(b.config);
   if(b.config.lang){lang=b.config.lang;try{localStorage.setItem('hots-lang',lang)}catch(e){}applyI18n();}
   render(b.status);showReplays(b.replays||[]);
   if(!b.config.wizardDone)openWizard();
 }catch(e){toast(t('toast.connect_fail',{err:e.message}),true)}
 setInterval(loadStatus,2500);setInterval(reloadReplays,30000);
});
