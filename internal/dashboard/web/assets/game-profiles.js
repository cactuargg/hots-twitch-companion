'use strict';
const gameProfileText={
 en:{title:'Game profiles',active:'Active account',select:'Editing profile',name:'Profile name',add:'Add account',remove:'Delete profile',account:'Account',help:'Twitch, predictions, OBS links and themes are shared. Match history and scores follow the selected player. Save changes to activate the selected profile.',blocked:'Stop monitoring and finish the current Prediction before switching accounts. Save other drafts first.',added:'New profile: choose yourself from a replay of this account, then save changes.',needName:'Enter a unique profile name.',deleted:'Profile removed from the draft. History is preserved; save changes to apply.',saved:'Game profile switched',none:'No game profile yet'},
 ru:{title:'Игровые профили',active:'Активный аккаунт',select:'Редактируемый профиль',name:'Название профиля',add:'Добавить аккаунт',remove:'Удалить профиль',account:'Аккаунт',help:'Twitch, прогнозы, ссылки OBS и темы общие. История и счёт относятся к выбранному игроку. Сохрани изменения, чтобы активировать выбранный профиль.',blocked:'Останови мониторинг и заверши текущий прогноз перед сменой аккаунта. Сначала сохрани остальные изменения.',added:'Новый профиль: выбери себя из реплея этого аккаунта и сохрани изменения.',needName:'Введи уникальное название профиля.',deleted:'Профиль удалён из черновика. История сохранена; сохрани изменения для применения.',saved:'Игровой профиль переключён',none:'Игровой профиль ещё не добавлен'}
};
for(const language of ['en','ru'])for(const [key,value]of Object.entries(gameProfileText[language]))I18N[language]['profiles.'+key]=value;
const gpt=key=>t('profiles.'+key);
let gameProfilesDraft=[],gameProfileID='',gameProfileBusy=false;
function profileBlocked(){const s=state.status||{};return !!(s.running||s.active||s.creationUnconfirmed||s.authPending||state.saving||state.languageSaving||state.busy||state.wideSaving||gameProfileBusy)}
function profileDirty(){state.hasUnsaved=true;renderSettingsDraft();if(typeof wideUpdateDraft==='function')wideUpdateDraft();renderGameProfiles()}
function stashGameProfile(){
 const p=gameProfilesDraft.find(p=>p.id===gameProfileID);if(!p)return;
 if($('game-profile-name'))p.name=$('game-profile-name').value.trim();
 p.playerName=$('playerName').value.trim();p.playerToonHandle=$('toonHandle').value.trim();p.replayRoot=$('replayRoot').value.trim();
}
function gameProfileInputs(){stashGameProfile();return {gameProfiles:gameProfilesDraft.map(p=>({...p})),activeGameProfile:gameProfileID}}
function loadGameProfiles(c){
 gameProfilesDraft=(c.gameProfiles||[]).map(p=>({...p}));gameProfileID=c.activeGameProfile||'';
 // Old configurations in previews/fixtures get the same migration as the server.
 if(!gameProfilesDraft.length&&(c.playerName||c.playerToonHandle)){
  gameProfileID='default';gameProfilesDraft=[{id:'default',name:c.playerName||'HotS',playerName:c.playerName||'',playerToonHandle:c.playerToonHandle||'',replayRoot:c.replayRoot||''}];
 }
 if(!$('game-profile-select'))return;
 gameProfileOptions();$('game-profile-name').value=gameProfilesDraft.find(p=>p.id===gameProfileID)?.name||'';renderGameProfiles();
}
function gameProfileOptions(){
 const select=$('game-profile-select');select.replaceChildren();
 for(const p of gameProfilesDraft)select.append(new Option(p.name,p.id));
 if(!gameProfilesDraft.length)select.append(new Option(gpt('none'),''));select.value=gameProfileID;
}
function chooseGameProfile(id){
 stashGameProfile();const p=gameProfilesDraft.find(p=>p.id===id);if(!p)return;
 gameProfileID=id;$('game-profile-name').value=p.name;
 $('playerName').value=p.playerName||'';$('toonHandle').value=p.playerToonHandle||'';$('replayRoot').value=p.replayRoot||'';
 gameProfileOptions();renderPlayerPickers();profileDirty();
}
function renderGameProfiles(){
 if(!$('game-profile-select'))return;
 const blocked=profileBlocked();
 $('game-profile-select').disabled=blocked;
 $('game-profile-add').disabled=blocked||gameProfilesDraft.length>=8;
 $('game-profile-delete').disabled=blocked||gameProfilesDraft.length<2;
 $('game-profile-name').disabled=!!(state.saving||state.languageSaving||state.wideSaving||gameProfileBusy)||!gameProfilesDraft.length;
 const select=$('game-profile-active'),c=state.config||{};
 const profiles=c.gameProfiles?.length?c.gameProfiles:(gameProfilesDraft.length&&!(state.hasUnsaved)?gameProfilesDraft:[]);
 const key=JSON.stringify([profiles.map(p=>[p.id,p.name,p.playerName]),lang]);
 if(select.dataset.key!==key){select.replaceChildren();for(const p of profiles)select.append(new Option(p.name+(p.playerName?' · '+p.playerName:''),p.id));if(!profiles.length)select.append(new Option(gpt('none'),''));select.dataset.key=key}
 select.value=c.activeGameProfile||(!c.gameProfiles?.length?'default':'');
 select.disabled=blocked||!!state.hasUnsaved||(typeof obsDirty!=='undefined'&&obsDirty)||profiles.length<2;
 $('game-profile-active-help').textContent=(blocked||state.hasUnsaved||(typeof obsDirty!=='undefined'&&obsDirty))?gpt('blocked'):gpt('help');
}
async function switchGameProfile(id){
 if(profileBlocked()||state.hasUnsaved||(typeof obsDirty!=='undefined'&&obsDirty)){renderGameProfiles();toast(gpt('blocked'),true);return}
 const c=state.config,p=c?.gameProfiles?.find(p=>p.id===id);if(!p)return;
 gameProfileBusy=true;state.saving=true;renderGameProfiles();
 try{
  const x=await api('config','POST',{...c,activeGameProfile:id,playerName:p.playerName||'',playerToonHandle:p.playerToonHandle||'',replayRoot:p.replayRoot||'',confirmedRealMode:true});
  state.config=x.config;setInputs(x.config);await loadStatus();await reloadReplays();if(typeof loadObs==='function')await loadObs();toast(gpt('saved'));
 }catch(e){toast(e.message,true)}finally{gameProfileBusy=false;state.saving=false;renderGameProfiles()}
}
function buildGameProfiles(){
 const panel=document.createElement('section');panel.className='game-profiles';panel.innerHTML=`<h3 data-i18n="profiles.title">${gpt('title')}</h3><label for="game-profile-select" data-i18n="profiles.select">${gpt('select')}</label><select id="game-profile-select"></select><label for="game-profile-name" data-i18n="profiles.name">${gpt('name')}</label><input id="game-profile-name" maxlength="80" autocomplete="off"><div class="obs-toolbar"><button type="button" class="btn btn-outline" id="game-profile-add" data-i18n="profiles.add">${gpt('add')}</button><button type="button" class="btn btn-outline" id="game-profile-delete" data-i18n="profiles.remove">${gpt('remove')}</button></div><p class="field-hint" data-i18n="profiles.help">${gpt('help')}</p>`;
 $('playerName').closest('.settings-card').querySelector('h2').after(panel);
 const active=document.createElement('section');active.className='game-profiles game-profile-home';active.innerHTML=`<label for="game-profile-active" data-i18n="profiles.active">${gpt('active')}</label><select id="game-profile-active"></select><small id="game-profile-active-help" class="field-hint"></small>`;
 $('view-dashboard').querySelector('.view-heading').after(active);
 $('game-profile-select').addEventListener('change',e=>{if(profileBlocked()){gameProfileOptions();return}chooseGameProfile(e.target.value)});
 $('game-profile-active').addEventListener('change',e=>switchGameProfile(e.target.value));
 $('game-profile-name').addEventListener('input',()=>{stashGameProfile();gameProfileOptions();profileDirty()});
 $('game-profile-add').addEventListener('click',()=>{
  if(profileBlocked()||gameProfilesDraft.length>=8)return;stashGameProfile();
  let n=gameProfilesDraft.length+1;while(gameProfilesDraft.some(p=>p.name===gpt('account')+' '+n))n++;
  const id=crypto.randomUUID();gameProfilesDraft.push({id,name:gpt('account')+' '+n,playerName:'',playerToonHandle:'',replayRoot:''});chooseGameProfile(id);toast(gpt('added'));
 });
 $('game-profile-delete').addEventListener('click',()=>{
  if(profileBlocked()||gameProfilesDraft.length<2)return;
  gameProfilesDraft=gameProfilesDraft.filter(p=>p.id!==gameProfileID);chooseGameProfile(gameProfilesDraft[0].id);toast(gpt('deleted'));
 });
 for(const id of ['playerName','toonHandle','replayRoot'])$(id).addEventListener('input',renderGameProfiles);
 if(state.config)loadGameProfiles(state.config);else renderGameProfiles();
}
window.addEventListener('DOMContentLoaded',buildGameProfiles);
