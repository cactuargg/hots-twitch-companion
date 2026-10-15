'use strict';
const test = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');

const html = fs.readFileSync(path.join(__dirname, '..', 'design', 'minimal-ui-prototype.html'), 'utf8');
const inline = html.match(/<script>([\s\S]*?)<\/script>/);
assert.ok(inline, 'Self-contained prototype JavaScript must exist');

function createPreview() {
  const elements = new Map();
  const $ = id => {
    if (!elements.has(id)) {
      elements.set(id, {
        textContent: '', className: '', hidden: false, value: '',
        addEventListener() {}, setAttribute() {}, focus() {}
      });
    }
    return elements.get(id);
  };
  const document = { getElementById: $, querySelectorAll: () => [] };
  const ctx = vm.createContext({ document });
  vm.runInContext(inline[1], ctx, { filename: 'minimal-ui-prototype.js' });
  return { elements, ctx, $ };
}

test('prototype has no external assets or game decorations', () => {
  assert.match(html, /<html lang="ru">/);
  assert.doesNotMatch(html, /<img\b|<svg\b|clip-path|box-shadow|<script\s+src=|<link\s+[^>]*href=/i);
  assert.doesNotMatch(html, /https?:\/\//);
  assert.match(html, /prefers-reduced-motion/);
  assert.match(html, /@media\s*\(max-width:\s*720px\)/);
});

test('all seven demonstration states render with visible information', () => {
  const { ctx, $ } = createPreview();
  const keys = Array.from(vm.runInContext('Object.keys(scenarios)', ctx));
  assert.deepEqual(keys, ['watching','off','tracking','awaiting','issue','setup','live']);
  for (const key of keys) {
    vm.runInContext('renderScenario(' + JSON.stringify(key) + ')', ctx);
    assert.ok($('hero-title').textContent, 'Missing status for ' + key);
    assert.ok($('hero-desc').textContent, 'Missing description for ' + key);
    assert.ok($('primary-action').textContent, 'Missing action for ' + key);
  }
  vm.runInContext("renderScenario('live')", ctx);
  assert.match($('mode-text').textContent, /Реальные баллы зрителей/);
  assert.equal($('hero-alert').hidden, false);
  vm.runInContext("renderScenario('issue')", ctx);
  assert.equal($('primary-action').textContent, 'Проверить настройки');
  vm.runInContext("renderScenario('watching')", ctx);
  assert.equal($('hero-alert').hidden, true);
});

test('main navigation is reduced to three accessible sections', () => {
  for(const name of ['dashboard','history','settings']) {
    assert.ok(html.includes('id="tab-' + name + '"'));
    assert.ok(html.includes('id="page-' + name + '"'));
    assert.ok(html.includes('aria-controls="page-' + name + '"'));
  }
  assert.equal((html.match(/role="tab"\s/g)||[]).length, 3);
  assert.match(html, /ArrowRight.*ArrowLeft.*Home.*End/s);
  assert.match(html, /<details><summary>Расширенные настройки<\/summary>/);
});
