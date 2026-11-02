import test from 'node:test';
import assert from 'node:assert/strict';
import { APIClient, APIError, CreationIntent } from './api.ts';

test('authenticated requests use current session and preserve version/idempotency headers', async () => {
  let access = 'first';
  const seen: RequestInit[] = [];
  const client = new APIClient('https://api.example/', () => access, async (url, init) => {
    assert.equal(url, 'https://api.example/v1/plans'); seen.push(init!);
    return new Response(JSON.stringify({ id: 'created' }), { status: 201 });
  });
  await client.request('/plans', { method: 'POST', body: { title: 'Cà phê' }, key: 'same-intent' });
  access = 'refreshed';
  await client.request('/plans', { method: 'PATCH', body: { title: 'Cà phê' }, version: 3 });
  assert.equal(new Headers(seen[0].headers).get('Authorization'), 'Bearer first');
  assert.equal(new Headers(seen[0].headers).get('Idempotency-Key'), 'same-intent');
  assert.equal(new Headers(seen[1].headers).get('Authorization'), 'Bearer refreshed');
  assert.equal(new Headers(seen[1].headers).get('If-Match'), '"3"');
});

test('missing session fails locally; server errors never expose raw body', async () => {
  const missing = new APIClient('https://api.example', () => null, async () => { throw new Error('must not call'); });
  await assert.rejects(missing.request('/me'), (e: unknown) => e instanceof APIError && e.status === 401);
  const denied = new APIClient('https://api.example', () => 'access', async () => new Response(JSON.stringify({ code: 'forbidden', error: 'SQL secret' }), { status: 403 }));
  await assert.rejects(denied.request('/me'), (e: unknown) => e instanceof APIError && !e.message.includes('SQL') && e.status === 403);
});

test('creation intent keeps retry key only for same actor and payload; clear rotates after success', () => {
  let count = 0; const intent = new CreationIntent(() => `key-${++count}`);
  assert.equal(intent.key('actor-a', { title: 'Coffee' }), 'key-1');
  assert.equal(intent.key('actor-a', { title: 'Coffee' }), 'key-1');
  assert.equal(intent.key('actor-a', { title: 'Dinner' }), 'key-2');
  assert.equal(intent.key('actor-b', { title: 'Dinner' }), 'key-3');
  intent.clear(); assert.equal(intent.key('actor-b', { title: 'Dinner' }), 'key-4');
});

test('malformed success and an already-aborted request fail safely', async () => {
  const malformed = new APIClient('https://api.example', () => 'access', async () => new Response('bad', { status: 200 }));
  await assert.rejects(malformed.request('/me'), (e: unknown) => e instanceof APIError && e.code === 'invalid_response');
  const controller = new AbortController(); controller.abort();
  const client = new APIClient('https://api.example', () => 'access', async (_, init) => { assert.equal(init?.signal?.aborted, true); throw new Error('abort'); });
  await assert.rejects(client.request('/me', { signal: controller.signal }), (e: unknown) => e instanceof APIError && e.code === 'cancelled');
});

test('session invalidation rejects an already-authorized delayed response', async () => {
  let release: (r: Response) => void = () => {};
  const client = new APIClient('https://api.example', () => 'access', async () => new Promise(resolve => { release = resolve; }));
  const pending = client.request('/plans/private');
  client.invalidate();
  release(new Response(JSON.stringify({ location: 'private' })));
  await assert.rejects(pending, (e: unknown) => e instanceof APIError && e.code === 'cancelled');
});
