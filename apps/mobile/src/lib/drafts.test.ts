import test from 'node:test';
import assert from 'node:assert/strict';
import { DraftStore } from './drafts.ts';
const draft = { title: 'Cà phê', activity: 'coffee', date: '2030-01-01', time: '19:30', capacity: '', location: '', group_id: '' };
function memory() { const data = new Map<string, string>(); return { data, async getItem(k: string) { return data.get(k) ?? null; }, async setItem(k: string, v: string) { data.set(k, v); }, async removeItem(k: string) { data.delete(k); } }; }
test('draft and prepared retry key survive controller restart, isolated by actor', async () => {
  const storage = memory(); let n = 0;
  const first = new DraftStore(storage, () => `key-${++n}`, () => 1000);
  const key = await first.prepare('alice', draft, { title: 'Cà phê' });
  const restarted = new DraftStore(storage, () => `key-${++n}`, () => 2000);
  assert.deepEqual((await restarted.load('alice')).draft, draft);
  assert.equal(await restarted.prepare('alice', draft, { title: 'Cà phê' }), key);
  assert.equal((await restarted.load('bob')).draft.title, '');
  assert.notEqual(await restarted.prepare('bob', draft, { title: 'Cà phê' }), key);
});
test('edits are serialized and changed payload rotates intent; success only clears its own attempt', async () => {
  const storage = memory(); let n = 0; const store = new DraftStore(storage, () => `key-${++n}`, () => 1000);
  const old = await store.prepare('alice', draft, { title: 'Cà phê' });
  await Promise.all([store.saveDraft('alice', { ...draft, title: 'A' }), store.saveDraft('alice', { ...draft, title: 'B' })]);
  assert.equal((await store.load('alice')).draft.title, 'B');
  const current = await store.prepare('alice', { ...draft, title: 'B' }, { title: 'B' });
  assert.notEqual(current, old); await store.complete('alice', old);
  assert.equal((await store.load('alice')).draft.title, 'B');
  await store.complete('alice', current); assert.equal((await store.load('alice')).draft.title, '');
});
test('storage failure prevents obtaining an unpersisted key; old or corrupted attempts do not silently retry', async () => {
  const bad = { async getItem() { return null; }, async setItem() { throw new Error('disk'); }, async removeItem() {} };
  await assert.rejects(new DraftStore(bad, () => 'key').prepare('alice', draft, {}));
  const storage = memory(); let now = 1000; const store = new DraftStore(storage, () => 'key', () => now);
  await store.prepare('alice', draft, {}); now += 24 * 3600000;
  await assert.rejects(store.prepare('alice', draft, {}), /kiểm tra danh sách/);
  storage.data.set('no-plans.plan-draft.alice', '{bad');
  await assert.rejects(store.load('alice'), /bản nháp/);
});
test('completion preserves newer edits even before they are prepared for submission', async () => {
  const store = new DraftStore(memory(), () => 'key');
  const key = await store.prepare('alice', draft, { title: draft.title });
  await store.saveDraft('alice', { ...draft, title: 'Bản nháp mới' });
  await store.complete('alice', key);
  const retained = await store.load('alice');
  assert.equal(retained.draft.title, 'Bản nháp mới');
  assert.equal(retained.pending, undefined);
});
test('retry key survives equivalent payload property order changes across app upgrades', async () => {
  let n = 0; const storage = memory();
  const first = new DraftStore(storage, () => `key-${++n}`);
  const key = await first.prepare('alice', draft, { title: draft.title, capacity: 0 });
  // The previously shipped version stored insertion-order JSON rather than canonical JSON.
  const legacy = JSON.parse(storage.data.get('no-plans.plan-draft.alice')!);
  legacy.pending.signature = JSON.stringify({ title: draft.title, capacity: 0 });
  storage.data.set('no-plans.plan-draft.alice', JSON.stringify(legacy));
  const restarted = new DraftStore(storage, () => `key-${++n}`);
  assert.equal(await restarted.prepare('alice', draft, { capacity: 0, title: draft.title }), key);
});
