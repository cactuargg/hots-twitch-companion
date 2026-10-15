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

/* ── setup wizard ────────────────────────────────────────────────────── */
const wiz={step:0,max:2};
function openWizard(){
  wiz.step=0;
  $('wz-player').value=$('playerName').value;
  $('wz-client').value=$('clientId').value;
  $('wz-test-result').hidden=true;$('wz-auth-box').hidden=true;
  renderPlayerPickers();
  renderPredictionPreviews();
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
  const twitch=st.twitchPhase||'disconnected';
  $('wz-twitch-status').textContent=t('onboarding.twitch.'+twitch,{name:st.channel||''});
  $('wz-twitch-status').classList.toggle('verified',twitch==='connected');
  if(st.authCode){
    $('wz-auth-box').hidden=false;
    $('wz-auth-code').textContent=st.authCode;
    $('wz-auth-info').textContent=st.authStatus||t('wiz.t.pending');
  }else if(twitch==='connected'){$('wz-auth-box').hidden=true;}
  renderPredictionPreviews();
}
async function wizNext(){
  if(wiz.step===0){$('playerName').value=$('wz-player').value.trim();try{await save();}catch(e){toast(e.message,true);return false;}}
  if(wiz.step===1){$('clientId').value=$('wz-client').value.trim();try{await save();}catch(e){toast(e.message,true);return false;}}
  if(wiz.step===wiz.max){try{await save({wizardDone:true});}catch(e){toast(e.message,true);return false;}closeWizard();return true;}
  wiz.step++;renderWizard();return true;
}
async function wizardAction(kind){
  try{
    if(kind==='test'){
      const res=$('wz-test-result');res.hidden=false;res.textContent='…';
      $('playerName').value=$('wz-player').value.trim();await save();
      const x=await api('replay/check','POST',{});
      if(!x.selected)throw Error(x.playerError||t('replay.no_selected'));
      res.className='wiz-result ok';
      res.textContent=t('wiz.p.ok',{res:x.selected.name+' → '+x.selected.result,map:x.map});
      await reloadReplays();
    }else if(kind==='console'){window.open('https://dev.twitch.tv/console/apps','_blank','noopener,noreferrer');}
    else if(kind==='auth'){
      $('clientId').value=$('wz-client').value.trim();await save();
      if(!$('clientId').value.trim()){toast(t('toast.need_client'),true);return;}
      const x=await api('auth','POST',{});
      if(x.code){const box=$('wz-auth-box');box.hidden=false;$('wz-auth-code').textContent=x.code;code.authUrl=x.verificationUri||'https://www.twitch.tv/activate';toast(t('toast.auth_code'));}
    }else if(kind==='activate'){window.open(code.authUrl,'_blank','noopener,noreferrer');}
    else if(kind==='copy-code'){await copyTwitchCode(true);}
    else if(kind==='start'){
      // Wizard simulation must never create real channel-point predictions.
      if(state.config?.realMode || $('realMode').checked){toast(t('wiz.sim_only'),true);return;}
      $('playerName').value=$('wz-player').value.trim();await save({realMode:false});
      await api('automation/start','POST',{});toast(t('toast.started'));await loadStatus();
    }
  }catch(e){
    if(kind==='test'){const res=$('wz-test-result');res.hidden=false;res.className='wiz-result err';res.textContent=t('wiz.p.fail',{err:e.message});}
    else toast(e.message||t('toast.request_error'),true);
  }
}
