import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import type { InviteLink, Plan } from '../src/lib/api';
const alice = '00000000-0000-4000-8000-000000000001';
const bob = '00000000-0000-4000-8000-000000000002';
const dave = '00000000-0000-4000-8000-000000000004';
const planID = '00000000-0000-4000-8000-000000000010';
const linkID = '00000000-0000-4000-8000-000000000020';
const token = linkID + '.' + 'a'.repeat(43);
async function fixture(page: Page, actor = alice) {
  const plan: Plan = { id: planID, owner_id: alice, title: 'Hội mình cà phê', activity: 'coffee', starts_at: '2030-01-01T12:30:00Z', location: 'Địa điểm riêng tư', capacity: 0, status: 'inviting', version: 1, members: { [alice]: 'going', [dave]: 'invited' } };
  const state = { plan, links: [] as InviteLink[], keys: [] as string[], invites: [] as unknown[], revoked: [] as string[], requests: [] as string[], membersDenied: false, reports: [] as unknown[] };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const user = { id: actor, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: {}, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z' };
  const jwt = [Buffer.from('{"alg":"ES256"}').toString('base64url'), Buffer.from(JSON.stringify({ sub: actor, exp })).toString('base64url'), 'fixture'].join('.');
  const headers = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,PATCH,PUT,DELETE' };
  await page.addInitScript(() => Object.defineProperty(navigator, 'share', { configurable: true, value: async (data: ShareData) => { (window as Window & { shared?: ShareData }).shared = data; } }));
  await page.route('https://auth.no-plans.test/**', async route => { if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; } await route.fulfill({ headers, json: { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user } }); });
  await page.route('https://api.no-plans.test/**', async route => {
    const req = route.request(); const url = new URL(req.url()); const path = url.pathname;
    expect(req.url()).not.toContain(token);
    if (req.method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; }
    expect(req.headers().authorization).toBe('Bearer ' + jwt); state.requests.push(req.method() + ' ' + path);
    if (path === '/v1/plans/' + planID) { await route.fulfill({ headers, json: plan }); return; }
    if (path === '/v1/reports' && req.method() === 'POST') {
      state.reports.push(req.postDataJSON());
      await route.fulfill({ status: 201, headers, json: { id: 'report-fixture' } }); return;
    }
    if (path.endsWith('/members')) {
      if (state.membersDenied) { await route.fulfill({ status: 403, headers, json: { code: 'forbidden' } }); return; }
      const ids = url.searchParams.has('cursor') ? Object.keys(plan.members).filter(id => id === bob) : Object.keys(plan.members).filter(id => id !== bob);
      await route.fulfill({ headers, json: { items: ids.map(id => ({ id, display_name: id === alice ? 'Alice' : id === bob ? 'Bob' : 'Dave', response: plan.members[id] })), ...(!url.searchParams.has('cursor') && bob in plan.members ? { next_cursor: dave } : {}) } }); return;
    }
    if (path.endsWith('/invitations')) { state.invites.push(req.postDataJSON()); plan.members[req.postDataJSON().user_id] = 'invited'; plan.version++; await route.fulfill({ headers, json: { status: 'ok' } }); return; }
    if (path.includes('/invitations/')) { const id = path.split('/').at(-1)!; state.revoked.push(id); delete plan.members[id]; plan.version++; await route.fulfill({ headers, json: { status: 'ok' } }); return; }
    if (path.endsWith('/invite-links')) {
      if (req.method() === 'POST') {
        state.keys.push(req.headers()['idempotency-key']); expect(req.postDataJSON()).toEqual({ max_uses: 7 });
        if (state.keys.length === 1) { await route.abort('failed'); return; }
        const link = { id: linkID, plan_id: planID, expires_at: '2030-01-01T00:00:00Z', max_uses: 7, use_count: 0 }; state.links = [link];
        await route.fulfill({ status: 201, headers, json: { ...link, token } }); return;
      }
      await route.fulfill({ headers, json: { items: state.links } }); return;
    }
    if (path === '/v1/invite-links/' + linkID) { state.links[0].revoked_at = new Date().toISOString(); await route.fulfill({ headers, json: { status: 'ok' } }); return; }
    if (path === '/v1/friend-requests') { await route.fulfill({ headers, json: { items: [{ id: 'friendship', low_id: alice, high_id: bob, requester_id: alice, status: 'accepted', peer: { id: bob, display_name: 'Bob' } }] } }); return; }
    await route.fulfill({ headers, json: { items: [] } });
  });
  await page.goto('/'); await page.getByLabel('Email', { exact: true }).fill('ban@example.com'); await page.getByLabel('Mật khẩu', { exact: true }).fill('fixture-password'); await page.getByRole('button', { name: 'Đăng nhập', exact: true }).click(); await expect(page.getByRole('heading', { name: 'Kèo của bạn.' })).toBeVisible();
  await page.goto('/plans/' + planID); await expect(page.getByRole('heading', { name: 'Hội mình cà phê' })).toBeVisible();
  return state;
}
test('owner invites friend, reads participant pages and explicitly revokes membership', async ({ page }) => {
  const state = await fixture(page);
  await page.getByRole('button', { name: 'Mời Bob vào kèo' }).click();
  await expect(page.getByRole('button', { name: 'Trang người tham gia tiếp' })).toBeVisible();
  await page.getByRole('button', { name: 'Trang người tham gia tiếp' }).click();
  await page.getByRole('button', { name: 'Thu hồi lời mời của Bob' }).click();
  expect(state.revoked).toEqual([]);
  await page.getByRole('button', { name: 'Xác nhận thu hồi thành viên' }).click();
  await expect(page.getByRole('heading', { name: 'Bob', exact: true })).toHaveCount(0);
  expect(state.invites).toEqual([{ user_id: bob }]); expect(state.revoked).toEqual([bob]);
});
test('link retry preserves key, shares fragment URL only and revoke removes share control', async ({ page }) => {
  const state = await fixture(page);
  await page.getByLabel('Số lượt cho link mới (1–1000)').fill('7');
  await page.getByRole('button', { name: 'Tạo link lời mời', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('Chưa kết nối');
  await page.getByRole('button', { name: 'Tạo link lời mời', exact: true }).click();
  const expected = 'https://invite.no-plans.test/#token=' + token;
  await expect(page.getByText(expected, { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Chia sẻ link vừa tạo' }).click();
  expect(await page.evaluate(() => (window as Window & { shared?: ShareData }).shared?.text)).toBe(expected);
  expect(state.keys).toHaveLength(2); expect(state.keys[1]).toBe(state.keys[0]);
  expect(await page.evaluate(t => Object.values(localStorage).some(value => String(value).includes(t)), token)).toBe(false);
  await page.getByRole('button', { name: 'Thu hồi link 1', exact: true }).click();
  expect(state.links[0].revoked_at).toBeUndefined();
  await page.getByRole('button', { name: 'Xác nhận thu hồi link' }).click();
  await expect(page.getByRole('button', { name: 'Chia sẻ link vừa tạo' })).toHaveCount(0);
  await expect(page.getByText(expected, { exact: true })).toHaveCount(0);
  expect(state.links[0].revoked_at).toBeTruthy(); expect(dave in state.plan.members).toBe(true);
});
test('member cannot fetch owner link metadata; member-list denial clears whole private plan', async ({ page }) => {
  const state = await fixture(page, dave);
  await expect(page.getByRole('heading', { name: 'Alice', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Tạo link lời mời', exact: true })).toHaveCount(0);
  expect(state.requests.some(path => path.endsWith('/invite-links'))).toBe(false);
  state.membersDenied = true;
  await page.getByRole('button', { name: 'Tải lại người tham gia' }).click();
  await expect(page.getByText('Địa điểm riêng tư', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Alice', exact: true })).toHaveCount(0);
  await expect(page.getByRole('alert')).toContainText('không có quyền');
});

test('participant can report another participant with a selected reason', async ({ page }) => {
  const state = await fixture(page);
  await expect(page.getByRole('button', { name: 'Báo cáo Alice' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Báo cáo Dave' }).click();
  const send = page.getByRole('button', { name: 'Gửi báo cáo' });
  await expect(send).toBeDisabled();
  await page.getByRole('button', { name: 'Tin rác' }).click();
  await send.click();
  await expect(page.getByText('Đã gửi báo cáo.')).toBeVisible();
  expect(state.reports).toEqual([{ plan_id: planID, target_id: dave, reason: 'spam' }]);
});
