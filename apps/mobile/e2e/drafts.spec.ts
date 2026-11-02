import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
const userID = '00000000-0000-4000-8000-000000000001';
const planID = '00000000-0000-4000-8000-000000000002';
// Real Expo UI/Auth SDK, controlled HTTP boundaries; does not replace Go/PG or device tests.
async function boundaries(page: Page) {
  const state = { keys: [] as string[], attempts: 0, revokeOnRSVP: false, revoked: false, holdRead: false, releaseRead: null as (() => void) | null };
  const plan = { id: planID, owner_id: userID, title: 'Cà phê bản nháp', starts_at: '2030-01-01T12:30:00Z', activity: 'coffee', location: 'Quán thử nghiệm riêng tư', status: 'inviting', version: 1, capacity: 0, members: { [userID]: 'going' } };
  const user = { id: userID, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: {}, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z' };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const jwt = [Buffer.from('{"alg":"ES256"}').toString('base64url'), Buffer.from(JSON.stringify({ sub: userID, aud: 'authenticated', role: 'authenticated', exp })).toString('base64url'), 'fixture'].join('.');
  const cors = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,PATCH,PUT,DELETE' };
  await page.route('https://auth.no-plans.test/**', async route => {
    if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers: cors }); return; }
    const userOnly = new URL(route.request().url()).pathname.endsWith('/user');
    await route.fulfill({ status: 200, headers: cors, json: userOnly ? user : { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user } });
  });
  await page.route('https://api.no-plans.test/**', async route => {
    const req = route.request();
    if (req.method() === 'OPTIONS') { await route.fulfill({ status: 204, headers: cors }); return; }
    expect(req.headers().authorization).toBe('Bearer ' + jwt);
    const path = new URL(req.url()).pathname;
    if (state.holdRead && path === '/v1/plans/' + planID) {
      state.holdRead = false;
      await new Promise<void>(resolve => { state.releaseRead = resolve; });
      await route.fulfill({ status: 200, headers: cors, json: plan }); return;
    }
    if (path.endsWith('/rsvp') && state.revokeOnRSVP) { state.revoked = true; await route.fulfill({ status: 403, headers: cors, json: { code: 'forbidden' } }); return; }
    if (state.revoked && path === '/v1/plans/' + planID) { await new Promise(resolve => setTimeout(resolve, 10000)); await route.fulfill({ status: 403, headers: cors, json: { code: 'forbidden' } }); return; }
    if (path === '/v1/plans' && req.method() === 'POST') {
      state.keys.push(req.headers()['idempotency-key']); state.attempts++;
      if (state.attempts === 1) { await route.abort('failed'); return; }
      await route.fulfill({ status: 201, headers: cors, json: { id: planID } }); return;
    }
    if (path === '/v1/plans/' + planID) { await route.fulfill({ status: 200, headers: cors, json: plan }); return; }
    await route.fulfill({ status: 200, headers: cors, json: { items: [] } });
  });
  return state;
}
async function login(page: Page) {
  await page.goto('/');
  await page.getByLabel('Email', { exact: true }).fill('ban@example.com');
  await page.getByLabel('Mật khẩu', { exact: true }).fill('fixture-password');
  await page.getByRole('button', { name: 'Đăng nhập', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Kèo của bạn.' })).toBeVisible();
}
test('form and idempotency key survive reload after an uncertain creation', async ({ page }, info) => {
  const renderErrors: string[] = [];
  page.on('console', message => { if (message.text().includes('Unexpected text node')) renderErrors.push(message.text()); });
  const state = await boundaries(page); await login(page);
  await page.getByRole('button', { name: '↗ Lên kèo' }).click();
  await page.getByLabel('Tên kèo', { exact: true }).fill('Cà phê bản nháp');
  await page.getByLabel('Ngày (YYYY-MM-DD)', { exact: true }).fill('2030-01-01');
  await page.screenshot({ path: info.outputPath('create-plan.png'), fullPage: true });
  await page.getByRole('button', { name: 'Tạo kèo ↗', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Chưa kết nối');
  await page.reload();
  await expect(page.getByLabel('Tên kèo', { exact: true })).toHaveValue('Cà phê bản nháp');
  await expect(page.getByLabel('Ngày (YYYY-MM-DD)', { exact: true })).toHaveValue('2030-01-01');
  await page.getByRole('button', { name: 'Tạo kèo ↗', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Cà phê bản nháp' })).toBeVisible();
  expect(state.keys).toHaveLength(2); expect(state.keys[0]).toBeTruthy(); expect(state.keys[1]).toBe(state.keys[0]);
  expect(await page.evaluate(actor => localStorage.getItem('no-plans.plan-draft.' + actor), userID)).toBeNull();
  expect(renderErrors).toEqual([]);
});

test('a pre-revocation GET arriving late cannot restore private details', async ({ page }) => {
  const state = await boundaries(page); await login(page); await page.goto('/plans/' + planID);
  await expect(page.getByText('Quán thử nghiệm riêng tư', { exact: true })).toBeVisible();
  state.holdRead = true;
  await page.getByRole('button', { name: 'Tải lại kèo', exact: true }).click();
  await expect.poll(() => !!state.releaseRead).toBe(true);
  state.revokeOnRSVP = true;
  await page.getByRole('button', { name: 'Chưa chắc', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('không có quyền');
  const delivered = page.waitForResponse(r => r.url().endsWith('/v1/plans/' + planID) && r.status() === 200);
  state.releaseRead!(); await delivered;
  // Allow the completed response's JSON and React update to settle before asserting absence.
  await page.waitForTimeout(300);
  await expect(page.getByText('Quán thử nghiệm riêng tư', { exact: true })).toHaveCount(0);
});

test('known403 from RSVP immediately hides private content while refetch is delayed', async ({ page }) => {
  const state = await boundaries(page); await login(page);
  await page.goto('/plans/' + planID);
  await expect(page.getByText('Quán thử nghiệm riêng tư', { exact: true })).toBeVisible();
  state.revokeOnRSVP = true;
  await page.getByRole('button', { name: 'Chưa chắc', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('không có quyền');
  await expect(page.getByText('Quán thử nghiệm riêng tư', { exact: true })).toHaveCount(0);
});
