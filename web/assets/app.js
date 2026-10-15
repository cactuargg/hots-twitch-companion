function monitoringView(st) {
 const phase=st.monitorPhase || (st.running?(st.active?'tracking':'watching'):'off');
 const prediction=st.predictionPhase || (st.active?(st.activeDryRun?'simulation':'awaiting_result'):'none');
 const twitch=st.twitchPhase || (st.connected?'connected':'disconnected');
 let detail=t('overview.detail.'+phase);
 if(prediction==='window_estimate'){
   const minutes=Math.max(1,Math.ceil((st.predictionWindowSecondsLeft||0)/60));
   detail=t('overview.detail.window',{min:minutes});
 }
 return {phase,prediction,twitch,title:t('overview.state.'+phase),detail};
}
function renderSettingsDraft(){
  const status=$('settings-save-status');
  if(status){
    status.textContent=t(state.hasUnsaved?'minimal.settings.unsaved':'minimal.settings.saved');
    status.classList.toggle('dirty',state.hasUnsaved);
  }
  const warning=$('settings-real-warning');
  if(warning){
    const checked=$('realMode').checked, enabled=!!state.config?.realMode;
    warning.hidden=!checked&&!enabled;
    const key=enabled?(checked?'minimal.settings.realActive':'minimal.settings.realStopPending'):'minimal.settings.realPending';
    warning.textContent=t(key);
    const row=$('realMode').closest?.('.settings-real-mode');
    row?.classList.toggle('active',checked||enabled);
  }
}

function setActionBusy(name, busy){
  if(typeof document==='undefined'||!document.querySelectorAll)return;
  for(const button of document.querySelectorAll('[data-action]')){
    if(button.dataset.action==='quit')continue;
    button.disabled=!!busy;
    // Mark just the activated control as pending; other disabled controls are not loading.
    if(button.dataset.action===name && button.classList?.contains('btn')){
      if(busy)button.setAttribute('aria-busy','true');
      else button.removeAttribute('aria-busy');
    }
  }
}

function render(s){
  if(s)state.status=s;
  const st=state.status||{connected:false,running:false,active:false,realMode:false,logs:[]};
  const view=monitoringView(st);
  const running=!!st.running,connected=view.twitch==='connected';
  const name=st.channel||'';
  const needsSetup=!!state.config && !(state.config.playerName||state.config.playerToonHandle);
  const title=needsSetup&&!running?t('minimal.setup.title'):view.title;
  const description=needsSetup&&!running?t('minimal.setup.detail'):view.detail;
  // Avoid repeating live-region announcements on every 2.5-second status poll.
  if($('overview-state-title').textContent!==title)$('overview-state-title').textContent=title;
  if($('overview-state-detail').textContent!==description)$('overview-state-detail').textContent=description;
  $('overview-beacon').classList.toggle('on',running);
  $('overview-beacon').classList.toggle('tracking',view.phase==='tracking');
  $('top-connection').classList.toggle('connected',connected);
  $('top-connection').classList.toggle('connecting',view.twitch==='authorizing'||view.twitch==='unverified');
  const twitchLabel=t('overview.twitch.'+view.twitch);
  $('top-connected-text').textContent=connected?t('top.connected'):twitchLabel;
  $('overview-twitch-dot').classList.toggle('good',connected);
  $('overview-twitch').textContent=connected&&name?'@'+name:twitchLabel;
  $('overview-prediction').textContent=t('overview.pred.'+view.prediction);
  const replay=state.replays[0];
  $('overview-last-map').textContent=replay?(replay.map||replay.file||'—'):t('minimal.noMatch');
  const outcome=$('overview-last-outcome');
  outcome.hidden=!replay;
  if(replay){
    const result=replay.result||'unknown';
    outcome.className='outcome-pill '+(result==='win'?'win':result==='loss'?'loss':'unknown');
    outcome.textContent=t(result==='win'?'badge.win':result==='loss'?'badge.loss':'badge.unknown');
  }
  const events=Array.isArray(st.logs)?st.logs:[];
  const event=events.length?events[events.length-1]:null;
  $('overview-latest-event').textContent=event?t('minimal.lastEvent',{message:String(event.message||'').slice(0,145)}):t('minimal.noEvents');
  const real=!!st.realMode;
  const safety=$('overview-safety');
  safety.classList.toggle('is-real',real);
  $('automation-mode').textContent=t(real?'minimal.realMode':'minimal.simMode');
  $('automation-mode-detail').textContent=t(real?'minimal.realDetail':'minimal.simDetail');
  const warning=$('overview-alert');
  const authIssue=real&&running&&!connected;
  const alertText=authIssue?t('minimal.twitchWarning'):t('minimal.liveWarning');
  const shouldWarn=real||authIssue;
  if(shouldWarn&&warning.textContent!==alertText)warning.textContent=alertText;
  warning.hidden=!shouldWarn;
  $('start-label').textContent=needsSetup&&!running?t('minimal.configure'):t(running?'start.stop':'start.start');
  const mainButton=document.querySelector('.overview-toggle');
  mainButton.dataset.action=needsSetup&&!running?'wizard':'toggle';
  mainButton.classList.toggle('is-stop',running);
  $('twitch-info').textContent=connected?t('twitch.info.on',{name}):view.twitch==='disconnected'?t('twitch.info.off'):twitchLabel;
  $('settings-connect-label').textContent=t(connected?'minimal.settings.reconnect':'twitch.connect');
  renderSettingsDraft();
  if(st.authCode){
    $('auth-progress').hidden=false;
    $('auth-code').textContent=st.authCode;
    $('auth-info').textContent=st.authStatus||t('auth.info.default');
  }else if(st.authPending){
    $('auth-progress').hidden=false;
    $('auth-info').textContent=t('auth.info.pending');
  }else $('auth-progress').hidden=true;
  showLogs(st.logs);
  if(!$('wizard').hidden)renderWizard();
}
async function loadStatus(){try{render(await api('status'))}catch(e){/* app closed, keep last UI */}}
async function reloadReplays(){try{const x=await api('replays');showReplays(x.replays||[])}catch(e){toast(e.message,true)}}
async function save(over={}){
 const data={...getInputs(),...over};
 if(data.realMode&&!state.config?.realMode){if(!confirm(t('confirm.real'))){data.realMode=false;$('realMode').checked=false}}
 const x=await api('config','POST',{...data,confirmedRealMode:data.realMode});
 state.config=x.config;state.hasUnsaved=false;lang=x.config.lang||lang;applyI18n();renderSettingsDraft();toast(t('toast.saved'));await loadStatus();return x;
}
async function action(name){if(state.busy&&name!=='quit')return;state.busy=true;setActionBusy(name,true);try{switch(name){
 case 'save':await save();break;
 case 'auth':{
   if(!$('clientId').value.trim()){
     go('settings');
     $('settings-advanced').open=true;
     $('clientId').focus();
     toast(t('minimal.settings.clientRequired'),true);
     break;
   }
   if(state.hasUnsaved)await save();
   const x=await api('auth','POST',{});
   if(x.code){
     go('settings');
     $('auth-progress').hidden=false;
     $('auth-code').textContent=x.code;
     code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';
     toast(t('toast.auth_code'));
   }
   break;
 }
 case 'activate':window.open(code.authUrl,'_blank','noopener,noreferrer');break;
  case 'copy-code':await copyTwitchCode(false);break;
 case 'developer':window.open('https://dev.twitch.tv/console/apps','_blank','noopener,noreferrer');break;
 case 'replay':{
    const result=await api('replay/check','POST',{});
    if(result.selected)toast(t('toast.replay_ok'));
    else{toast(t('toast.replay_no_player',{err:result.playerError||t('replay.no_selected')}),true);go('replays');}
    await reloadReplays();
    const first=document.querySelector('#full-replays .match-head');
    if(first&&first.getAttribute('aria-expanded')!=='true')first.click();
    break;
  }
 case 'start':if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast(t('toast.started'));await loadStatus();break;
 case 'stop':await api('automation/stop','POST',{});toast(t('toast.stopping'));await loadStatus();break;
 case 'toggle':if(state.status?.running){await api('automation/stop','POST',{});toast(t('toast.stopping'))}else{if(state.hasUnsaved)await save();await api('automation/start','POST',{});toast(t('toast.started'))}await loadStatus();break;
 case 'refresh-replays':await reloadReplays();toast(t('toast.refreshed'));break;
 case 'wizard':openWizard();break;
 case 'check-root':await checkReplayRoot();break;
  case 'reset-root':
    $('replayRoot').value='';state.hasUnsaved=true;renderSettingsDraft();
    await checkReplayRoot();
    break;
  case 'quit':if(!confirm(t('confirm.quit')))break;await api('quit','POST',{});window.close();document.body.innerHTML='<div style="font:18px Segoe UI;color:#ccdaee;margin:70px;text-align:center">'+safe(t('quit.message'))+'</div>';break;
 }}catch(e){toast(e.message||t('toast.request_error'),true)}finally{state.busy=false;setActionBusy(name,false)}}

async function checkReplayRoot(){
  const box=$('root-check-result');box.hidden=false;box.replaceChildren(line(t('root.checking'),'empty-mini'));
  try{
    const x=await api('replayroot/check','POST',{path:$('replayRoot').value.trim()});
    box.replaceChildren();
    for(const en of (x.entries||[])){
      const row=document.createElement('button');row.type='button';
      row.className='root-entry'+(en.exists?' ok':' bad');
      const mark=Object.assign(document.createElement('b'),{textContent:en.exists?'✓':'✗'});
      const info=Object.assign(document.createElement('span'),{className:'root-path',textContent:en.path});
      const cnt=Object.assign(document.createElement('small'),{textContent:en.exists?t('root.found',{n:en.replays}):t('root.none')});
      row.title=t('root.use');
      row.append(mark,info,cnt);
      row.addEventListener('click',()=>{$('replayRoot').value=en.path;state.hasUnsaved=true;renderSettingsDraft();});
      box.append(row);
    }
  }catch(e){box.replaceChildren(line(e.message,'empty-mini'));}
}

window.addEventListener('DOMContentLoaded',async()=>{
 document.querySelectorAll('[data-view]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.view)));
 $('nav').addEventListener('keydown',handleTabKeyDown);
 document.querySelectorAll('[data-goto]').forEach(el=>el.addEventListener('click',()=>go(el.dataset.goto)));
 document.querySelectorAll('[data-action]').forEach(el=>el.addEventListener('click',()=>action(el.dataset.action)));
 document.querySelectorAll('[data-wiz]').forEach(el=>{
   if(!['next','back','skip'].includes(el.dataset.wiz))
     el.addEventListener('click',()=>wizardAction(el.dataset.wiz));
 });
 document.querySelectorAll('[data-wiz-lang]').forEach(el=>el.addEventListener('click',()=>setLang(el.dataset.wizLang)));
 document.querySelector('[data-wiz="next"]').addEventListener('click',wizNext);
 document.querySelector('[data-wiz="back"]').addEventListener('click',()=>{if(!wiz.busy&&wiz.step>0){wiz.step--;renderWizard();}});
 document.querySelector('[data-wiz="skip"]').addEventListener('click',wizardSkip);
 for (const id of ['clientId','playerName','toonHandle','replayRoot','realMode','predictionWindow','predTitle','predWin','predLoss']){
    $(id).addEventListener('input',()=>{
      state.hasUnsaved=true;renderSettingsDraft();
      if(['predictionWindow','predTitle','predWin','predLoss'].includes(id))renderPredictionPreviews();
    });
    $(id).addEventListener('change',()=>{
      state.hasUnsaved=true;renderSettingsDraft();
      if(['predictionWindow','predTitle','predWin','predLoss'].includes(id))renderPredictionPreviews();
    });
  }
 $('lang').addEventListener('change',()=>setLang($('lang').value));
 for(const id of ['history-search','history-filter'])$(id).addEventListener(id==='history-search'?'input':'change',renderHistory);
 for(const id of ['log-search','log-filter'])$(id).addEventListener(id==='log-search'?'input':'change',()=>showLogs(state.logs));
  $('wz-player').addEventListener('input',renderPlayerPickers);
  $('playerName').addEventListener('input',renderPlayerPickers);
  $('toonHandle').addEventListener('input',renderPlayerPickers);
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
