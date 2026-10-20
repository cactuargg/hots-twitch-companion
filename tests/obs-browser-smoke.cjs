const {chromium}=require('playwright');
const fs=require('node:fs'),path=require('node:path'),http=require('node:http'),assert=require('node:assert/strict');
const root=path.resolve('internal/dashboard/web');
fs.mkdirSync('.research',{recursive:true});
const prefs={enabled:true,rankedOnly:false,style:'nexus',scale:100,backdrop:true,winColor:'#72dcff',lossColor:'#ff91b2',labels:true,maskEnabled:true,maskMode:'auto',maskRankedOnly:true};
let journal={preferences:prefs,current:{id:'s1',player:'p',startedAt:Date.now()-3600000},sessions:[],matches:[],player:'p',wins:7,losses:3,phase:'unknown',scoreURL:'http://127.0.0.1:18765/demo/score',maskURL:'http://127.0.0.1:18765/demo/mask'};
journal.sessions=[journal.current];
journal.matches=[{id:'a',player:'p',sessionId:'s1',map:'Hanamura',hero:'Jaina',mode:50091,result:'win',startedAt:Date.now()-1800000},{id:'b',player:'p',map:'Towers of Doom',hero:'Valla',mode:50091,result:'loss',needsAssignment:true,recordedAt:Date.now()-100000}];
const status={running:false,connected:false,active:false,logs:[],updates:{enabled:false,phase:'idle'}};
const config={playerName:'Streamer',playerToonHandle:'p',wizardDone:true,lang:'ru',disableUpdateChecks:true,predictionWindowMin:2};
const server=http.createServer(async(req,res)=>{
 const name=new URL(req.url,'http://local').pathname.replace(/^\/demo\//,'');
 if(name.startsWith('api/')){let body='';for await(const b of req)body+=b;let data=body?JSON.parse(body):{};let result={};switch(name){case 'api/bootstrap':result={status,config,replays:[]};break;case 'api/config':Object.assign(config,data);result={config};break;case 'api/status':result=status;break;case 'api/replays':result={replays:[]};break;case 'api/sessions':result=journal;break;case 'api/sessions/preferences':journal.preferences=data.preferences;result=journal;break;case 'api/sessions/new':journal.current={id:'s2',player:'p',startedAt:Date.now()};journal.sessions.push(journal.current);journal.wins=0;journal.losses=0;result=journal;break;case 'api/sessions/match':{const m=journal.matches.find(m=>m.id===data.matchId);Object.assign(m,data);if(data.sessionId)m.needsAssignment=false;result=journal;break}}
 res.setHeader('Content-Type','application/json');res.end(JSON.stringify(result));return}
 const file=path.join(root,name||'index.html');if(!file.startsWith(root)||!fs.existsSync(file)){res.statusCode=404;res.end();return}res.setHeader('Content-Type',file.endsWith('.js')?'text/javascript':file.endsWith('.css')?'text/css':file.endsWith('.svg')?'image/svg+xml':'text/html');res.end(fs.readFileSync(file))
});
(async()=>{await new Promise(r=>server.listen(18766,'127.0.0.1',r));const browser=await chromium.launch({executablePath:'C:/Program Files (x86)/Microsoft/Edge/Application/msedge.exe',headless:true});try{
 const page=await browser.newPage({viewport:{width:1000,height:850}});const errors=[];page.on('pageerror',e=>{errors.push(e.message);console.error('PAGE ERROR',e.stack)});await page.goto('http://127.0.0.1:18766/demo/');await page.waitForSelector('#session-score');await page.waitForFunction(()=>document.querySelector('#session-score').textContent==='7–3');
 await page.locator('[data-view=settings]').click();assert.equal(await page.locator('.settings-nav').count(),0);
 await page.locator('.wide-predictioncopy #predictionWindow').selectOption('5');
 await page.locator('[data-proxy=obs-style] input[value=minimal]').check();await page.locator('#predTitle').fill('Победим?');await page.locator('#predWin').fill('Да');await page.locator('#predLoss').fill('Нет');await page.locator('[data-wide-save]').click();
 await page.waitForFunction(()=>!document.querySelector('[data-wide-save]').disabled&&document.querySelector('#settings-save-status').textContent==='Все изменения сохранены');assert.equal(journal.preferences.style,'minimal');assert.equal(config.predictionTitle,'Победим?');assert.equal(config.predictionWindowMin,5);assert.equal(config.predictionWin,'Да');assert.equal(config.predictionLoss,'Нет');
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
 assert.deepEqual(errors,[]);console.log('PASS: UI settings, persistence requests, manual mask, session assignment, 480px layout; screenshots in .research')
}finally{await browser.close();server.close()}})().catch(e=>{console.error(e);server.close();process.exitCode=1});
