'use strict';
const test=require('node:test');
const assert=require('node:assert/strict');
const fs=require('node:fs');
const vm=require('node:vm');
const src=fs.readFileSync('internal/dashboard/web/assets/replays.js','utf8');
const logic=src.slice(src.indexOf('function isSelectedReplayPlayer('),src.indexOf('function playerRow('));
test('history highlights only the selected ToonHandle when names collide',()=>{
  const ctx=vm.createContext({});
  vm.runInContext(logic,ctx);
  const selected={selected:'Alex',selectedToonHandle:'1-S2-123-456'};
  assert.equal(ctx.isSelectedReplayPlayer(selected,{name:'Alex',toonHandle:'1-S2-123-456'}),true);
  assert.equal(ctx.isSelectedReplayPlayer(selected,{name:'Alex',toonHandle:'1-S2-999-888'}),false);
  assert.equal(ctx.isSelectedReplayPlayer({selected:'Alex'},{name:'Alex'}),true,'legacy replay summaries still render');
  assert.equal(ctx.isSelectedReplayPlayer({selected:'Alex',selectedToonHandle:'abc'},{name:'Alex'}),false,'unknown handle must not highlight another player');
});
