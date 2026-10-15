'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.join(__dirname, '..');
const source = fs.readFileSync(path.join(root, 'web/assets/app.js'), 'utf8');
const html = fs.readFileSync(path.join(root, 'web/index.html'), 'utf8');

function section(start, end) {
  const from = source.indexOf(start);
  const to = source.indexOf(end, from + start.length);
  assert.ok(from >= 0 && to > from, 'Missing JS function: ' + start);
  return source.slice(from, to);
}

function makeHelpers(values = {}) {
  const elements = new Map();
  for (const [id, value] of Object.entries(values)) {
    elements.set(id, { value, textContent: '' });
  }
  const context = {
    $: id => {
      if (!elements.has(id)) elements.set(id, { value: '', textContent: '' });
      return elements.get(id);
    },
    t: (key, vars) => {
      const labels = {
        'prediction.title': 'HotS win?',
        'prediction.win': 'Win',
        'prediction.loss': 'Loss',
        'onboarding.preview.duration': 'Betting window: {min} min'
      };
      const label = labels[key] ?? key;
      return vars ? label.replace('{min}', String(vars.min)) : label;
    }
  };
  vm.createContext(context);
  const functions = section('function latestReplayPlayers(', 'function renderPlayerPickers(') +
    '\n' + section('function renderPredictionPreviews(', 'async function copyTwitchCode(');
  vm.runInContext(functions, context);
  return { context, elements };
}

test('player selector uses latest readable replay and deduplicates identities', () => {
  const { context } = makeHelpers();
  const selected = context.latestReplayPlayers([
    { error: 'corrupt', players: [{ name: 'Ignore', toonHandle: 'a' }] },
    { map: 'Garden of Terror', players: [
      { name: 'Me', toonHandle: '2-Hero-1-11' },
      { name: 'Me', toonHandle: '2-Hero-1-11' },
      { name: 'Other', toonHandle: '2-Hero-1-22' },
      { name: '  ' },
    ]},
    { map: 'Old Map', players: [{ name: 'Older', toonHandle: 'other' }] }
  ]);
  assert.equal(selected.map, 'Garden of Terror');
  assert.deepEqual(Array.from(selected.players, p => p.name), ['Me', 'Other']);
  assert.equal(selected.players[0].toonHandle, '2-Hero-1-11');
});

test('empty or invalid replays keep manual name entry available', () => {
  const { context } = makeHelpers();
  assert.equal(context.latestReplayPlayers([]).players.length, 0);
  assert.equal(context.latestReplayPlayers([{ players: [] }]).players.length, 0);
});

test('Prediction preview uses localized defaults without any Twitch request', () => {
  const { context, elements } = makeHelpers({
    predTitle: '', predWin: '', predLoss: '', predictionWindow: '2'
  });
  context.renderPredictionPreviews();
  for (const prefix of ['preview-', 'wz-preview-']) {
    assert.equal(elements.get(prefix + 'title').textContent, 'HotS win?');
    assert.equal(elements.get(prefix + 'win').textContent, 'Win');
    assert.equal(elements.get(prefix + 'loss').textContent, 'Loss');
    assert.equal(elements.get(prefix + 'duration').textContent, 'Betting window: 2 min');
  }
  elements.get('predTitle').value = 'My custom prediction';
  elements.get('predWin').value = 'Blue';
  elements.get('predLoss').value = 'Red';
  elements.get('predictionWindow').value = '15';
  context.renderPredictionPreviews();
  assert.equal(elements.get('preview-title').textContent, 'My custom prediction');
  assert.equal(elements.get('wz-preview-win').textContent, 'Blue');
  assert.equal(elements.get('wz-preview-loss').textContent, 'Red');
  assert.equal(elements.get('preview-duration').textContent, 'Betting window: 15 min');
});

test('both screens contain picker, preview, and OAuth copy controls', () => {
  for (const id of [
    'settings-player-picker', 'wizard-player-picker',
    'preview-title', 'preview-win', 'preview-loss', 'preview-duration',
    'wz-preview-title', 'wz-preview-win', 'wz-preview-loss', 'wz-preview-duration',
    'wz-twitch-status'
  ]) {
    assert.ok(html.includes('id="' + id + '"'), 'Missing UI element: ' + id);
  }
  assert.ok(html.includes('data-action="copy-code"'));
  assert.ok(html.includes('data-wiz="copy-code"'));
});
