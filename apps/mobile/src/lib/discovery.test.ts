import test from 'node:test';
import assert from 'node:assert/strict';
import { catalogueQuery, budgetLabel } from './discovery.ts';
test('catalogue filters encode area and enforce numeric bounds', () => {
  const url = new URL('https://api.example' + catalogueQuery({ activity: 'food', kind: 'idea', area: 'Q1 & kind=place', budget: '80000', people: '4' }, 'next'));
  assert.equal(url.searchParams.get('kind'), 'idea'); assert.equal(url.searchParams.get('area'), 'Q1 & kind=place'); assert.equal(url.searchParams.get('max_budget'), '80000'); assert.equal(url.searchParams.get('people'), '4'); assert.equal(url.searchParams.get('cursor'), 'next');
  for (const patch of [{ people: '0' }, { people: '1001' }, { budget: '-1' }, { budget: '1.5' }, { activity: 'fake' }]) assert.throws(() => catalogueQuery({ activity: '', kind: '', area: '', budget: '', people: '1', ...patch }));
});
test('unknown prices are never displayed as free', () => {
  assert.equal(budgetLabel(null, null), 'Chi phí chưa xác minh');
  assert.match(budgetLabel(40000, 80000), /40.*80/); assert.match(budgetLabel(0, 0), /Dự trù/);
});
