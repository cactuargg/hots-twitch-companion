function selectEvents(items,level,query){
  const needle=String(query||'').trim().toLocaleLowerCase();
  return (Array.isArray(items)?items:[]).filter(ev=>{
    if(level&&level!=='all'&&(ev.level||'info')!==level)return false;
    return !needle||String(ev.message||'').toLocaleLowerCase().includes(needle);
  });
}
function diagnosticLevel(entry){
  return ['info','warning','error'].includes(entry.level)?entry.level:'info';
}
function diagnosticHelp(entry){
  const known=['twitch_auth','token_store','prediction_create','prediction_settle','prediction_timeout',
    'replay_process','player_not_found','replay_check','local_server'];
  return known.includes(entry.code)?t('diag.help.'+entry.code):'';
}
function createLogRow(entry){
  const level=diagnosticLevel(entry);
  const row=document.createElement('div');row.className='log-row diagnostic-'+level;
  const time=Object.assign(document.createElement('span'),{className:'log-time',textContent:(entry.time||'').slice(0,8)});
  const ico=Object.assign(document.createElement('span'),{className:'log-icon',textContent:level==='error'?'!':level==='warning'?'△':'◆'});
  const content=document.createElement('div');content.className='log-content';
  content.append(line(entry.message||'','log-msg'));
  const help=diagnosticHelp(entry);
  if(help)content.append(line(help,'log-help'));
  row.append(time,ico,content);return row;
}
function showLogs(logs){
  state.logs=Array.isArray(logs)?logs:[];
  const full=$('full-log'),scroll=full.scrollTop;
  // The overview no longer duplicates the event log. Full diagnostics live in History.

  const events=selectEvents(state.logs,$('log-filter').value,$('log-search').value);
  $('log-summary').textContent=t('logs.count',{shown:events.length,total:state.logs.length});
  full.replaceChildren();
  if(!events.length)full.append(line(state.logs.length?t('logs.filtered.empty'):t('logs.empty.full'),'empty-mini'));
  else for(const entry of events)full.append(createLogRow(entry));
  full.scrollTop=scroll;
}
