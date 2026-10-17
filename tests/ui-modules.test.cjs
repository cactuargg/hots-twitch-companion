'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const root = path.join(__dirname, '..');
const assets = path.join(root, 'internal', 'dashboard', 'web', 'assets');
const html = fs.readFileSync(path.join(root, 'internal', 'dashboard', 'web', 'index.html'), 'utf8');
const scripts = ['core.js', 'i18n.js', 'replays.js', 'diagnostics.js', 'onboarding.js', 'desktop.js', 'app.js'];
const styles = ['style.css', 'wizard.css', 'overview.css', 'onboarding.css', 'diagnostics.css', 'settings.css', 'compact.css'];

test('HTML loads every local module in dependency order', () => {
  const loaded = [...html.matchAll(/<script defer src="\.\/assets\/([^"]+)"\s*><\/script>/g)].map(m => m[1]);
  assert.deepEqual(loaded, scripts);
  const css = [...html.matchAll(/<link rel="stylesheet" href="\.\/assets\/([^"]+)"\s*\/>/g)].map(m => m[1]);
  assert.deepEqual(css, styles);
  for (const name of [...scripts, ...styles]) {
    assert.ok(fs.statSync(path.join(assets, name)).size > 0, 'Missing/empty asset: ' + name);
  }
});

test('Classic scripts evaluate sequentially in one shared browser scope', () => {
  const registered = [];
  const sandbox = {
    URL,
    location: { href: 'http://127.0.0.1:12345/secret/' },
    localStorage: { getItem: () => 'en' },
    navigator: { language: 'en-US' },
    window: { addEventListener: (type, fn) => registered.push({ type, fn }) }
  };
  vm.createContext(sandbox);
  for (const file of scripts) {
    vm.runInContext(fs.readFileSync(path.join(assets, file), 'utf8'), sandbox, { filename: file });
  }
  assert.deepEqual(registered.map(e => e.type), ['DOMContentLoaded']);
  assert.equal(typeof registered[0].fn, 'function');
  assert.equal(vm.runInContext('typeof renderPlayerPickers', sandbox), 'function');
  assert.equal(vm.runInContext('typeof renderHistory', sandbox), 'function');
  assert.equal(vm.runInContext('typeof showLogs', sandbox), 'function');
  assert.equal(vm.runInContext("t('nav.sections')", sandbox), 'Application sections');
});

test('Keyboard tab navigation updates focus, selected state and visible panel', () => {
  const selected = {};
  const tabs = ['dashboard','replays','settings'].map(view => {
    const tab = {
      dataset: { view },
      tabIndex: view === 'dashboard' ? 0 : -1,
      attrs: {},
      classList: { toggle: (name, on) => { if(name==='active')selected[view] = on; } },
      setAttribute: (name, val) => { tab.attrs[name] = val; },
      focus: () => { selected.focus = view; }
    };
    return tab;
  });
  const panels = ['dashboard','replays','settings'].map(view => ({
    id: 'view-' + view,
    classList: { toggle: (name, on) => { if(name==='active')selected['panel-'+view] = on; } }
  }));
  const sandbox = { document: { querySelectorAll: selector =>
    selector === '.tab' ? tabs : selector === '.view' ? panels : []
  }};
  vm.createContext(sandbox);
  const source = fs.readFileSync(path.join(assets, 'core.js'), 'utf8');
  const from = source.indexOf('function go('), to = source.indexOf('function line(', from);
  assert.ok(from > 0 && to > from);
  vm.runInContext(source.slice(from, to), sandbox);
  sandbox.go('settings');
  assert.equal(tabs[2].attrs['aria-selected'], 'true');
  assert.equal(tabs[2].tabIndex, 0);
  assert.equal(tabs[0].tabIndex, -1);
  assert.equal(selected['panel-settings'], true);
  let prevented = false;
  sandbox.handleTabKeyDown({ key:'ArrowRight', target: tabs[2], preventDefault:()=>{prevented=true;} });
  assert.equal(prevented, true);
  assert.equal(selected.focus, 'dashboard');
  assert.equal(tabs[0].attrs['aria-selected'], 'true');
  sandbox.handleTabKeyDown({ key:'Home', target: tabs[0], preventDefault:()=>{} });
  assert.equal(selected.focus, 'dashboard');
  sandbox.handleTabKeyDown({ key:'ArrowLeft', target: tabs[0], preventDefault:()=>{} });
  assert.equal(selected.focus, 'settings');
});

test('Tabs link to matching labeled panels', () => {
  assert.match(html, /<nav[^>]*role="tablist"/);
  for (const view of ['dashboard','replays','settings']) {
    assert.ok(html.includes('id="tab-' + view + '" data-view="' + view + '" role="tab"'));
    assert.ok(html.includes('aria-controls="view-' + view + '"'));
    assert.ok(html.includes('id="view-' + view + '" role="tabpanel" aria-labelledby="tab-' + view + '"'));
  }
  assert.ok(html.includes('role="status" aria-live="polite"'));
});
