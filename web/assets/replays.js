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
  if(r.error){
    box.append(line(t('match.error')+': '+r.error,'empty-mini'));
    box.append(line(t('history.parseError'),'match-diagnostic'));
    return box;
  }
  if(r.selectionError){
    const warning=document.createElement('div');warning.className='match-diagnostic';
    warning.append(line(t('history.playerUnknown')));
    warning.append(line(r.selectionError,'match-diagnostic-detail'));
    const settings=document.createElement('button');settings.type='button';settings.className='text-action';
    settings.textContent=t('history.openSettings');
    settings.addEventListener('click',()=>go('settings'));
    warning.append(settings);box.append(warning);
  }
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

function selectReplays(items,query,outcome){
  const needle=String(query||'').trim().toLocaleLowerCase();
  return (Array.isArray(items)?items:[]).filter(r=>{
    if(outcome&&outcome!=='all'&&r.result!==outcome)return false;
    if(!needle)return true;
    return [r.map,r.file,r.selected,...(r.players||[]).map(p=>p.name)].join(' ').toLocaleLowerCase().includes(needle);
  });
}
function renderHistory(){
  const full=$('full-replays'),items=selectReplays(state.replays,$('history-search').value,$('history-filter').value);
  full.replaceChildren();
  $('history-summary').textContent=t('history.count',{shown:items.length,total:state.replays.length});
  if(!items.length){
    full.append(line(state.replays.length?t('history.filtered.empty'):t('replays.empty.full'),'empty-mini'));
    return;
  }
  items.forEach((r,i)=>{
    const key=(r.file||'')+'|'+(r.modified||''),opened=state.expandedReplays.has(key);
    const item=document.createElement('div');item.className='match-item'+(opened?' open':'');
    const head=document.createElement('button');head.type='button';head.className='match-head';
    const detailId='replay-detail-'+i;
    head.setAttribute('aria-expanded',String(opened));head.setAttribute('aria-controls',detailId);
    const title=document.createElement('span');title.className='match-title';title.textContent=r.map||r.file;
    const sub=document.createElement('small');sub.textContent=(r.file||'')+' · '+relative(r.modified);
    head.append(title,sub,resultBadge(r.result),Object.assign(document.createElement('span'),{className:'chev'}));
    head.querySelector('.chev').innerHTML='<svg><use href="#i-chev"/></svg>';
    const detail=matchDetail(r);detail.id=detailId;detail.hidden=!opened;
    head.addEventListener('click',()=>{
      const open=!state.expandedReplays.has(key);
      if(open)state.expandedReplays.add(key);else state.expandedReplays.delete(key);
      item.classList.toggle('open',open);detail.hidden=!open;
      head.setAttribute('aria-expanded',String(open));
    });
    item.append(head,detail);full.append(item);
  });
}
function showReplays(items){
  state.replays=Array.isArray(items)?items:[];
  const keys=new Set(state.replays.map(r=>(r.file||'')+'|'+(r.modified||'')));
  for(const key of state.expandedReplays)if(!keys.has(key))state.expandedReplays.delete(key);
  renderHistory();
  renderPlayerPickers();
  // A new replay can arrive between status polls. Update the compact summary immediately.
  if(state.status)render(state.status);
}
