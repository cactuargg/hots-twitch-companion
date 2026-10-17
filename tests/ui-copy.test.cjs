'use strict';
const test=require('node:test'),assert=require('node:assert/strict');
const fs=require('node:fs'),path=require('node:path'),vm=require('node:vm');
const dir='internal/dashboard/web';
const translations=fs.readFileSync(path.join(dir,'assets/i18n.js'),'utf8');
const ctx=vm.createContext({localStorage:{getItem:()=>null},navigator:{language:'en'}});
vm.runInContext(translations+';globalThis.catalog=I18N',ctx);
const catalog=ctx.catalog;
test('both UI languages cover every label and direct message with matching placeholders',()=>{
 assert.deepEqual(Object.keys(catalog.ru).sort(),Object.keys(catalog.en).sort());
 const files=fs.readdirSync(path.join(dir,'assets')).filter(x=>x.endsWith('.js')).map(x=>path.join(dir,'assets',x));
 const html=fs.readFileSync(path.join(dir,'index.html'),'utf8');
 const keys=[...html.matchAll(/data-i18n(?:-placeholder|-title|-alt|-aria-label)?="([^"]+)"/g)].map(x=>x[1]);
 for(const file of files)for(const match of fs.readFileSync(file,'utf8').matchAll(/\bt\(['"]([^'"]+)['"]\s*[,)]/g))keys.push(match[1]);
 for(const lang of ['en','ru'])for(const key of keys)assert.ok(catalog[lang][key],`${lang}: missing ${key}`);
 for(const key of Object.keys(catalog.en)){
  const tokens=value=>[...value.matchAll(/\{(\w+)\}/g)].map(x=>x[1]).sort();
  assert.deepEqual(tokens(catalog.en[key]),tokens(catalog.ru[key]),key);
  for(const lang of ['en','ru'])assert.ok(!catalog[lang][key].includes('\\n'),`${lang}/${key}: literal newline escape in displayed text`);
 }
});
test('manual confirmations use real paragraphs and identify the selected player',()=>{
 for(const lang of ['en','ru'])for(const outcome of ['win','loss','cancel']){
  assert.ok(catalog[lang]['manual.confirm.'+outcome].includes('\n\n'));
 }
 assert.match(catalog.ru['onboarding.simple.playerHelp'],/одним ником.*ручное/);
 assert.match(catalog.en['onboarding.simple.playerHelp'],/nickname alone.*manual/);
});
test('paused monitoring with a saved Prediction explains recovery instead of showing a betting countdown',()=>{
 const source=fs.readFileSync(path.join(dir,'assets/app.js'),'utf8');
 const context=vm.createContext({t:key=>key});
 vm.runInContext(source.slice(0,source.indexOf('function renderSettingsDraft')),context);
 assert.equal(context.monitoringView({running:false,active:true,predictionPhase:'window_estimate'}).detail,'overview.detail.pausedPrediction');
 assert.equal(context.monitoringView({running:true,active:true,predictionPhase:'window_estimate'}).detail,'overview.detail.window');
});
