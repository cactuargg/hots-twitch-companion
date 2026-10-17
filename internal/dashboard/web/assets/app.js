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
}

function setActionBusy(name, busy){
  if(typeof document==='undefined'||!document.querySelectorAll)return;
  for(const button of document.querySelectorAll('[data-action]')){
    if(button.dataset.action==='quit')continue;
    button.disabled=!!busy;
    // Mark just the activated control as pending; other disabled controls are not loading.
    if(!busy)button.removeAttribute('aria-busy');
    else if(button.dataset.action===name&&button.classList?.contains('btn'))
      button.setAttribute('aria-busy','true');
  }
}

function render(s){
 if(state.closing)return;
 if(s?.version)$('app-version').textContent='v'+s.version;
  if(s)state.status=s;
  const st=state.status||{connected:false,running:false,active:false,realMode:false,logs:[]};
  // The badge opens the release page in the system browser; no in-app update.
  const upd=$('update-badge');
  if(upd){
   const label=st.updateVersion&&st.updateURL?t('update.available',{version:st.updateVersion}):'';
   upd.hidden=!label;
   if(label&&upd.textContent!==label)upd.textContent=label;
  }
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
  // Manual controls remain available even when monitoring is stopped.
  const manual=$('prediction-manual');
  manual.hidden=!st.active;
  $('prediction-sync').hidden=!st.active||!!st.activeDryRun;
  $('prediction-manual-description').textContent=st.pendingOutcome?
    t('manual.pending',{outcome:t('manual.'+st.pendingOutcome)}):t(st.activeDryRun?'manual.description.sim':'manual.description.real');
  const scan=st.replayScan||{};
  const stale=!!st.running&&!!st.active&&Number(scan.checkedAt)>0&&
    Date.now()-Number(scan.checkedAt)>20000;
  const blocked=String(scan.code||'').startsWith('correlation_')||scan.code==='settle_error'||scan.code==='create_error';
  const reason=st.creationUnconfirmed?'create_error':blocked?scan.code:st.running?(stale?'stalled':(scan.code||'checking')):'paused';
  const reasonKey='replay.scan.'+reason;
  const scanText=t(reasonKey);
  const detail=scan.detail?String(scan.detail).slice(0,170):'';
  const filename=scan.file?String(scan.file).slice(0,110):'';
  const display=[scanText,filename,detail].filter(Boolean).join(' · ');
  if($('prediction-scan-message').textContent!==display)
    $('prediction-scan-message').textContent=display;
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
  const real=true;
  const safety=$('overview-safety');
  safety.classList.toggle('is-real',real);
  $('automation-mode').textContent='Twitch Predictions';
  $('automation-mode-detail').textContent=t('minimal.realDetail');
  const warning=$('overview-alert');
  const authIssue=real&&running&&!connected;
  // Persisted uncertain POSTs take priority over generic live notices.
  const alertText=state.connectionLost?t('connection.lost'):st.creationUnconfirmed?t('minimal.creationUnconfirmed'):
    authIssue?t('minimal.twitchWarning'):t('minimal.liveWarning');
  const shouldWarn=!!st.creationUnconfirmed||real||authIssue;
  if(shouldWarn&&warning.textContent!==alertText)warning.textContent=alertText;
  warning.hidden=!shouldWarn;
  $('reconcile-creation').hidden=!st.creationUnconfirmed;
  $('start-label').textContent=needsSetup&&!running?t('minimal.configure'):t(running?'start.stop':'start.start');
  const mainButton=document.querySelector('.overview-toggle');
  mainButton.dataset.action=needsSetup&&!running?'wizard':'toggle';
  mainButton.classList.toggle('is-stop',running);
  $('twitch-info').textContent=connected?t('twitch.info.on',{name}):view.twitch==='disconnected'?t('twitch.info.off'):twitchLabel;
  $('settings-connect-label').textContent=t(connected?'minimal.settings.reconnect':'twitch.connect');
  const clientHintKey=st.hasTwitchApp?'minimal.settings.clientOverride':'onboarding.twitch.steps';
  $('twitch-client-hint').dataset.i18n=clientHintKey;
  $('twitch-client-hint').textContent=t(clientHintKey);
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
async function loadStatus(){
 if(state.closing||state.statusPending)return;
 state.statusPending=true;
 try{const status=await api('status');if(state.closing)return;state.connectionLost=false;render(status)}
 catch(e){if(state.closing)return;state.connectionLost=true;render(state.status)}
 finally{state.statusPending=false}
}
async function reloadReplays(){if(state.closing)return;try{const x=await api('replays');if(!state.closing)showReplays(x.replays||[])}catch(e){if(!state.closing)toast(e.message,true)}}
function stopPolling(){
 state.closing=true;
 clearInterval(state.statusTimer);clearInterval(state.replaysTimer);
 for(const controller of state.requestControllers||[])controller.abort();
}
async function save(over={}){
 if(state.saving||state.languageSaving)throw Error(t('settings.wait'));
 const draft=JSON.stringify(getInputs());
 const data={...getInputs(),...over};
 data.realMode=true;
 state.saving=true;
 try{
  const x=await api('config','POST',{...data,confirmedRealMode:data.realMode});
  const changed=JSON.stringify(getInputs())!==draft;
  state.config=x.config;state.hasUnsaved=changed;lang=x.config.lang||lang;
  applyI18n();renderSettingsDraft();toast(t(changed?'settings.draftRemaining':'toast.saved'));
  await loadStatus();return x;
 }finally{state.saving=false}
}
async function startAutomation(){
 if(state.connectionLost){toast(t('connection.lost'),true);return false}
 if(state.hasUnsaved){toast(t('settings.draftRemaining'),true);return false}
 if(state.status?.creationUnconfirmed){toast(t('minimal.creationUnconfirmed'),true);return false}
 if(!state.status?.connected){go('settings');toast(t('onboarding.simple.needTwitch'),true);return false}
 if(!confirm(t('confirm.real')))return false;
 await api('automation/start','POST',{confirmedRealMode:true});
 return true;
}
async function manualPrediction(outcome){
 if(state.connectionLost)throw Error(t('connection.lost'));
 const st=state.status||{};
 if(!st.active||!st.activePredictionId||!st.activePredictionStartedAt)
   throw Error(t('manual.stale'));
 if(st.pendingOutcome&&outcome!=='sync'&&outcome!==st.pendingOutcome)
   throw Error(t('manual.pending',{outcome:t('manual.'+st.pendingOutcome)}));
 const isReal=!st.activeDryRun;
 if(outcome!=='sync'&&isReal){
   const prompt=t('manual.confirm.'+outcome);
   if(!confirm(prompt))return false;
 }
 await api('prediction/manual','POST',{
   outcome,id:st.activePredictionId,startedAt:st.activePredictionStartedAt
 });
 toast(t(outcome==='sync'?'manual.success.sync':'manual.success.updated'));
 await loadStatus();
 return true;
}
async function action(name){if(state.connectionLost&&name!=='quit'){toast(t('connection.lost'),true);return}if(state.busy&&name!=='quit')return;state.busy=true;setActionBusy(name,true);try{switch(name){
 case 'reset-profile':
   if(!confirm(t('reset.confirm')))break;
   await api('profile/reset','POST',{confirmed:true});
   stopPolling();
   try{localStorage.clear()}catch(e){}
   document.body.replaceChildren(line(t('reset.done'),'empty-mini'));
   break;
 case 'save':await save();break;
 case 'auth':{
   if(!$('clientId').value.trim()&&!state.status?.hasTwitchApp){
     go('settings');
     $('settings-advanced').open=true;
     $('clientId').focus();
     toast(t('minimal.settings.clientRequired'),true);
     break;
   }
   if(state.hasUnsaved)await save();
   if(state.hasUnsaved)break;
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
 case 'activate':openExternalLink(code.authUrl);break;
  case 'copy-code':await copyTwitchCode(false);break;
 case 'developer':openExternalLink('https://dev.twitch.tv/console/apps');break;
 case 'replay':{
    const result=await api('replay/check','POST',{});
    if(result.selected)toast(t('toast.replay_ok'));
    else{toast(t('toast.replay_no_player',{err:result.playerError||t('replay.no_selected')}),true);go('replays');}
    await reloadReplays();
    const first=document.querySelector('#full-replays .match-head');
    if(first&&first.getAttribute('aria-expanded')!=='true')first.click();
    break;
  }
 case 'start':if(state.hasUnsaved)await save();if(await startAutomation()){toast(t('toast.started'));await loadStatus()}break;
 case 'stop':await api('automation/stop','POST',{});toast(t('toast.stopping'));await loadStatus();break;
 case 'toggle':if(state.status?.running){await api('automation/stop','POST',{});toast(t('toast.stopping'))}else{if(state.hasUnsaved)await save();if(await startAutomation())toast(t('toast.started'))}await loadStatus();break;
 case 'recheck-result':{
   const st=state.status||{};
   if(!st.active||!st.activePredictionId||!st.activePredictionStartedAt)
     throw Error(t('manual.stale'));
   if(!st.activeDryRun&&!confirm(t('replay.scan.confirmReal')))break;
   await api('prediction/recheck','POST',{
     id:st.activePredictionId,startedAt:st.activePredictionStartedAt
   });
   await loadStatus();
   toast(t('replay.scan.rechecked'));
   break;
 }
 case 'reconcile-creation':
   if(!state.status?.creationUnconfirmed)break;
   if(!confirm(t('minimal.creationConfirm')))break;
   await api('prediction/acknowledge-unconfirmed','POST',{confirmed:true});
   toast(t('minimal.creationResolved'));
   await loadStatus();
   break;
 case 'manual-win':await manualPrediction('win');break;
 case 'manual-loss':await manualPrediction('loss');break;
 case 'manual-cancel':await manualPrediction('cancel');break;
 case 'manual-sync':await manualPrediction('sync');break;
 case 'refresh-replays':await reloadReplays();toast(t('toast.refreshed'));break;
 case 'wizard':openWizard();break;
 case 'check-root':await checkReplayRoot();break;
  case 'reset-root':
    $('replayRoot').value='';state.hasUnsaved=true;renderSettingsDraft();
    await checkReplayRoot();
    break;
  case 'update':if(state.status?.updateURL)openExternalLink(state.status.updateURL);break;
  case 'quit':if(!confirm(t('confirm.quit')))break;await api('quit','POST',{});stopPolling();window.close();document.body.innerHTML='<div style="font:18px Segoe UI;color:#ccdaee;margin:70px;text-align:center">'+safe(t('quit.message'))+'</div>';break;
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
 initDesktopShell();
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
 for (const id of ['clientId','playerName','toonHandle','replayRoot','predictionWindow','predTitle','predWin','predLoss']){
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
   const b=await api('bootstrap');if(state.closing)return;state.config=b.config;setInputs(b.config);
   if(b.config.lang){lang=b.config.lang;try{localStorage.setItem('hots-lang',lang)}catch(e){}applyI18n();}
   render(b.status);showReplays(b.replays||[]);
   if(!b.config.wizardDone)openWizard();
 }catch(e){toast(t('toast.connect_fail',{err:e.message}),true)}
 if(!state.closing){state.statusTimer=setInterval(loadStatus,2500);state.replaysTimer=setInterval(reloadReplays,30000);}
});
window.addEventListener('pagehide',stopPolling);
