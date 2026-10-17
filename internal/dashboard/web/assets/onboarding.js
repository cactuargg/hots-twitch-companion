/* ── helpers ─────────────────────────────────────────────────────────── */

function latestReplayPlayers(replays){
 const match=(Array.isArray(replays)?replays:[]).find(r=>!r.error&&Array.isArray(r.players)&&r.players.length);
 if(!match)return {map:'',players:[]};
 const seen=new Set();
 const players=match.players.filter(p=>{
   const name=String(p.name||'').trim(),handle=String(p.toonHandle||'').trim();
   if(!name)return false;
   const key=name+'|'+handle;
   if(seen.has(key))return false;
   seen.add(key);return true;
 });
 return {map:match.map||match.file||'',players};
}
function renderPlayerPickers(){
 const match=latestReplayPlayers(state.replays);
 for(const [id,wizard] of [['settings-player-picker',false],['wizard-player-picker',true]]){
   const box=$(id);
   if(!box)continue;
   box.replaceChildren();
   if(!match.players.length){box.append(line(t('onboarding.players.none'),'picker-empty'));continue;}
   box.append(line(t('onboarding.players.from',{map:match.map,n:match.players.length}),'picker-caption'));
   const list=document.createElement('div');list.className='player-picker-grid';
   for(const player of match.players){
     const button=document.createElement('button');button.type='button';button.className='player-choice';
     const name=document.createElement('strong');name.textContent=player.name;
     const handle=document.createElement('small');handle.textContent=player.toonHandle||'—';
     button.append(name,handle);
     const currentName=(wizard?$('wz-player'):$('playerName')).value.trim();
     const currentHandle=$('toonHandle').value.trim();
     const selected=currentName===player.name&&(!player.toonHandle||currentHandle===player.toonHandle);
     button.classList.toggle('selected',selected);
     button.setAttribute('aria-pressed',String(selected));
     button.addEventListener('click',()=>{
       if(wizard)$('wz-player').value=player.name;
       else $('playerName').value=player.name;
       $('toonHandle').value=player.toonHandle||'';
       state.hasUnsaved=true;renderPlayerPickers();renderSettingsDraft();
       toast(t('onboarding.players.selected',{name:player.name}));
     });
     list.append(button);
   }
   box.append(list);
 }
}
function renderPredictionPreviews(){
 const title=$('predTitle').value.trim()||t('prediction.title');
 const win=$('predWin').value.trim()||t('prediction.win');
 const loss=$('predLoss').value.trim()||t('prediction.loss');
 const min=Number.parseInt($('predictionWindow').value,10)||2;
 for(const prefix of ['preview-','wz-preview-']){
   $(prefix+'title').textContent=title;
   $(prefix+'win').textContent=win;
   $(prefix+'loss').textContent=loss;
   $(prefix+'duration').textContent=t('onboarding.preview.duration',{min});
 }
}
async function copyTwitchCode(wizard){
 const value=(wizard?$('wz-auth-code'):$('auth-code')).textContent.trim();
 if(!value||value==='—')return;
 try{await navigator.clipboard.writeText(value);toast(t('onboarding.copy.ok'));}
 catch(e){toast(t('onboarding.copy.fail'),true);}
}

/* ── Minimal first-run setup ─────────────────────────────────────────── */
const wiz={step:0,max:2,busy:false};

function openWizard(){
  wiz.step=0;
  $('wz-player').value=$('playerName').value;
  $('wz-client').value=$('clientId').value;
  $('wz-test-result').hidden=true;
  $('wz-auth-box').hidden=true;
  $('wz-twitch-details').open=false;
  $('wizard').hidden=false;
  renderPlayerPickers();
  renderPredictionPreviews();
  renderWizard();
}

function closeWizard(){
  $('wizard').hidden=true;
}

function renderWizard(){
  const steps=document.querySelectorAll('.wizard-body section');
  steps.forEach(sec=>sec.hidden=Number(sec.dataset.step)!==wiz.step);
  const dots=$('wiz-dots');dots.replaceChildren();
  for(let i=0;i<=wiz.max;i++){
    const dot=document.createElement('span');
    dot.className='dot'+(i===wiz.step?' on':'')+(i<wiz.step?' done':'');
    dots.append(dot);
  }
  const progress=t('onboarding.simple.progress',{step:wiz.step+1,total:wiz.max+1});
  if($('wiz-progress').textContent!==progress)$('wiz-progress').textContent=progress;
  document.querySelector('[data-wiz="back"]').disabled=wiz.step===0||wiz.busy;
  $('wiz-next').textContent=t(wiz.step===wiz.max?'onboarding.simple.start':'wiz.next');
  $('wiz-skip').textContent=t(wiz.step===wiz.max?'onboarding.simple.finishLater':'onboarding.simple.later');
  $('wiz-next').disabled=wiz.busy;
  $('wiz-skip').disabled=wiz.busy;
  const st=state.status||{};
  $('wz-ch-player').textContent=$('wz-player').value.trim()||t('wiz.pending');
  $('wz-ch-twitch').textContent=st.connected?
    (st.channel?'@'+st.channel:t('overview.twitch.connected')):
    t('onboarding.simple.twitchOptional');
  $('wz-ch-monitor').textContent=t('minimal.realDetail');
  // Client ID stays a developer-only fallback when the EXE has its own OAuth app.
  $('wz-twitch-details').hidden=!!st.hasTwitchApp&&!$('wz-client').value.trim();
  const twitch=st.twitchPhase||'disconnected';
  $('wz-twitch-status').textContent=t('onboarding.twitch.'+twitch,{name:st.channel||''});
  $('wz-twitch-status').classList.toggle('verified',twitch==='connected');
  if(st.authCode){
    $('wz-auth-box').hidden=false;
    $('wz-auth-code').textContent=st.authCode;
    $('wz-auth-info').textContent=st.authStatus||t('wiz.t.pending');
  }else if(twitch==='connected'){
    $('wz-auth-box').hidden=true;
  }
  const note=$('wz-safe-note');
  if(note){
    const liveRunning=!!st.running&&!!st.realMode;
    note.textContent=t(liveRunning?'onboarding.simple.liveRunning':'onboarding.simple.safe');
    note.classList.toggle('warning',liveRunning);
  }
  renderPredictionPreviews();
}

async function wizardSkip(){
  if(wiz.busy)return false;
  wiz.busy=true;renderWizard();
  try{
    // Mark the wizard dismissed without committing unrelated draft Settings fields.
    const cfg=state.config||getInputs();
    const x=await api('config','POST',{
      ...cfg,wizardDone:true,realMode:true,confirmedRealMode:true
    });
    state.config=x.config;
    closeWizard();
    render(state.status);
    return true;
  }catch(e){
    toast(e.message||t('toast.request_error'),true);
    return false;
  }finally{wiz.busy=false;if(!$('wizard').hidden)renderWizard();}
}

async function wizNext(){
  if(wiz.busy)return false;
  if(wiz.step===wiz.max)return wizardAction('start');
  wiz.busy=true;renderWizard();
  try{
    if(wiz.step===0){
      const name=$('wz-player').value.trim();
      if(!name){
        toast(t('onboarding.simple.needPlayer'),true);
        $('wz-player').focus();
        return false;
      }
      $('playerName').value=name;
      await save();
    }else if(wiz.step===1){
      if(!state.status?.connected){toast(t('onboarding.simple.needTwitch'),true);return false}
      const id=$('wz-client').value.trim();
      if(id!==$('clientId').value.trim()){
        $('clientId').value=id;
        await save();
      }
    }
    wiz.step++;
    renderWizard();
    return true;
  }catch(e){toast(e.message||t('toast.request_error'),true);return false;}
  finally{wiz.busy=false;renderWizard();}
}

async function wizardAction(kind){
  if(wiz.busy)return false;
  wiz.busy=true;renderWizard();
  try{
    if(kind==='test'){
      const res=$('wz-test-result');res.hidden=false;res.textContent='…';
      if(!$('wz-player').value.trim())throw Error(t('onboarding.simple.needPlayer'));
      $('playerName').value=$('wz-player').value.trim();
      await save();
      const x=await api('replay/check','POST',{});
      if(!x.selected)throw Error(x.playerError||t('replay.no_selected'));
      res.className='wiz-result ok';
      res.textContent=t('wiz.p.ok',{res:x.selected.name+' → '+t('badge.'+x.selected.result),map:x.map});
      await reloadReplays();
    }else if(kind==='console'){
      openExternalLink('https://dev.twitch.tv/console/apps');
    }else if(kind==='auth'){
      const id=$('wz-client').value.trim();
      if(!id&&!state.status?.hasTwitchApp){
        $('wz-twitch-details').open=true;
        $('wz-client').focus();
        toast(t('minimal.settings.clientRequired'),true);
        return false;
      }
      if(id!==$('clientId').value.trim()||state.hasUnsaved){
        $('clientId').value=id;
        await save();
      }
      const x=await api('auth','POST',{});
      if(x.code){
        $('wz-auth-box').hidden=false;
        $('wz-auth-code').textContent=x.code;
        code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';
        toast(t('toast.auth_code'));
      }
    }else if(kind==='activate'){
      openExternalLink(code.authUrl);
    }else if(kind==='copy-code'){
      await copyTwitchCode(true);
    }else if(kind==='start'){
      const name=$('wz-player').value.trim();
      if(!name){
        wiz.step=0;
        renderWizard();
        toast(t('onboarding.simple.needPlayer'),true);
        return false;
      }
      if(state.status?.running){
        toast(t('onboarding.simple.liveRunning'),true);
        return false;
      }
      $('playerName').value=name;
      if(!state.status?.connected){wiz.step=1;renderWizard();toast(t('onboarding.simple.needTwitch'),true);return false}
      await save();
      if(!await startAutomation())return false;
      const x=await api('config','POST',{
        ...state.config,wizardDone:true,realMode:true,confirmedRealMode:true
      });
      state.config=x.config;
      state.hasUnsaved=false;
      closeWizard();
      toast(t('toast.started'));
      await loadStatus();
      go('dashboard');
    }
    return true;
  }catch(e){
    if(kind==='test'){
      const res=$('wz-test-result');
      res.hidden=false;res.className='wiz-result err';
      res.textContent=t('wiz.p.fail',{err:e.message});
    }else toast(e.message||t('toast.request_error'),true);
    return false;
  }finally{wiz.busy=false;if(!$('wizard').hidden)renderWizard();}
}
