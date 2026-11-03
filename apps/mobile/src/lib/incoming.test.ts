import test from 'node:test';
import assert from 'node:assert/strict';
import { parseIncoming, IncomingInvite } from './incoming.ts';
const token = '00000000-0000-4000-8000-000000000001.' + 'a'.repeat(43);
const second = '00000000-0000-4000-8000-000000000002.' + 'b'.repeat(43);
function storage() { let value: string | null = null; return { async getItem() { return value; }, async setItem(_key: string, next: string) { value = next; }, async removeItem() { value = null; } }; }
test('native and configured HTTPS links accept fragment invites only', () => {
  for (const url of ['no-plans://invite#token=' + token, 'no-plans:///invite#token=' + token, '/invite#token=' + token]) assert.deepEqual(parseIncoming(url), { type: 'invite', token, invalid: false });
  assert.deepEqual(parseIncoming('https://invite.example/app#token=' + token, 'https://invite.example/app'), { type: 'invite', token, invalid: false });
  assert.equal(parseIncoming('https://evil.example/app#token=' + token, 'https://invite.example/app').type, 'invalid');
  assert.deepEqual(parseIncoming('no-plans://invite?token=' + token), { type: 'invite', token: null, invalid: true });
  assert.deepEqual(parseIncoming('/invite#token=bad'), { type: 'invite', token: null, invalid: true });
});
test('private routes accept UUID targets without trusting arbitrary redirect parameters', () => {
  assert.deepEqual(parseIncoming('no-plans://plans/00000000-0000-4000-8000-000000000001?next=https://evil.example'), { type: 'route', path: '/plans/00000000-0000-4000-8000-000000000001' });
  for (const path of ['javascript:alert(1)', '//evil.example', 'no-plans://plans/not-an-id', 'no-plans://unknown', 'invalid']) assert.equal(parseIncoming(path).type, 'invalid');
});
test('Expo Go development launch URLs preserve home and invitation entry points', () => {
  assert.deepEqual(parseIncoming('exp://192.168.1.10:8081/--/', '', true), { type: 'route', path: '/' });
  assert.deepEqual(parseIncoming('exp://192.168.1.10:8081/--/invite#token=' + token, '', true), { type: 'invite', token, invalid: false });
  assert.equal(parseIncoming('exp://192.168.1.10:8081/--/').type, 'invalid');
});
test('pending invite restores and malformed incoming link clears older valid invite', async () => {
  const disk = storage(); const first = new IncomingInvite(disk);
  await first.receive({ token, invalid: false });
  const restarted = new IncomingInvite(disk); await restarted.load(); assert.equal(restarted.getSnapshot().token, token);
  await restarted.receive({ token: null, invalid: true });
  const final = new IncomingInvite(disk); await final.load(); assert.equal(final.getSnapshot().token, null); assert.equal(final.getSnapshot().invalid, true);
});
test('old redemption cannot erase a newer incoming invite', async () => {
  const store = new IncomingInvite(storage()); await store.receive({ token, invalid: false }); await store.receive({ token: second, invalid: false });
  assert.equal(await store.consume(token), false); assert.equal(store.getSnapshot().token, second);
  assert.equal(await store.consume(second), true); assert.equal(store.getSnapshot().token, null);
});
test('late hydration does not overwrite a newly received link', async () => {
  let release: (value: string | null) => void = () => {};
  const disk = { async getItem() { return new Promise<string | null>(resolve => { release = resolve; }); }, async setItem() {}, async removeItem() {} };
  const store = new IncomingInvite(disk); const loading = store.load(); await store.receive({ token: second, invalid: false });
  release(JSON.stringify({ token, invalid: false })); await loading; assert.equal(store.getSnapshot().token, second);
});
test('storage-denied receive still works in memory; failed consume is recoverable', async () => {
  const store = new IncomingInvite({ async getItem() { throw new Error(); }, async setItem() { throw new Error(); }, async removeItem() { throw new Error(); } });
  await store.receive({ token, invalid: false }); assert.equal(store.getSnapshot().token, token);
  await assert.rejects(store.consume(token)); assert.equal(store.getSnapshot().token, token);
});
