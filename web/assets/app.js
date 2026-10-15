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
function render(s){
 if(s)state.status=s;
 const st=state.status||{connected:false,running:false,active:false,realMode:false,logs:[]};
 const running=!!st.running, active=!!st.active, simulation=!st.realMode, name=st.channel||'';
 const view=monitoringView(st), connected=view.twitch==='connected';
 const twitchLabel=t('overview.twitch.'+view.twitch), predictionLabel=t('overview.pred.'+view.prediction);
 $('top-connection').classList.toggle('connected',connected);
 $('top-connection').classList.toggle('connecting',view.twitch==='authorizing'||view.twitch==='unverified');
 $('top-connected-text').textContent=connected?t('top.connected'):twitchLabel;
 // Keep the announced heading unchanged between polling updates unless the phase changes.
 if($('overview-state-title').textContent!==view.title)$('overview-state-title').textContent=view.title;
 $('overview-state-detail').textContent=view.detail;
 $('overview-player').textContent=st.playerName||st.playerToonHandle||t('dash.player.empty');
 $('overview-twitch').textContent=connected&&name?'@'+name:twitchLabel;
 $('overview-prediction').textContent=predictionLabel;
 $('overview-beacon').classList.toggle('on',running);
 $('overview-beacon').classList.toggle('tracking',view.phase==='tracking');
 $('overview-safety').classList.toggle('is-real',!simulation);
 $('dash-channel').textContent=connected?(name?'@'+name:t('dash.channel.authorized')):twitchLabel;
 $('channel-pill').textContent=twitchLabel;
 $('channel-pill').classList.toggle('online',connected);
 $('dash-player').textContent=st.playerName||st.playerToonHandle||t('dash.player.empty');
 $('dash-prediction').textContent=predictionLabel;
 $('dash-result').textContent=st.lastResult||t('dash.result.none');
 $('match-status').textContent=view.phase==='tracking'?t('match.status.tracking'):running?t('match.status.waiting'):t('match.status.off');
 $('match-description').textContent=view.phase==='tracking'?t('match.desc.tracking'):running?t('match.desc.waiting'):t('match.desc.off');
 $('match-mini').textContent=active?t('match.mini.active'):running?t('match.mini.running'):t('match.mini.waiting');
 $('prediction-state').textContent=predictionLabel;
 $('prediction-help').textContent=active?t('pred.help.active'):simulation?t('pred.help.sim'):t('pred.help.wait');
 $('watch-tag').classList.toggle('on',running);
 $('watch-tag').innerHTML='<span class="pulse-dot"></span> '+t(running?'watch.on':'watch.off');
 $('start-label').textContent=running?t('start.stop'):t('start.start');
 $('automation-mode').textContent=simulation?t('mode.sim'):t('mode.real');
 $('automation-mode-detail').textContent=simulation?t('mode.sim.detail'):t('mode.real.detail');
 $('footer-mode').textContent=simulation?t('footer.mode.sim'):t('footer.mode.real');
 $('footer-detail').textContent=simulation?t('footer.detail.sim'):t('footer.detail.real');
 $('footer-status').textContent=running?t('footer.status.on'):t('footer.status.off');
 $('twitch-info').textContent=connected?t('twitch.info.on',{name}):view.twitch==='disconnected'?t('twitch.info.off'):twitchLabel;
 if(st.authCode){$('auth-progress').hidden=false;$('auth-code').textContent=st.authCode;$('auth-info').textContent=st.authStatus||t('auth.info.default')}
 else if(st.authPending){$('auth-progress').hidden=false;$('auth-info').textContent=t('auth.info.pending')}
 else{$('auth-progress').hidden=true}
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
    $('replayRoot').value='';state.hasUnsaved=true;
    await checkReplayRoot();
    break;
  case 'quit':if(!confirm(t('confirm.quit')))break;await api('quit','POST',{});window.close();document.body.innerHTML='<div style="font:18px Segoe UI;color:#ccdaee;margin:70px;text-align:center">'+safe(t('quit.message'))+'</div>';break;
 }}catch(e){toast(e.message||t('toast.request_error'),true)}finally{state.busy=false}}

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
      row.addEventListener('click',()=>{$('replayRoot').value=en.path;state.hasUnsaved=true;});
      box.append(row);
    }
  }catch(e){box.replaceChildren(line(e.message,'empty-mini'));}
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
 for (const id of ['clientId','playerName','toonHandle','replayRoot','realMode','predictionWindow','predTitle','predWin','predLoss']){
    $(id).addEventListener('input',()=>{
      state.hasUnsaved=true;
      if(['predictionWindow','predTitle','predWin','predLoss'].includes(id))renderPredictionPreviews();
    });
    $(id).addEventListener('change',()=>{
      state.hasUnsaved=true;
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
