import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
const actor = '00000000-0000-4000-8000-000000000002';
const owner = '00000000-0000-4000-8000-000000000001';
const planA = '00000000-0000-4000-8000-000000000010';
const planB = '00000000-0000-4000-8000-000000000011';
const tokenA = '00000000-0000-4000-8000-000000000020.' + 'a'.repeat(43);
const tokenB = '00000000-0000-4000-8000-000000000021.' + 'b'.repeat(43);
async function boundaries(page: Page) {
  const state = { members: new Set<string>(), previews: [] as string[], redeems: [] as string[], reads: [] as string[], hold: false, release: null as (() => void) | null, full: false };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const user = { id: actor, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: {}, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z' };
  const jwt = [Buffer.from('{"alg":"ES256"}').toString('base64url'), Buffer.from(JSON.stringify({ sub: actor, exp })).toString('base64url'), 'fixture'].join('.');
  const headers = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,PATCH,PUT,DELETE' };
  await page.route('https://auth.no-plans.test/**', async route => { if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; } await route.fulfill({ headers, json: { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user } }); });
  await page.route('https://api.no-plans.test/**', async route => {
    const req = route.request(); const path = new URL(req.url()).pathname;
    expect(req.url()).not.toContain(tokenA); expect(req.url()).not.toContain(tokenB);
    if (req.method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; }
    if (path === '/v1/invite-preview') {
      expect(req.headers().authorization).toBeUndefined(); const token = req.headers()['x-invite-token']; state.previews.push(token);
      await route.fulfill(state.full ? { status: 404, headers, json: { code: 'invite_unavailable' } } : { headers, json: { title: token === tokenB ? 'Lời mời B' : 'Lời mời A', activity: 'coffee', starts_at: '2030-01-01T12:30:00Z' } }); return;
    }
    expect(req.headers().authorization).toBe('Bearer ' + jwt);
    if (path === '/v1/invite-redeem') {
      const token = req.headers()['x-invite-token']; state.redeems.push(token);
      if (state.hold) { state.hold = false; await new Promise<void>(resolve => { state.release = resolve; }); }
      const id = token === tokenB ? planB : planA; state.members.add(id); await route.fulfill({ headers, json: { plan_id: id } }); return;
    }
    if (path === '/v1/plans/' + planA || path === '/v1/plans/' + planB) {
      const id = path.split('/').at(-1)!; state.reads.push(id);
      if (!state.members.has(id)) { await route.fulfill({ status: 403, headers, json: { code: 'forbidden' } }); return; }
      await route.fulfill({ headers, json: { id, owner_id: owner, title: id === planB ? 'Kèo B' : 'Kèo A', activity: 'coffee', starts_at: '2030-01-01T12:30:00Z', status: 'inviting', location: 'Địa điểm riêng tư', capacity: 0, version: 2, members: { [owner]: 'going', [actor]: 'invited' } } }); return;
    }
    await route.fulfill({ headers, json: { items: [] } });
  });
  return state;
}
async function credentials(page: Page) { await page.getByLabel('Email', { exact: true }).fill('ban@example.com'); await page.getByLabel('Mật khẩu', { exact: true }).fill('fixture-password'); await page.getByRole('button', { name: 'Đăng nhập', exact: true }).click(); }
test('cold invite previews privately, survives login/reload, redeems and clears bearer storage', async ({ page }) => {
  const state = await boundaries(page); await page.goto('/invite#token=' + tokenA);
  await expect(page.getByRole('heading', { name: 'Lời mời A', exact: true })).toBeVisible();
  expect(new URL(page.url()).hash).toBe(''); expect(state.reads).toEqual([]);
  await expect(page.getByText('Địa điểm riêng tư', { exact: true })).toHaveCount(0);
  expect(await page.evaluate(t => Object.values(localStorage).some(v => String(v).includes(t)), tokenA)).toBe(false);
  await page.reload(); await expect(page.getByRole('heading', { name: 'Lời mời A', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Đăng nhập để tham gia' }).click(); await credentials(page);
  await page.getByRole('button', { name: 'Tham gia kèo', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Kèo A', exact: true })).toBeVisible();
  await expect(page.getByText('Địa điểm riêng tư', { exact: true })).toBeVisible();
  expect(state.redeems).toEqual([tokenA]); expect(await page.evaluate(() => sessionStorage.getItem('no-plans.incoming-invite'))).toBeNull();
});
for (const [label, malformed] of [['malformed', 'bad'], ['oversized', 'a'.repeat(4100)]]) test(`warm ${label} invitation clears older bearer without falling back after reload`, async ({ page }) => {
  const state = await boundaries(page); await page.goto('/invite#token=' + tokenA);
  await expect(page.getByRole('heading', { name: 'Lời mời A', exact: true })).toBeVisible();
  await page.evaluate(value => { window.location.hash = 'token=' + value; }, malformed);
  await expect(page.getByRole('heading', { name: 'Lời mời không hợp lệ' })).toBeVisible();
  expect(new URL(page.url()).hash).toBe(''); await page.reload();
  await expect(page.getByRole('heading', { name: 'Lời mời không hợp lệ' })).toBeVisible(); expect(state.redeems).toEqual([]);
});
test('new warm invitation wins over a delayed earlier redemption', async ({ page }) => {
  const state = await boundaries(page); await page.goto('/invite#token=' + tokenA);
  await page.getByRole('button', { name: 'Đăng nhập để tham gia' }).click(); await credentials(page);
  state.hold = true; await page.getByRole('button', { name: 'Tham gia kèo', exact: true }).click(); await expect.poll(() => !!state.release).toBe(true);
  await page.evaluate(token => { window.location.hash = 'token=' + token; }, tokenB);
  await expect(page.getByRole('heading', { name: 'Lời mời B', exact: true })).toBeVisible();
  expect(new URL(page.url()).hash).toBe('');
  state.release!(); await page.getByRole('button', { name: 'Tham gia kèo', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Kèo B', exact: true })).toBeVisible(); expect(state.redeems).toEqual([tokenA, tokenB]); expect(state.reads).not.toContain(planA);
});
test('private ID link resumes after authentication but does not grant membership', async ({ page }) => {
  const state = await boundaries(page); await page.goto('/plans/' + planA);
  await expect(page.getByLabel('Email', { exact: true })).toBeVisible(); expect(state.reads).toEqual([]);
  await credentials(page); await expect(page.getByRole('alert')).toContainText('không có quyền');
  await expect(page.getByText('Địa điểm riêng tư', { exact: true })).toHaveCount(0); expect(state.redeems).toEqual([]);
});
test('already joined member can reopen a full-link preview', async ({ page }) => {
  const state = await boundaries(page); state.full = true; state.members.add(planA); await page.goto('/invite#token=' + tokenA);
  await page.getByRole('button', { name: 'Đăng nhập để tham gia' }).click(); await credentials(page);
  await page.getByRole('button', { name: 'Đã tham gia? Mở lại kèo' }).click(); await expect(page.getByRole('heading', { name: 'Kèo A', exact: true })).toBeVisible();
});
