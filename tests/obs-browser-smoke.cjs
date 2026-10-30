const {chromium}=require('playwright');
const fs=require('node:fs'),path=require('node:path'),http=require('node:http'),assert=require('node:assert/strict');
const root=path.resolve('internal/dashboard/web');
fs.mkdirSync('.research',{recursive:true});
const prefs={enabled:true,rankedOnly:false,style:'nexus',scale:100,backdrop:true,winColor:'#72dcff',lossColor:'#ff91b2',labels:true,maskEnabled:true,maskMode:'auto',maskRankedOnly:true};
let journal={preferences:prefs,current:{id:'s1',player:'p',startedAt:Date.now()-3600000},sessions:[],matches:[],player:'p',wins:7,losses:3,phase:'unknown',scoreURL:'http://127.0.0.1:18765/demo/score',maskURL:'http://127.0.0.1:18765/demo/mask'};
journal.sessions=[journal.current];
journal.matches=[{id:'a',player:'p',sessionId:'s1',map:'Hanamura',hero:'Jaina',mode:50091,result:'win',startedAt:Date.now()-1800000},{id:'b',player:'p',map:'Towers of Doom',hero:'Valla',mode:50091,result:'loss',needsAssignment:true,recordedAt:Date.now()-100000}];
let gameReplays=[];
const status={running:false,connected:false,active:false,logs:[],updates:{enabled:false,phase:'idle'}};
const config={playerName:'Streamer',playerToonHandle:'p',wizardDone:true,lang:'ru',disableUpdateChecks:true,predictionWindowMin:2};
const server=http.createServer(async(req,res)=>{
 res.setHeader('Content-Security-Policy', "default-src 'self'; img-src 'self' data:; script-src 'self'; style-src 'self'; connect-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'");
 const name=new URL(req.url,'http://local').pathname.replace(/^\/demo\//,'');
 if(name.startsWith('api/')){let body='';for await(const b of req)body+=b;let data=body?JSON.parse(body):{};let result={};switch(name){case 'api/bootstrap':result={status,config,replays:gameReplays};break;case 'api/config':Object.assign(config,data);result={config};break;case 'api/status':result=status;break;case 'api/replays':result={replays:gameReplays};break;case 'api/sessions':result=journal;break;case 'api/sessions/preferences':journal.preferences=data.preferences;result=journal;break;case 'api/sessions/new':journal.current={id:'s2',player:'p',startedAt:Date.now()};journal.sessions.push(journal.current);journal.wins=0;journal.losses=0;result=journal;break;case 'api/sessions/match':{const m=journal.matches.find(m=>m.id===data.matchId);Object.assign(m,data);if(data.sessionId)m.needsAssignment=false;result=journal;break}}
 res.setHeader('Content-Type','application/json');res.end(JSON.stringify(result));return}
 const file=path.join(root,name||'index.html');if(!file.startsWith(root)||!fs.existsSync(file)){res.statusCode=404;res.end();return}res.setHeader('Content-Type',file.endsWith('.js')?'text/javascript':file.endsWith('.css')?'text/css':file.endsWith('.svg')?'image/svg+xml':'text/html');res.end(fs.readFileSync(file))
});
(async()=>{await new Promise(r=>server.listen(18766,'127.0.0.1',r));const browser=await chromium.launch({executablePath:'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',headless:true});try{
 const page=await browser.newPage({viewport:{width:1000,height:850}});const errors=[];page.on('pageerror',e=>{errors.push(e.message);console.error('PAGE ERROR',e.stack)});await page.goto('http://127.0.0.1:18766/demo/');await page.waitForSelector('#session-score');await page.waitForFunction(()=>document.querySelector('#session-score').textContent==='7–3');
 await page.locator('[data-view=settings]').click();assert.equal(await page.locator('.settings-nav').count(),0);
 await page.locator('.wide-predictioncopy #predictionWindow').selectOption('5');
 await page.locator('[data-proxy=obs-style] input[value=minimal]').check();await page.locator('#predTitle').fill('Победим?');await page.locator('#predWin').fill('Да');await page.locator('#predLoss').fill('Нет');await page.locator('[data-wide-save]').click();
 await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled&&document.querySelector('#settings-save-status').textContent==='Все изменения сохранены');assert.equal(journal.preferences.style,'minimal');assert.equal(config.predictionTitle,'Победим?');assert.equal(config.predictionWindowMin,5);assert.equal(config.predictionWin,'Да');assert.equal(config.predictionLoss,'Нет');
 await page.locator('#obs-custom-css').evaluate(el=>el.closest('details').open=true);
 await page.locator('#obs-custom-css').fill('body { background: rgb(12, 13, 14); } #wins { color: rgb(221, 170, 255) !important; }');
 await page.waitForFunction(()=>{const d=document.querySelector('#obs-css-preview').contentDocument;return d?.getElementById('wins')&&getComputedStyle(d.getElementById('wins')).color==='rgb(221, 170, 255)'});
 assert.notEqual(await page.locator('body').evaluate(el=>getComputedStyle(el).backgroundColor),'rgb(12, 13, 14)','custom body CSS must remain isolated');
 await page.locator('[data-wide-save]').click();
 await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled&&document.querySelector('#settings-save-status').textContent==='Все изменения сохранены');
 assert.ok(journal.preferences.customCSS.includes('#wins'));
 await page.reload();await page.waitForSelector('#session-score');await page.locator('[data-view=settings]').click();
 await page.waitForFunction(()=>document.querySelector('#obs-custom-css').value.includes('#wins'));
 await page.locator('#obs-custom-css').evaluate(el=>el.closest('details').open=true);
 await page.locator('#obs-custom-css').fill('');await page.locator('[data-wide-save]').click();
 await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled&&document.querySelector('#obs-css-preview').hidden);
 assert.equal(journal.preferences.customCSS,'');
 await page.locator('#obs-theme').selectOption('builtin:neon');
 await page.waitForFunction(()=>{const d=document.querySelector('#obs-css-preview').contentDocument;return d?.getElementById('wins')&&getComputedStyle(d.getElementById('wins')).color==='rgb(88, 255, 220)'});
 await page.locator('#obs-theme-name').fill('Моя тема');await page.locator('#obs-theme-save').click();
 assert.equal(journal.preferences.scoreThemes?.length||0,0,'library must remain a draft until Save changes');
 await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(journal.preferences.scoreThemes[0].name,'Моя тема');
 await page.reload();await page.waitForSelector('#session-score');await page.locator('[data-view=settings]').click();
 await page.waitForFunction(()=>document.querySelector('#obs-theme').value==='saved:0');
 await page.locator('#obs-custom-css').evaluate(el=>el.closest('details').open=true);
 await page.locator('#obs-custom-css').fill('#wins { color: gold !important; }');
 await page.locator('#obs-theme-update').click();
 await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(journal.preferences.scoreThemes[0].css,'#wins { color: gold !important; }');
 await page.locator('#obs-theme').selectOption('builtin:ice');await page.locator('#obs-theme').selectOption('saved:0');
 assert.equal(await page.locator('#obs-custom-css').inputValue(),'#wins { color: gold !important; }');
 await page.locator('#obs-theme-delete').click();
 assert.equal(await page.locator('#obs-custom-css').inputValue(),'#wins { color: gold !important; }','deleting a library entry must preserve the active appearance');
 await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(journal.preferences.scoreThemes.length,0);
 await page.locator('#obs-theme').selectOption('builtin:default');
 await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(journal.preferences.customCSS,'');
 await page.screenshot({path:'.research/obs-settings.png',fullPage:true});
 await page.locator('[data-view=dashboard]').click();await page.locator('[data-proxy=session-mask-mode] input[value=hide]').check();await page.waitForFunction(()=>document.querySelector('#session-mask-mode').value==='hide');assert.equal(journal.preferences.maskMode,'hide');
 assert.equal(await page.locator('#wide-matches .wide-match').count(),2);await page.locator('.wide-match summary').first().click();await page.locator('.wide-match[open] .wide-matchbody .session-match select').first().selectOption('s1');await page.waitForFunction(()=>document.querySelectorAll('#wide-matches .wide-match').length===2);assert.equal(journal.matches[1].needsAssignment,false);
 await page.screenshot({path:'.research/obs-history.png',fullPage:true});
 for(const width of [1100,736,480]){await page.setViewportSize({width,height:800});await page.locator('[data-view=settings]').click();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'settings overflow '+width);await page.locator('[data-view=dashboard]').click();assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'overview overflow '+width)}await page.screenshot({path:'.research/obs-mobile.png',fullPage:true});
 await page.setViewportSize({width:1100,height:900});
 await page.locator('[data-view=dashboard]').click();await page.locator('.wide-archive>summary').click();await page.locator('.wide-archive').screenshot({path:'.research/panels-sessions.png'});
 await page.locator('[data-view=settings]').click();await page.evaluate(()=>document.querySelectorAll('#view-settings details').forEach(d=>d.open=true));await page.screenshot({path:'.research/panels-expanded.png',fullPage:true});
 await page.evaluate(()=>document.querySelectorAll('.toast').forEach(el=>el.hidden=true));
 const segmentHeights=await page.locator('[data-proxy=obs-ranked] label').evaluateAll(labels=>labels.map(el=>el.getBoundingClientRect().height));assert.ok(segmentHeights.every(h=>h<=44),'expanded mode segments must remain compact');
 await page.locator('.wide-settinggroup').filter({has:page.locator('#obs-enabled')}).screenshot({path:'.research/obs-expanded-detail.png'});
 for(const width of [736,480]){await page.setViewportSize({width,height:900});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'expanded settings overflow '+width)}
 await page.screenshot({path:'.research/panels-expanded-mobile.png',fullPage:true});

 // Game profiles: migrate current identity, select a new player from a replay,
 // keep shared preferences, survive reload and guard unsafe switches.
 await page.setViewportSize({width:1100,height:900});
 await page.locator('[data-view=settings]').click();
 assert.equal(await page.locator('#game-profile-select option').count(),1);
 const sharedPreferences=JSON.stringify(journal.preferences),oldHistory=JSON.stringify(journal.matches);
 await page.locator('#game-profile-add').click();
 await page.locator('#game-profile-name').fill('Второй аккаунт');
 gameReplays=[{file:'alt.StormReplay',map:'Alt map',players:[{name:'Alt',toonHandle:'2-Hero-1-900'}]}];
 await page.evaluate(()=>reloadReplays());
 await page.locator('#settings-player-choose').evaluate(el=>el.open=true);
 await page.locator('#settings-player-picker .player-choice').click();
 await page.locator('#replayRoot').evaluate(el=>{el.value='E:/AltReplays';el.dispatchEvent(new Event('input',{bubbles:true}))});
 await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(config.gameProfiles.length,2);assert.equal(config.playerToonHandle,'2-Hero-1-900');
 const altID=config.activeGameProfile;
 assert.equal(config.gameProfiles[0].playerToonHandle,'p');
 assert.equal(config.gameProfiles.find(p=>p.id===altID).replayRoot,'E:/AltReplays');
 assert.equal(JSON.stringify(journal.preferences),sharedPreferences);
 assert.equal(JSON.stringify(journal.matches),oldHistory);
 await page.reload();await page.waitForSelector('#session-score');
 await page.waitForFunction(id=>document.querySelector('#game-profile-active').value===id,altID);
 await page.locator('#game-profile-active').selectOption('default');
 await page.waitForFunction(()=>document.querySelector('#game-profile-active').value==='default'&&!document.querySelector('#game-profile-active').disabled);
 assert.equal(config.playerToonHandle,'p');assert.equal(config.replayRoot,'');
 await page.locator('#game-profile-active').selectOption(altID);
 await page.waitForFunction(id=>document.querySelector('#game-profile-active').value===id&&!document.querySelector('#game-profile-active').disabled,altID);
 assert.equal(config.playerToonHandle,'2-Hero-1-900');assert.equal(config.replayRoot,'E:/AltReplays');
 for(const flag of ['running','active','creationUnconfirmed']){
  status[flag]=true;await page.evaluate(()=>loadStatus());assert.equal(await page.locator('#game-profile-active').isDisabled(),true);
  await page.locator('[data-view=settings]').click();assert.equal(await page.locator('#game-profile-select').isDisabled(),true);
  status[flag]=false;await page.evaluate(()=>loadStatus());await page.locator('[data-view=dashboard]').click();
 }
 await page.locator('[data-view=settings]').click();await page.locator('#game-profile-name').fill('Переименованный');
 await page.locator('[data-view=dashboard]').click();assert.equal(await page.locator('#game-profile-active').isDisabled(),true,'unsaved settings must block quick switch');
 await page.locator('[data-view=settings]').click();await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(config.gameProfiles.find(p=>p.id===altID).name,'Переименованный');
 await page.locator('#game-profile-delete').click();await page.locator('[data-wide-save]').click();await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled);
 assert.equal(config.gameProfiles.length,1);assert.equal(config.playerToonHandle,'p');assert.equal(JSON.stringify(journal.matches),oldHistory);
 await page.locator('.game-profiles').filter({has:page.locator('#game-profile-select')}).screenshot({path:'.research/game-profiles-settings.png'});
 for(const width of [1100,736,480]){await page.setViewportSize({width,height:900});assert.equal(await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth),false,'profiles overflow '+width)}
 assert.deepEqual(errors,[]);console.log('PASS: UI settings, persistence requests, manual mask, session assignment, 480px layout; screenshots in .research')
}finally{await browser.close();server.close()}})().catch(e=>{console.error(e);server.close();process.exitCode=1});
