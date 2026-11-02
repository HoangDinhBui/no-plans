import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

const token = '00000000-0000-4000-8000-000000000001.' + 'a'.repeat(43);
const owner = '00000000-0000-4000-8000-000000000002';
const guest = '00000000-0000-4000-8000-000000000003';
const planId = '00000000-0000-4000-8000-000000000010';

// These browser tests exercise the real UI and Auth SDK against controlled HTTP
// boundaries. Go/PostgreSQL/JWT integration is independently covered in backend.
async function boundaries(page: Page) {
  const state = { full: false, revoked: false, joined: false, response: 'invited' };
  const user = { id: guest, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: { provider: 'email', providers: ['email'] }, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z', email_confirmed_at: '2026-01-01T00:00:00Z' };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const jwt = [Buffer.from(JSON.stringify({ alg: 'ES256', typ: 'JWT' })).toString('base64url'), Buffer.from(JSON.stringify({ sub: guest, aud: 'authenticated', role: 'authenticated', exp })).toString('base64url'), Buffer.from('test-signature').toString('base64url')].join('.');
  await page.route('https://auth.no-plans.test/auth/v1/**', async route => {
    const path = new URL(route.request().url()).pathname;
    const cors = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*' };
    if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers: { ...cors, 'access-control-allow-methods': 'GET, POST, PUT, DELETE' } }); return; }
    if (path.endsWith('/logout')) { await route.fulfill({ status: 204, headers: cors }); return; }
    const body = path.endsWith('/signup') || path.endsWith('/user') ? user : { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user };
    await route.fulfill({ status: 200, headers: cors, json: body });
  });
  await page.route(/\/v1\/(invite-preview|invite-redeem|plans\/)/, async route => {
    const req = route.request(); const path = new URL(req.url()).pathname;
    expect(req.url()).not.toContain(token);
    if (path.endsWith('/invite-preview')) {
      expect(req.headers()['x-invite-token']).toBe(token);
      await route.fulfill(state.full ? { status: 404, json: { code: 'invite_unavailable' } } : { status: 200, json: { title: 'Cà phê cuối tuần', activity: 'coffee', starts_at: '2030-01-05T12:30:00Z' } }); return;
    }
    if (req.headers().authorization !== 'Bearer ' + jwt) { await route.fulfill({ status: 401, json: { code: 'unauthorized' } }); return; }
    if (path.endsWith('/invite-redeem')) {
      expect(req.headers()['x-invite-token']).toBe(token);
      if (state.full && !state.joined) { await route.fulfill({ status: 404, json: { code: 'invite_unavailable' } }); return; }
      state.joined = true; await route.fulfill({ status: 200, json: { plan_id: planId } }); return;
    }
    if (state.revoked) { await route.fulfill({ status: 403, json: { code: 'forbidden' } }); return; }
    if (path.endsWith('/rsvp')) { state.response = req.postDataJSON().response; await route.fulfill({ status: 200, json: { changed: true } }); return; }
    await route.fulfill({ status: 200, json: { id: planId, owner_id: owner, title: 'Cà phê cuối tuần', activity: 'coffee', starts_at: '2030-01-05T12:30:00Z', status: 'inviting', location: 'Quán cà phê kiểm thử', version: 4, members: { [owner]: 'going', [guest]: state.response } } });
  });
  return state;
}

async function signIn(page: Page) {
  await page.getByRole('button', { name: /Mình tham gia/ }).focus(); await page.keyboard.press('Enter');
  await page.getByLabel('Email', { exact: true }).fill('ban@example.com');
  await page.getByLabel('Mật khẩu', { exact: true }).fill('test-password');
  await page.getByRole('button', { name: 'Đăng nhập & tham gia' }).click();
  await expect(page.getByText('Quán cà phê kiểm thử', { exact: true })).toBeVisible();
}

test('private preview is responsive, keyboard usable and has no automated AA violations', async ({ page }, info) => {
  await boundaries(page); await page.goto('/#token=' + token);
  await expect(page.getByRole('heading', { name: 'Cà phê cuối tuần' })).toBeVisible();
  await expect(page.getByText('Quán cà phê kiểm thử', { exact: true })).toHaveCount(0);
  expect(new URL(page.url()).hash).toBe('');
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  const report = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21aa']).analyze();
  expect(report.violations).toEqual([]);
  await page.screenshot({ path: info.outputPath('preview.png'), fullPage: true });
});

test('sign in, redeem, RSVP and reload preserve the plan without retaining invite secret', async ({ page }) => {
  await boundaries(page); await page.goto('/#token=' + token); await signIn(page);
  await page.getByRole('button', { name: 'Chưa chắc', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Chưa chắc', exact: true })).toHaveAttribute('aria-pressed', 'true');
  await expect(page.getByRole('status')).toHaveText('Đã lưu phản hồi của bạn.');
  expect(await page.evaluate(() => sessionStorage.getItem('no-plans.pending-invite'))).toBeNull();
  await page.reload(); await expect(page.getByText('Quán cà phê kiểm thử', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Chưa chắc', exact: true })).toHaveAttribute('aria-pressed', 'true');
});

test('existing member can reopen a link whose last use is already consumed', async ({ page }) => {
  const state = await boundaries(page); await page.goto('/#token=' + token); await signIn(page); state.full = true;
  await page.goto('/#token=' + token);
  await page.getByRole('button', { name: 'Đã tham gia? Mở lại kèo' }).click();
  await expect(page.getByText('Quán cà phê kiểm thử', { exact: true })).toBeVisible();
});

test('known membership revocation clears private details on foreground refresh', async ({ page }) => {
  const state = await boundaries(page); await page.goto('/#token=' + token); await signIn(page); state.revoked = true;
  await page.evaluate(() => window.dispatchEvent(new Event('focus')));
  await expect(page.getByText('Quán cà phê kiểm thử', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('alert')).toContainText('không còn quyền');
});
