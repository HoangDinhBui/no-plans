import test from 'node:test';
import assert from 'node:assert/strict';
import { inviteURL, linkStatus, inviteLimit } from './invites.ts';
const token = '00000000-0000-4000-8000-000000000001.' + 'a'.repeat(43);
test('share link transports token in fragment only and rejects unsafe config/token', () => {
  const url = new URL(inviteURL('https://invite.example/app', token));
  assert.equal(url.search, ''); assert.equal(url.hash, '#token=' + token); assert.equal(url.pathname, '/app');
  for (const base of ['', 'javascript:alert(1)', 'https://user:password@invite.example', 'https://invite.example/?q=x', 'https://invite.example/#old', 'http://invite.example']) assert.throws(() => inviteURL(base, token));
  assert.throws(() => inviteURL('https://invite.example', 'bad token'));
  assert.equal(new URL(inviteURL('http://localhost:5173', token, true)).hostname, 'localhost');
});
test('link lifecycle exposes exhausted expired and revoked states', () => {
  const link = { id: 'id', plan_id: 'plan', expires_at: '2030-01-01T00:00:00Z', max_uses: 2, use_count: 1 };
  const now = Date.parse('2029-01-01T00:00:00Z');
  assert.equal(linkStatus(link, now), 'active');
  assert.equal(linkStatus({ ...link, use_count: 2 }, now), 'full');
  assert.equal(linkStatus(link, Date.parse(link.expires_at)), 'expired');
  assert.equal(linkStatus({ ...link, revoked_at: '2028-01-01T00:00:00Z' }, now), 'revoked');
});
test('invite usage limit accepts whole bounded values only', () => {
  assert.equal(inviteLimit(' 20 '), 20);
  for (const value of ['', '0', '1001', '-1', '1.5', 'abc']) assert.throws(() => inviteLimit(value));
});
