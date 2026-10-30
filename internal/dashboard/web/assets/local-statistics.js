'use strict';
const localStatsText={
 en:{title:'Local replay statistics',help:'Observed results from local replays across all game profiles. These are not global win rates or predictions. Import old replays once; recent completed replays are collected while Companion is running, even with the session counter off.',heroes:'Heroes',players:'Players',mode:'Game mode',all:'All modes',ranked:'Storm League',quick:'Quick Match',aram:'ARAM',unranked:'Unranked Draft',unknown:'Unknown / custom',search:'Find a hero, player or ID',games:'Games',wins:'Wins',losses:'Losses',rate:'Win rate',name:'Hero / player',empty:'No observations yet. Import replays or wait for completed matches.',noRows:'No observations for these filters.',coverage:'Replays: {n} · counted player results: {p} · skipped entries: {s}',small:'Small sample',back:'All heroes and players',player:'Heroes played by {name}',prev:'Previous',next:'Next',page:'Page {n} of {total}',import:'Import replays',refresh:'Refresh',loading:'Loading statistics…',note:'Humans with a stable player ID and a known win/loss are counted. A hero appearing twice in one match contributes two player results. Copies of a replay count once. Session score edits and exclusions do not change this separate replay database.'},
 ru:{title:'Локальная статистика реплеев',help:'Результаты из локальных реплеев по всем игровым профилям. Это не глобальный винрейт и не прогноз. Старые реплеи импортируются кнопкой; новые завершённые матчи собираются, пока Companion работает, даже при выключенном счётчике сессии.',heroes:'Герои',players:'Игроки',mode:'Режим игры',all:'Все режимы',ranked:'Storm League',quick:'Быстрая игра',aram:'ARAM',unranked:'Без рейтинга',unknown:'Не определён / своя игра',search:'Найти героя, игрока или ID',games:'Игр',wins:'Побед',losses:'Поражений',rate:'Винрейт',name:'Герой / игрок',empty:'Наблюдений пока нет. Импортируй реплеи или дождись завершения матчей.',noRows:'Нет наблюдений для этих фильтров.',coverage:'Реплеев: {n} · учтено результатов игроков: {p} · пропущено записей: {s}',small:'Малая выборка',back:'Все герои и игроки',player:'Герои игрока {name}',prev:'Назад',next:'Далее',page:'Страница {n} из {total}',import:'Импортировать реплеи',refresh:'Обновить',loading:'Загрузка статистики…',note:'Учитываются люди с постоянным ID и известной победой/поражением. Два одинаковых героя в одном матче дают два результата игроков. Копии реплея учитываются один раз. Правки счёта и исключения в истории сессий не меняют отдельную базу реплеев.'}
};
for(const language of ['ru','en'])for(const[key,value]of Object.entries(localStatsText[language]))I18N[language]['localStats.'+key]=value;
const lst=(key,vars)=>t('localStats.'+key,vars);
const localStats={page:1,player:'',playerName:'',request:0,timer:null,searchTimer:null};
function localStatsLabel(key){return `<span data-i18n="localStats.${key}">${lst(key)}</span>`}
function buildLocalStatistics(){
 const panel=document.createElement('details');panel.id='local-statistics';panel.className='secondary-details local-statistics';panel.innerHTML=`<summary>${localStatsLabel('title')}</summary><div class="secondary-details-body"><p data-i18n="localStats.help">${lst('help')}</p><div class="local-stats-controls"><label>${localStatsLabel('name')}<select id="local-stats-kind"><option value="heroes" data-i18n="localStats.heroes">${lst('heroes')}</option><option value="players" data-i18n="localStats.players">${lst('players')}</option></select></label><label>${localStatsLabel('mode')}<select id="local-stats-mode">${[['all','all'],['50091','ranked'],['50001','quick'],['50101','aram'],['50051','unranked'],['0','unknown']].map(([id,key])=>`<option value="${id}" data-i18n="localStats.${key}">${lst(key)}</option>`).join('')}</select></label><label>${localStatsLabel('search')}<input id="local-stats-search" type="search" maxlength="160"></label></div><div class="obs-toolbar"><button id="local-stats-import" class="btn btn-outline" type="button">${localStatsLabel('import')}</button><button id="local-stats-refresh" class="btn btn-outline" type="button">${localStatsLabel('refresh')}</button><button id="local-stats-back" class="text-action" type="button" hidden>${localStatsLabel('back')}</button></div><p id="local-stats-scope"></p><p id="local-stats-coverage" class="field-hint"></p><p id="local-stats-error" role="status"></p><div id="local-stats-table" class="local-stats-table"></div><div id="local-stats-pages" class="obs-toolbar"></div><p class="field-hint" data-i18n="localStats.note">${lst('note')}</p></div>`;
 const home=$('view-dashboard');(home.querySelector('.wide-workspace')||home.querySelector('.overview-summary')).after(panel);
 panel.addEventListener('toggle',()=>{if(panel.open)loadLocalStatistics()});
 for(const id of ['local-stats-kind','local-stats-mode'])$(id).addEventListener('change',()=>{localStats.page=1;if(id==='local-stats-kind'){localStats.player='';localStats.playerName=''}loadLocalStatistics()});
 $('local-stats-search').addEventListener('input',()=>{clearTimeout(localStats.searchTimer);localStats.searchTimer=setTimeout(()=>{localStats.page=1;loadLocalStatistics()},250)});
 $('local-stats-refresh').addEventListener('click',loadLocalStatistics);
 $('local-stats-import').addEventListener('click',async()=>{try{await wideImport();await loadLocalStatistics()}catch(e){$('local-stats-error').textContent=e.message}});
 $('local-stats-back').addEventListener('click',()=>{localStats.player='';localStats.playerName='';localStats.page=1;$('local-stats-search').value='';loadLocalStatistics()});
 localStats.timer=setInterval(()=>{if(panel.open&&!document.hidden)loadLocalStatistics()},15000);
}
async function loadLocalStatistics(){
 if(!$('local-statistics')?.open||state.closing)return;
 const request=++localStats.request;
 const query=new URLSearchParams({mode:$('local-stats-mode').value,kind:$('local-stats-kind').value,search:$('local-stats-search').value.trim(),player:localStats.player,page:String(localStats.page)});
 $('local-stats-error').textContent=lst('loading');
 $('local-stats-import').disabled=!!obsState?.import?.running;
 try{
  const data=await api('statistics?'+query);
  if(request!==localStats.request||state.closing)return;
  localStats.page=data.page;
  $('local-stats-error').textContent='';$('local-stats-back').hidden=!localStats.player;
  $('local-stats-scope').textContent=localStats.player?lst('player',{name:localStats.playerName}):'';
  $('local-stats-coverage').textContent=lst('coverage',{n:data.matches,p:data.observations,s:data.skipped});
  const box=$('local-stats-table');box.replaceChildren();
  if(!data.rows.length){box.append(line(lst(data.matches?'noRows':'empty'),'field-hint'))}else{
   const table=document.createElement('table'),head=document.createElement('thead'),header=document.createElement('tr');
   for(const key of ['name','games','wins','losses','rate']){const th=document.createElement('th');th.scope='col';th.textContent=lst(key);header.append(th)}head.append(header);table.append(head);
   const body=document.createElement('tbody');
   for(const row of data.rows){
    const tr=document.createElement('tr'),name=document.createElement('td');
    if(data.kind==='players'){
     const button=document.createElement('button');button.type='button';button.className='text-action';button.textContent=row.name;
     button.addEventListener('click',()=>{localStats.player=row.key;localStats.playerName=row.name;localStats.page=1;$('local-stats-kind').value='heroes';$('local-stats-search').value='';loadLocalStatistics()});name.append(button,line(row.key,'field-hint'));
    }else{name.textContent=row.name}
    if(row.games<20)name.append(line(lst('small'),'field-hint'));
    tr.append(name);for(const value of [row.games,row.wins,row.losses,row.winRate.toFixed(1)+'%']){const td=document.createElement('td');td.textContent=value;tr.append(td)}body.append(tr);
   }table.append(body);box.append(table);
  }
  const pages=$('local-stats-pages');pages.replaceChildren();
  for(const [key,step]of [['prev',-1],['next',1]]){const b=document.createElement('button');b.type='button';b.className='btn btn-outline';b.textContent=lst(key);b.disabled=step<0?data.page<=1:data.page>=data.pages;b.addEventListener('click',()=>{localStats.page+=step;loadLocalStatistics()});pages.append(b)}
  pages.append(line(lst('page',{n:data.page,total:data.pages}),'field-hint'));
 }catch(e){if(request===localStats.request&&!state.closing)$('local-stats-error').textContent=e.message}
}
window.addEventListener('DOMContentLoaded',buildLocalStatistics);
window.addEventListener('pagehide',()=>{clearInterval(localStats.timer);clearTimeout(localStats.searchTimer)});
