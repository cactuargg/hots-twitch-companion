'use strict';
const obsText={
 en:{customCSS:'Custom CSS',customCSSHelp:'Selectors: #score, #wins, #losses, .divider. Use !important for number colours. Clear to restore the default. Save with Save changes.',basic:'General',predictions:'Predictions',overlays:'OBS overlays',application:'Application',score:'Session score',enabled:'Count matches',ranked:'Count only Storm League',style:'Appearance',nexus:'Nexus',minimal:'Minimal',scale:'Scale (%)',backdrop:'Background',labels:'W / L labels',winColor:'Win colour',lossColor:'Loss colour',mask:'Hide draft picks',maskEnabled:'Enable mask',maskRanked:'Only Storm League',auto:'Automatic (BattleLobby)',show:'Show mask',hide:'Hide mask',maskHelp:'Auto hides the mask when a fresh BattleLobby appears (loading / match), then shows it after the matching replay is written. It does not detect the draft screen directly. Until the mode is known, the mask remains visible. Use manual control for cancelled drafts or a delayed replay.',image:'PNG / JPEG, up to 2 MiB',clearImage:'Use default panel',apply:'Apply overlays',saved:'Overlay settings saved',copy:'Copy OBS URL',copied:'Copied',obsHelp:'OBS → Browser source → URL. Score: 560 × 140; mask: size of the area you cover. Keep Companion running. Overlay settings are applied separately.',session:'Current session',newSession:'New session',undo:'Undo last reset',since:'Since {time}',off:'Counter is off',history:'Sessions and matches',empty:'No sessions yet',noMatches:'No matches in this session',pending:'Needs a session',unassigned:'Matches without a confirmed start are not counted. Select a session below.',manual:'Edited manually',excluded:'Excluded',include:'Include',exclude:'Exclude',boundary:'Start session here',assignment:'Move to session…',win:'Win',loss:'Loss',unknown:'Unknown',warning:'A new session keeps the current match in the previous session. Previous results remain in History.',phase_unknown:'Game state unknown',phase_loading_or_game:'Loading / match',phase_between_games:'Between matches',phase_excluded_mode:'Excluded mode',immediate:'Quick controls apply immediately. Appearance changes use “Apply overlays”.',settingsHint:'Choose a section. Main settings use Save; overlays have their own Apply button.',imageError:'Choose a PNG or JPEG under 2 MiB',resetConfirm:'Reset also deletes session history and OBS links.',noURL:'OBS server unavailable',preview:'Preview',all:'All supported modes',filter:'Match filter',newPlayer:'Choose and save your HotS player first',source:'Mode {mode} · {hero}',notRunning:'Waiting for the local OBS server'},
 ru:{customCSS:'Свой CSS',customCSSHelp:'Селекторы: #score, #wins, #losses, .divider. Для цвета цифр используй !important. Очисти поле для стандартного стиля. Сохраняется кнопкой «Сохранить изменения».',basic:'Основные',predictions:'Прогнозы',overlays:'Оверлеи OBS',application:'Приложение',score:'Счёт сессии',enabled:'Считать матчи',ranked:'Считать только Storm League',style:'Оформление',nexus:'Нексус',minimal:'Минимальное',scale:'Масштаб (%)',backdrop:'Подложка',labels:'Подписи W / L',winColor:'Цвет побед',lossColor:'Цвет поражений',mask:'Скрытие препиков',maskEnabled:'Включить маску',maskRanked:'Только Storm League',auto:'Автоматически (BattleLobby)',show:'Показывать маску',hide:'Скрыть маску',maskHelp:'Автоматика скрывает маску при появлении свежего BattleLobby (загрузка / матч), возвращает после записи соответствующего реплея. Сам экран драфта не распознаётся. Пока режим неизвестен, маска видна. При отмене драфта или задержке реплея используй ручное управление.',image:'PNG / JPEG, до 2 МиБ',clearImage:'Стандартная панель',apply:'Применить оверлеи',saved:'Настройки оверлеев сохранены',copy:'Копировать URL для OBS',copied:'Скопировано',obsHelp:'OBS → Источник «Браузер» → URL. Счёт: 560 × 140; маска: размер закрываемой области. Companion должен работать. Настройки оверлеев применяются отдельно.',session:'Текущая сессия',newSession:'Новая сессия',undo:'Отменить последний сброс',since:'С {time}',off:'Счётчик выключен',history:'Сессии и матчи',empty:'Сессий пока нет',noMatches:'В этой сессии пока нет матчей',pending:'Нужно выбрать сессию',unassigned:'Матчи без подтверждённого начала не входят в счёт. Выбери для них сессию ниже.',manual:'Исправлено вручную',excluded:'Исключён',include:'Учитывать',exclude:'Исключить',boundary:'Начать сессию отсюда',assignment:'Перенести в сессию…',win:'Победа',loss:'Поражение',unknown:'Не определён',warning:'Текущий матч останется в предыдущей сессии. Её результаты сохранятся в истории.',phase_unknown:'Состояние игры неизвестно',phase_loading_or_game:'Загрузка / матч',phase_between_games:'Между матчами',phase_excluded_mode:'Исключённый режим',immediate:'Быстрые переключатели применяются сразу. Оформление — кнопкой «Применить оверлеи».',settingsHint:'Выбери раздел. Основные настройки сохраняются общей кнопкой, оверлеи — своей.',imageError:'Выбери PNG или JPEG до 2 МиБ',resetConfirm:'Сброс также удалит историю сессий и ссылки OBS.',noURL:'Сервер OBS недоступен',preview:'Предпросмотр',all:'Все поддержанные режимы',filter:'Учитываемые матчи',newPlayer:'Сначала выбери и сохрани игрока HotS',source:'Режим {mode} · {hero}',notRunning:'Ожидание локального сервера OBS'}
};
Object.assign(obsText.en,{theme:'Score theme',themeDefault:'Default',themeNeon:'Neon',themeIce:'Ice',themeBroadcast:'Scoreboard',themeRetro:'Retro',themeCustom:'Unsaved CSS',themeBuiltins:'Built-in themes',themeSaved:'My themes',themeName:'Theme name',themeSave:'Save as new theme',themeUpdate:'Update selected',themeDelete:'Delete selected',themeHelp:'Choose a theme or edit CSS below. Save changes to keep your library and apply the appearance in OBS.',themeNameError:'Enter a unique name (up to 120 bytes).',themeLimit:'Up to 20 themes; 64 KiB per theme and 192 KiB total.',themeDraft:'Theme library updated. Save changes to keep it.'});
Object.assign(obsText.ru,{theme:'Тема счётчика',themeDefault:'Стандартная',themeNeon:'Неон',themeIce:'Лёд',themeBroadcast:'Табло',themeRetro:'Ретро',themeCustom:'Несохранённый CSS',themeBuiltins:'Готовые темы',themeSaved:'Мои темы',themeName:'Название темы',themeSave:'Сохранить как новую',themeUpdate:'Обновить выбранную',themeDelete:'Удалить выбранную',themeHelp:'Выбери тему или измени CSS ниже. Сохрани изменения, чтобы записать библиотеку и применить оформление в OBS.',themeNameError:'Введи уникальное название (до 120 байт).',themeLimit:'До 20 тем; 64 КиБ на тему и 192 КиБ суммарно.',themeDraft:'Библиотека тем изменена. Сохрани изменения, чтобы записать её.'});
const obsBuiltInThemes=[
 {id:'default',label:'themeDefault',css:''},
 {id:'neon',label:'themeNeon',css:'#score { gap: 14px; padding: 12px 20px; border-radius: 12px; }\n#score:is(.nexus, .minimal).backdrop { background: #100d24ed; border: 1px solid #bb65ff; box-shadow: inset 0 0 18px #9b40ff33; }\n#score b { font-size: 32px; }\n#wins { color: #58ffdc !important; text-shadow: 0 0 12px #58ffdc99; }\n#losses { color: #ff72d2 !important; text-shadow: 0 0 12px #ff72d299; }'},
 {id:'ice',label:'themeIce',css:'#score { gap: 14px; padding: 12px 20px; border-radius: 3px; letter-spacing: 2px; }\n#score:is(.nexus, .minimal).backdrop { background: linear-gradient(120deg, #071d32f5, #133f56ed); border: 1px solid #8de4ff; border-left: 4px solid #b9f5ff; box-shadow: none; }\n#wins { color: #b9f5ff !important; }\n#losses { color: #c4cfff !important; }'},
 {id:'broadcast',label:'themeBroadcast',css:'#score { gap: 0; padding: 0; border-radius: 4px; overflow: hidden; letter-spacing: 1px; }\n#score:is(.nexus, .minimal).backdrop { background: #111827ed; border: 1px solid #ffffff44; box-shadow: none; }\n#score b { padding: 10px 18px; font-size: 32px; }\n#wins { color: #ffffff !important; background: #126047; }\n#losses { color: #ffffff !important; background: #903444; }\n#win-label, #loss-label, .divider { padding: 0 12px; }'},
 {id:'retro',label:'themeRetro',css:'#score { font-family: Consolas, monospace; gap: 14px; padding: 10px 18px; border-radius: 0; text-shadow: none; }\n#score:is(.nexus, .minimal).backdrop { background: #17190ff5; border: 2px solid #d7d58b; box-shadow: inset 0 0 0 2px #17190f; }\n#score b { font-size: 30px; }\n#wins { color: #c5ef78 !important; }\n#losses { color: #ffb36b !important; }\n.divider { color: #d7d58b; opacity: 1; }'}
];
let obsThemesDraft=[],obsSelectedTheme='';
Object.assign(obsText.en,{lobbyTitle:'Lobby statistics widget',lobbyEnabled:'Show lobby widget in OBS',lobbyHelp:'OBS → Browser source → 560 × 820. Current hero, local win rate, game counts and known players. Updates automatically from BattleLobby; no live lobby means a waiting message. Save changes to apply.',lobbyPreview:'Live widget preview'});
Object.assign(obsText.ru,{lobbyTitle:'Виджет статистики лобби',lobbyEnabled:'Показывать виджет лобби в OBS',lobbyHelp:'OBS → Источник «Браузер» → 560 × 820. Текущий герой, локальный винрейт, число игр и знакомые участники. Обновляется по BattleLobby; без текущего лобби показывает ожидание. Применяется кнопкой «Сохранить изменения».',lobbyPreview:'Предпросмотр виджета'});
for(const language of ['en','ru'])for(const [key,value]of Object.entries(obsText[language]))I18N[language]['obs.'+key]=value;
let obsState=null,obsDirty=false,obsBusy=false,obsPending=false,obsImage='',obsHistoryKey='',obsTimer;
const ot=(key,vars)=>t('obs.'+key,vars);
const obsLabel=(key)=>`<span data-i18n="obs.${key}">${ot(key)}</span>`;
const obsButton=(id,key)=>`<button type="button" class="btn btn-outline" id="${id}">${obsLabel(key)}</button>`;
function obsSection(name){
 document.querySelectorAll('.settings-page').forEach(p=>p.hidden=p.dataset.section!==name);
 document.querySelectorAll('[data-settings-section]').forEach(b=>b.setAttribute('aria-pressed',String(b.dataset.settingsSection===name)));
 $('settings-section-select').value=name;
 const savebar=$('view-settings').querySelector('.settings-savebar');if(savebar)savebar.hidden=name==='overlays';
}
function buildObsUI(){
 const view=$('view-settings'),nav=document.createElement('nav');nav.className='settings-nav';nav.setAttribute('aria-label',ot('basic'));
 nav.innerHTML=['basic','predictions','overlays','application'].map(k=>`<button type="button" data-settings-section="${k}" aria-pressed="${k==='basic'}">${obsLabel(k)}</button>`).join('')+`<select id="settings-section-select" aria-label="${ot('basic')}">${['basic','predictions','overlays','application'].map(k=>`<option value="${k}" data-i18n="obs.${k}">${ot(k)}</option>`).join('')}</select>`;
 view.querySelector('.view-heading').after(nav);
 const pages={};for(const name of ['basic','predictions','overlays','application']){const p=document.createElement('div');p.className='settings-page';p.dataset.section=name;p.hidden=name!=='basic';nav.before(p);pages[name]=p;}nav.after(...Object.values(pages));
 const cards=[...view.querySelectorAll('.settings-card')];pages.basic.append(cards[0],cards[1]);pages.predictions.append(cards[2]);
 pages.application.append(view.querySelector('.settings-language'),view.querySelector('.update-settings'));
 const advanced=$('settings-advanced'),groups=[...advanced.querySelectorAll('.settings-advanced-group')];pages.basic.append(advanced);
 for(const [group,page]of [[groups[1],pages.basic],[groups[2],pages.predictions]]){const details=document.createElement('details');details.className='secondary-details settings-advanced';const summary=document.createElement('summary');summary.textContent=group.querySelector('h3').textContent;summary.dataset.i18n=group.querySelector('h3').dataset.i18n;const body=document.createElement('div');body.className='secondary-details-body';body.append(group);details.append(summary,body);page.append(details)}
 const reset=view.querySelector('[data-action="reset-profile"]').closest('details');pages.application.append(reset);reset.querySelector('.secondary-details-body').append(line(ot('resetConfirm'),'session-notice'));
 const hint=view.querySelector('.view-heading p');hint.dataset.i18n='obs.settingsHint';hint.textContent=ot('settingsHint');
 const overlay=pages.overlays;overlay.innerHTML=`<p class="field-hint" data-i18n="obs.obsHelp">${ot('obsHelp')}</p><div id="obs-error" class="session-error" role="status"></div>
 <section class="obs-card"><h2>${obsLabel('score')}</h2><label class="obs-switch"><input type="checkbox" id="obs-enabled">${obsLabel('enabled')}</label>
 <div class="obs-fields"><label>${obsLabel('filter')}<select id="obs-ranked"><option value="false" data-i18n="obs.all">${ot('all')}</option><option value="true" data-i18n="obs.ranked">${ot('ranked')}</option></select></label>
 <label>${obsLabel('style')}<select id="obs-style"><option value="nexus" data-i18n="obs.nexus">${ot('nexus')}</option><option value="minimal" data-i18n="obs.minimal">${ot('minimal')}</option></select></label>
 <label>${obsLabel('scale')}<input id="obs-scale" type="number" min="50" max="200" value="100"></label><label>${obsLabel('winColor')}<input type="color" id="obs-win-color" value="#72dcff"></label><label>${obsLabel('lossColor')}<input type="color" id="obs-loss-color" value="#ff91b2"></label></div>
 <label class="obs-switch"><input type="checkbox" id="obs-backdrop" checked>${obsLabel('backdrop')}</label><label class="obs-switch"><input type="checkbox" id="obs-labels" checked>${obsLabel('labels')}</label>
 <div id="obs-theme-controls"><label for="obs-theme">${obsLabel('theme')}</label><select id="obs-theme" aria-describedby="obs-theme-help"></select><p id="obs-theme-help" data-i18n="obs.themeHelp">${ot('themeHelp')}</p><label for="obs-theme-name">${obsLabel('themeName')}</label><input id="obs-theme-name" type="text" maxlength="120" autocomplete="off"><div class="obs-toolbar">${obsButton('obs-theme-save','themeSave')}${obsButton('obs-theme-update','themeUpdate')}${obsButton('obs-theme-delete','themeDelete')}</div><p id="obs-theme-status" class="field-hint" role="status"></p></div>
 <label class="obs-css-label" for="obs-custom-css">${obsLabel('customCSS')}</label><textarea id="obs-custom-css" rows="8" spellcheck="false" aria-describedby="obs-custom-css-help"></textarea><p id="obs-custom-css-help" data-i18n="obs.customCSSHelp">${ot('customCSSHelp')}</p>
 <iframe id="obs-css-preview" class="obs-css-preview" sandbox="allow-same-origin" title="${ot('preview')}" hidden></iframe>
 <div class="obs-preview" aria-label="${ot('preview')}"><div id="obs-score-preview" class="obs-score nexus backdrop"><span id="obs-preview-w">W</span><b id="obs-preview-wins">7</b><span>/</span><b id="obs-preview-losses">3</b><span id="obs-preview-l">L</span></div></div>
 ${obsButton('obs-copy-score','copy')}<code class="obs-url" id="obs-score-url"></code></section>
 <section class="obs-card"><h2>${obsLabel('mask')}</h2><label class="obs-switch"><input type="checkbox" id="obs-mask-enabled">${obsLabel('maskEnabled')}</label><label class="obs-switch"><input type="checkbox" id="obs-mask-ranked" checked>${obsLabel('maskRanked')}</label>
 <p data-i18n="obs.maskHelp">${ot('maskHelp')}</p><div class="obs-fields"><label>${obsLabel('image')}<input id="obs-mask-file" type="file" accept="image/png,image/jpeg"></label></div><div class="obs-preview-mask" id="obs-mask-preview">NEXUS</div>${obsButton('obs-clear-image','clearImage')} ${obsButton('obs-copy-mask','copy')}<code class="obs-url" id="obs-mask-url"></code></section>
 <section class="obs-card lobby-settings-card"><h2>${obsLabel('lobbyTitle')}</h2><label class="obs-switch"><input type="checkbox" id="obs-lobby-enabled">${obsLabel('lobbyEnabled')}</label><p data-i18n="obs.lobbyHelp">${ot('lobbyHelp')}</p>${obsButton('obs-copy-lobby','copy')}<code class="obs-url" id="obs-lobby-url"></code><details id="obs-lobby-preview" class="secondary-details"><summary>${obsLabel('lobbyPreview')}</summary><iframe id="obs-lobby-frame" class="lobby-preview" sandbox="allow-scripts allow-same-origin" title="${ot('lobbyPreview')}"></iframe></details></section>
 <div class="obs-toolbar"><button type="button" class="btn btn-primary" id="obs-apply">${obsLabel('apply')}</button><span class="field-hint" data-i18n="obs.immediate">${ot('immediate')}</span></div>`;
 const strip=document.createElement('section');strip.className='session-strip';strip.innerHTML=`<div><small data-i18n="obs.session">${ot('session')}</small><div><strong id="session-score">0–0</strong></div><small id="session-since"></small></div><div class="obs-toolbar">${obsButton('session-new','newSession')}<button type="button" class="text-action" id="session-history-link">${obsLabel('history')}</button><label><span class="sr-only" data-i18n="obs.mask">${ot('mask')}</span><select id="session-mask-mode">${['auto','show','hide'].map(k=>`<option value="${k}" data-i18n="obs.${k}">${ot(k)}</option>`).join('')}</select></label><label class="obs-switch"><input type="checkbox" id="session-mask-enabled">${obsLabel('maskEnabled')}</label></div><small id="session-phase"></small><div id="session-feedback" class="session-notice" role="status"></div>`;
 $('overview-command')?.after(strip);if(!strip.isConnected)document.querySelector('.overview-command').after(strip);
 const journal=document.createElement('section');journal.className='obs-card';journal.id='session-journal';journal.innerHTML=`<div class="obs-toolbar"><h2 data-i18n="obs.history">${ot('history')}</h2>${obsButton('session-undo','undo')}</div><p data-i18n="obs.warning">${ot('warning')}</p><div id="session-archive"></div>`;$('view-replays').querySelector('.view-heading').after(journal);
 nav.querySelectorAll('button').forEach(b=>b.addEventListener('click',()=>obsSection(b.dataset.settingsSection)));$('settings-section-select').addEventListener('change',e=>obsSection(e.target.value));
 $('session-history-link').addEventListener('click',()=>{go('replays');journal.scrollIntoView({block:'start',behavior:'smooth'})});
 for(const id of ['obs-lobby-enabled','obs-custom-css','obs-ranked','obs-style','obs-scale','obs-win-color','obs-loss-color','obs-backdrop','obs-labels','obs-mask-ranked'])$(id).addEventListener('input',()=>{obsDirty=true;obsPreview()});
 obsThemeEvents();
 $('obs-mask-file').addEventListener('change',async e=>{try{const file=e.target.files[0];if(!file)return;if(!['image/png','image/jpeg'].includes(file.type)||file.size>2*1024*1024)throw Error(ot('imageError'));obsImage=await new Promise((resolve,reject)=>{const reader=new FileReader();reader.onload=()=>resolve(reader.result);reader.onerror=reject;reader.readAsDataURL(file)});obsDirty=true;obsPreview()}catch(e){toast(e.message,true)}});
 $('obs-clear-image').addEventListener('click',()=>{obsImage='';$('obs-mask-file').value='';obsDirty=true;obsPreview()});
 $('obs-apply').addEventListener('click',()=>obsRun(async()=>{obsState=await api('sessions/preferences','POST',{preferences:obsInputs()});obsDirty=false;renderObs();toast(ot('saved'))}));
 $('session-new').addEventListener('click',()=>obsMutate('new',{}));$('session-undo').addEventListener('click',()=>obsMutate('undo',{}));
 for(const [id,key]of [['obs-enabled','enabled'],['obs-mask-enabled','maskEnabled'],['session-mask-enabled','maskEnabled']])$(id).addEventListener('change',e=>obsQuick({[key]:e.target.checked}));
 $('session-mask-mode').addEventListener('change',e=>obsQuick({maskMode:e.target.value}));
 $('obs-lobby-preview').addEventListener('toggle',()=>{const frame=$('obs-lobby-frame');if($('obs-lobby-preview').open)frame.src=new URL('./assets/lobby-widget.html?preview=1',location.href).href;else frame.removeAttribute('src')});
 for(const kind of ['score','mask','lobby'])$('obs-copy-'+kind).addEventListener('click',()=>obsRun(async()=>{const value=obsState?.[kind+'URL'];if(!value)throw Error(ot('noURL'));await navigator.clipboard.writeText(value);toast(ot('copied'))}));
 obsSection('basic');obsPreview();loadObs();obsTimer=setInterval(loadObs,3000);
}
function obsInputs(){return {...obsState.preferences,lobbyWidgetEnabled:$('obs-lobby-enabled').checked,scoreThemes:obsThemesDraft.map(v=>({...v})),customCSS:$('obs-custom-css').value,rankedOnly:$('obs-ranked').value==='true',style:$('obs-style').value,scale:Number($('obs-scale').value),winColor:$('obs-win-color').value,lossColor:$('obs-loss-color').value,backdrop:$('obs-backdrop').checked,labels:$('obs-labels').checked,maskRankedOnly:$('obs-mask-ranked').checked,maskImage:obsImage}}
function obsPreview(){
 const apply=$('obs-apply');if(apply)apply.classList.toggle('has-draft',obsDirty);
 const el=$('obs-score-preview');if(!el)return;el.className='obs-score '+$('obs-style').value+($('obs-backdrop').checked?' backdrop':'');el.style.transform=`scale(${Number($('obs-scale').value)/100})`;
 $('obs-preview-wins').style.color=$('obs-win-color').value;$('obs-preview-losses').style.color=$('obs-loss-color').value;for(const id of ['obs-preview-w','obs-preview-l'])$(id).hidden=!$('obs-labels').checked;
 obsCSSPreview();
 const panel=$('obs-mask-preview');panel.style.backgroundImage=obsImage?`url("${obsImage}")`:'';panel.textContent=obsImage?'':'NEXUS';
}
async function obsRun(fn){if(obsBusy||state.closing)return;obsBusy=true;document.querySelectorAll('.session-strip,.settings-page[data-section=overlays],#session-journal').forEach(e=>e.classList.add('obs-busy'));try{await fn()}catch(e){if(!state.closing)toast(e.message,true);renderObs()}finally{obsBusy=false;document.querySelectorAll('.obs-busy').forEach(e=>e.classList.remove('obs-busy'))}}
async function obsMutate(route,data){return obsRun(async()=>{obsState=await api('sessions/'+route,'POST',data);renderObs()})}
async function obsQuick(patch){return obsRun(async()=>{if(!obsState)throw Error(ot('notRunning'));obsState=await api('sessions/preferences','POST',{preferences:{...obsState.preferences,...patch}});renderObs()})}
async function loadObs(){if(obsBusy||obsPending||state.closing)return;obsPending=true;try{obsState=await api('sessions');renderObs()}catch(e){if(!state.closing)$('session-feedback').textContent=e.message}finally{obsPending=false}}
function renderObs(){
 if(!obsState||!$('session-score'))return;const s=obsState,p=s.preferences;
 $('session-score').textContent=p.enabled?`${s.wins}–${s.losses}`:'—';$('session-since').textContent=s.current?ot('since',{time:new Date(s.current.startedAt).toLocaleString(lang==='ru'?'ru-RU':'en-US')}):ot('off');
 $('session-new').disabled=!p.enabled||!s.player;$('session-mask-mode').value=p.maskMode;$('session-mask-mode').disabled=!p.maskEnabled;$('session-mask-enabled').checked=p.maskEnabled;
 $('session-phase').textContent=ot('phase_'+s.phase);$('session-feedback').textContent=s.error||s.scanError||'';$('obs-error').textContent=s.error||s.scanError||'';
 $('obs-enabled').checked=p.enabled;$('obs-mask-enabled').checked=p.maskEnabled;
 if(!obsDirty){$('obs-lobby-enabled').checked=!!p.lobbyWidgetEnabled;$('obs-theme-status').textContent='';obsThemesDraft=(p.scoreThemes||[]).map(v=>({...v}));$('obs-custom-css').value=p.customCSS||'';$('obs-ranked').value=String(p.rankedOnly);$('obs-style').value=p.style;$('obs-scale').value=p.scale;$('obs-win-color').value=p.winColor;$('obs-loss-color').value=p.lossColor;$('obs-backdrop').checked=p.backdrop;$('obs-labels').checked=p.labels;$('obs-mask-ranked').checked=p.maskRankedOnly;obsImage=p.maskImage||'';obsThemeOptions();obsPreview()}
 for(const kind of ['score','mask','lobby']){$('obs-'+kind+'-url').textContent=s[kind+'URL']||ot('noURL');$('obs-copy-'+kind).disabled=!s[kind+'URL']}
 const key=JSON.stringify([s.sessions,s.matches,s.player,p.rankedOnly,lang]);if(key!==obsHistoryKey){obsHistoryKey=key;renderSessions()}
}
function renderSessions(){
 if(typeof renderWideSessions==='function'&&wideState.ready){renderWideSessions();return}
 const s=obsState,box=$('session-archive'),matches=(s.matches||[]).filter(m=>m.player===s.player),sessions=(s.sessions||[]).filter(v=>v.player===s.player);box.replaceChildren();
 if(!sessions.length)box.append(line(ot('empty'),'field-hint'));
 const pending=matches.filter(m=>m.needsAssignment);if(pending.length){box.append(line(ot('unassigned'),'session-notice'));for(const m of pending)box.append(sessionMatch(m,sessions))}
 for(const v of [...sessions].reverse()){
  const rows=matches.filter(m=>m.sessionId===v.id&&!m.needsAssignment).sort((a,b)=>(b.startedAt||b.recordedAt)-(a.startedAt||a.recordedAt)),count=result=>rows.filter(m=>!m.excluded&&m.result===result&&(!s.preferences.rankedOnly||m.mode===50091)).length;
  const details=document.createElement('details');details.className='session-archive';details.open=v.id===s.current?.id;const summary=document.createElement('summary');summary.textContent=`${new Date(v.startedAt).toLocaleString(lang==='ru'?'ru-RU':'en-US')} · ${count('win')}–${count('loss')}`;details.append(summary);for(const m of rows)details.append(sessionMatch(m,sessions));if(!rows.length)details.append(line(ot('noMatches'),'field-hint'));box.append(details)
 }
 $('session-undo').disabled=sessions.length<2;
}
function sessionMatch(m,sessions){
 const row=document.createElement('div');row.className='session-match';const title=document.createElement('strong');title.textContent=`${m.map||'—'} · ${ot(m.result==='win'?'win':m.result==='loss'?'loss':'unknown')}`;
 row.append(title,line([new Date(m.startedAt||m.recordedAt||m.discoveredAt).toLocaleString(),ot('source',{mode:m.mode||'?',hero:m.hero||'—'}),m.manual?ot('manual'):'',m.excluded?ot('excluded'):'',m.needsAssignment?ot('pending'):''].filter(Boolean).join(' · '),'field-hint'));
 const tools=document.createElement('div');tools.className='obs-toolbar';
 const button=(label,fn)=>{const b=document.createElement('button');b.type='button';b.className='text-action';b.textContent=label;b.addEventListener('click',fn);tools.append(b)};
 button(ot(m.excluded?'include':'exclude'),()=>obsMutate('match',{matchId:m.id,excluded:!m.excluded}));
 if(m.startedAt&&(!obsState.current||m.startedAt>obsState.current.startedAt))button(ot('boundary'),()=>obsMutate('boundary',{matchId:m.id}));
 const select=document.createElement('select');select.setAttribute('aria-label',ot('assignment'));select.append(new Option(ot('assignment'),''));for(const v of sessions)select.append(new Option(new Date(v.startedAt).toLocaleString(),v.id));select.addEventListener('change',()=>{if(select.value)obsMutate('match',{matchId:m.id,sessionId:select.value})});tools.append(select);
 const result=document.createElement('select');result.setAttribute('aria-label',ot('score'));for(const value of ['win','loss','unknown'])result.append(new Option(ot(value),value));result.value=['win','loss'].includes(m.result)?m.result:'unknown';result.addEventListener('change',()=>obsMutate('match',{matchId:m.id,result:result.value}));tools.append(result);row.append(tools);return row;
}
window.addEventListener('DOMContentLoaded',buildObsUI);
window.addEventListener('pagehide',()=>clearInterval(obsTimer));

// Keep user selectors, including body and *, isolated from the settings page.
function obsCSSPreview(){
 const frame=$('obs-css-preview'),css=$('obs-custom-css').value;
 frame.hidden=!css.trim();$('obs-score-preview').parentElement.hidden=!!css.trim();
 if(frame.hidden)return;
 const doc=frame.contentDocument;if(!doc)return;
 if(!doc.getElementById('score')){
  const link=doc.createElement('link');link.rel='stylesheet';link.href=new URL('./assets/overlay.css',location.href).href;doc.head.append(link);
  doc.body.innerHTML='<div id="score"><span id="win-label">W</span><b id="wins">7</b><span class="divider">/</span><b id="losses">3</b><span id="loss-label">L</span></div>';
 }
 const score=doc.getElementById('score');score.className=$('obs-style').value+($('obs-backdrop').checked?' backdrop':'');score.style.transform=`scale(${Number($('obs-scale').value)/100})`;
 doc.getElementById('wins').style.color=$('obs-win-color').value;doc.getElementById('losses').style.color=$('obs-loss-color').value;
 for(const id of ['win-label','loss-label'])doc.getElementById(id).hidden=!$('obs-labels').checked;
 const sheet=new frame.contentWindow.CSSStyleSheet();sheet.replaceSync(css);doc.adoptedStyleSheets=[sheet];
}

function obsThemeOptions(){
 const select=$('obs-theme'),css=$('obs-custom-css').value;
 const old=obsSelectedTheme;
 select.replaceChildren();
 const group=(label,rows)=>{const g=document.createElement('optgroup');g.label=ot(label);for(const [name,value]of rows)g.append(new Option(name,value));select.append(g)};
 group('themeBuiltins',obsBuiltInThemes.map(v=>[ot(v.label),'builtin:'+v.id]));
 if(obsThemesDraft.length)group('themeSaved',obsThemesDraft.map((v,i)=>[v.name,'saved:'+i]));
 const choices=[...obsBuiltInThemes.map(v=>({value:'builtin:'+v.id,css:v.css})),...obsThemesDraft.map((v,i)=>({value:'saved:'+i,css:v.css}))];
 obsSelectedTheme=choices.find(v=>v.value===old&&v.css===css)?.value||choices.find(v=>v.value.startsWith('saved:')&&v.css===css)?.value||choices.find(v=>v.css===css)?.value||'';
 if(!obsSelectedTheme)select.append(new Option(ot('themeCustom'),''));
 select.value=obsSelectedTheme;
 const selected=obsSelectedTheme.startsWith('saved:');
 if(selected&&!$('obs-theme-name').value)$('obs-theme-name').value=obsThemesDraft[Number(obsSelectedTheme.slice(6))].name;
 $('obs-theme-update').disabled=!selected;$('obs-theme-delete').disabled=!selected;
}
function obsThemeChanged(){obsDirty=true;obsPreview();if(typeof wideUpdateDraft==='function')wideUpdateDraft()}
function obsThemeEvents(){
 $('obs-theme').addEventListener('change',()=>{
  obsSelectedTheme=$('obs-theme').value;
  const saved=obsSelectedTheme.startsWith('saved:');
  const theme=saved?obsThemesDraft[Number(obsSelectedTheme.slice(6))]:obsBuiltInThemes.find(v=>'builtin:'+v.id===obsSelectedTheme);
  if(!theme)return;
  $('obs-custom-css').value=theme.css;$('obs-theme-name').value=saved?theme.name:'';
  obsThemeOptions();obsThemeChanged();
 });
 $('obs-custom-css').addEventListener('input',()=>{
  // Keep a saved selection while editing it so Update remains available.
  if(!obsSelectedTheme.startsWith('saved:'))obsThemeOptions();
 });
 const edit=(action)=>{
  const index=obsSelectedTheme.startsWith('saved:')?Number(obsSelectedTheme.slice(6)):-1;
  if(action!=='save'&&index<0)return;
  if(action==='delete'){
   obsThemesDraft.splice(index,1);obsSelectedTheme='';
   $('obs-theme-name').value='';
  }else{
   const name=$('obs-theme-name').value.trim(),css=$('obs-custom-css').value,bytes=v=>new TextEncoder().encode(v).length;
   if(!name||bytes(name)>120||/[\r\n\t]/.test(name)||obsThemesDraft.some((v,i)=>v.name.toLowerCase()===name.toLowerCase()&&(action==='save'||i!==index))){toast(ot('themeNameError'),true);return}
   const themes=obsThemesDraft.map(v=>({...v}));
   if(action==='save')themes.push({name,css});else themes[index]={name,css};
   if(themes.length>20||themes.some(v=>bytes(v.css)>65536)||themes.reduce((n,v)=>n+bytes(v.css),0)>196608){toast(ot('themeLimit'),true);return}
   obsThemesDraft=themes;obsSelectedTheme='saved:'+(action==='save'?themes.length-1:index);
  }
  obsThemeOptions();obsThemeChanged();$('obs-theme-status').textContent=ot('themeDraft');
 };
 for(const action of ['save','update','delete'])$('obs-theme-'+action).addEventListener('click',()=>edit(action));
}
