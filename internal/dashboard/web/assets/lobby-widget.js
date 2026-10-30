'use strict';
const preview=new URLSearchParams(location.search).get('preview')==='1';
if(preview)document.documentElement.classList.add('preview');
const text={
 ru:{title:'Статистика лобби',waiting:'Ожидание текущего лобби…',unavailable:'Статистика временно недоступна',offline:'Нет связи с Companion',heroUnknown:'Герой не подтверждён',streamer:'Стример',player:'Знакомый игрок',onHero:'На этом герое',overall:'Все герои · этот режим',games:n=>n+' игр',familiar:n=>'Знакомые участники · '+n,empty:'Других знакомых игроков в этом лобби пока нет.',small:'Мало данных: меньше 20 игр.',pending:'Герой ещё не подтверждён. Показана общая статистика игрока в этом режиме.',nodata:'На этом герое пока нет наблюдений.',footer:'Локальные реплеи · не прогноз победы',meeting:n=>'Встреч со стримером: '+n,hero:'На герое',total:'Всего',modes:{50091:'Storm League',50001:'Быстрая игра',50051:'Без рейтинга',50101:'ARAM',0:'Режим не определён'}},
 en:{title:'Lobby statistics',waiting:'Waiting for a current lobby…',unavailable:'Statistics temporarily unavailable',offline:'Companion disconnected',heroUnknown:'Hero not confirmed',streamer:'Streamer',player:'Known player',onHero:'On this hero',overall:'All heroes · this mode',games:n=>n+' games',familiar:n=>'Known participants · '+n,empty:'No other known players in this lobby yet.',small:'Small sample: fewer than 20 games.',pending:'Hero not confirmed yet. Showing the player’s overall results in this mode.',nodata:'No observations on this hero yet.',footer:'Local replays · not a win prediction',meeting:n=>'Matches with streamer: '+n,hero:'On hero',total:'Overall',modes:{50091:'Storm League',50001:'Quick Match',50051:'Unranked Draft',50101:'ARAM',0:'Mode unknown'}}
};
let inFlight=false,lastEnabled=null,language='en';
const el=id=>document.getElementById(id),rate=c=>c&&c.games>0&&Number.isFinite(c.winRate)?c.winRate.toFixed(1)+'%':'—';
function clearLive(){el('widget-live').hidden=true;el('widget-familiar').replaceChildren();for(const id of ['widget-hero','widget-name','widget-rate','widget-games','widget-note','widget-mode'])el(id).textContent=''}
function drawLobby(s){
 language=s.language==='ru'?'ru':'en';const t=text[language];lastEnabled=!!s.enabled;
 el('lobby-widget').hidden=preview?false:!s.enabled;clearLive();el('widget-title').textContent=t.title;el('widget-footer').textContent=t.footer;el('widget-status').hidden=false;
 if(!s.enabled&&!preview)return;
 if(s.phase!=='live'){el('widget-status').textContent=s.phase==='unavailable'?t.unavailable:t.waiting;return}
 el('widget-status').hidden=true;el('widget-live').hidden=false;
 const p=s.streamer||{},c=p.onHero||p.overall||{games:0};
 el('widget-mode').textContent=t.modes[s.mode]||t.modes[0];el('widget-name').textContent=p.name||t.streamer;
 el('widget-hero').textContent=p.hero||t.heroUnknown;el('widget-mark').textContent=p.hero?p.hero.slice(0,2).toUpperCase():'?';
 el('widget-scope').textContent=p.onHero?t.onHero:t.overall;el('widget-rate').textContent=rate(c);el('widget-games').textContent=t.games(c.games||0);
 el('widget-note').textContent=s.heroPending?t.pending:!c.games?t.nodata:c.games<20?t.small:'';
 const familiar=Array.isArray(s.familiar)?s.familiar.slice(0,9):[];el('widget-familiar-title').textContent=t.familiar(familiar.length);el('widget-empty').hidden=!!familiar.length;el('widget-empty').textContent=t.empty;
 for(const p of familiar){
  const row=document.createElement('div');row.className='familiar-row';const name=document.createElement('strong');name.textContent=p.name||t.player;name.title=name.textContent;
  const description=document.createElement('small');description.textContent=(p.hero?p.hero+' · ':'')+t.meeting(p.meetings||0);
  const score=document.createElement('div');score.className='person-rate';score.textContent=t.total+': '+rate(p.overall)+' · '+t.games(p.overall?.games||0);
  if(p.onHero){const hero=document.createElement('span');hero.textContent=t.hero+': '+rate(p.onHero)+' · '+t.games(p.onHero.games||0);score.append(hero)}
  row.append(name,description,score);el('widget-familiar').append(row);
 }
}
async function pollLobby(){
 if(inFlight)return;inFlight=true;
 try{const url=preview?'../api/lobby-widget':'lobby-state';const r=await fetch(url,{cache:'no-store',headers:preview?{'X-Companion':'1'}:{},signal:AbortSignal.timeout(4000)});if(!r.ok)throw Error();drawLobby(await r.json())}
 catch(e){clearLive();el('lobby-widget').hidden=lastEnabled===false&&!preview;el('widget-title').textContent=text[language].title;el('widget-status').hidden=false;el('widget-status').textContent=text[language].offline}
 finally{inFlight=false}
}
pollLobby();setInterval(pollLobby,1500);
