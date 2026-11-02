import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import type { Plan } from '../src/lib/api';
const owner = '00000000-0000-4000-8000-000000000001';
const member = '00000000-0000-4000-8000-000000000003';
const id = '00000000-0000-4000-8000-000000000002';
async function fixture(page: Page, actor = owner, past = false) {
  const plan: Plan = { id, owner_id: owner, title: 'Kèo của hội', activity: 'coffee', starts_at: past ? '2020-01-01T12:30:00Z' : '2030-01-01T12:30:45.123Z', location: 'Địa điểm ban đầu', capacity: 5, status: past ? 'confirmed' : 'inviting', version: 3, members: { [owner]: 'going', [member]: 'invited' } };
  const state = { plan, conflict: true, failNextRefresh: false, failReads: 0, patches: [] as { body: unknown; version: string | undefined }[], commands: [] as string[] };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const user = { id: actor, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: {}, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z' };
  const jwt = [Buffer.from('{"alg":"ES256"}').toString('base64url'), Buffer.from(JSON.stringify({ sub: actor, exp })).toString('base64url'), 'fixture'].join('.');
  const headers = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,PATCH,PUT,DELETE' };
  await page.route('https://auth.no-plans.test/**', async route => {
    if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; }
    await route.fulfill({ headers, json: { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user } });
  });
  await page.route('https://api.no-plans.test/**', async route => {
    const req = route.request(); const path = new URL(req.url()).pathname;
    if (req.method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; }
    expect(req.headers().authorization).toBe('Bearer ' + jwt);
    if (path === '/v1/friend-requests' || path.endsWith('/members') || path.endsWith('/invite-links')) { await route.fulfill({ headers, json: { items: [] } }); return; }
    if (req.method() === 'GET' && path === '/v1/plans/' + id && state.failReads > 0) { state.failReads--; await route.fulfill({ status: 503, headers, json: { code: 'unavailable' } }); return; }
    if (req.method() === 'PATCH') {
      state.patches.push({ body: req.postDataJSON(), version: req.headers()['if-match'] });
      if (state.conflict) { state.conflict = false; plan.version++; plan.location = 'Địa điểm đã đổi'; if (state.failNextRefresh) state.failReads++; await route.fulfill({ status: 409, headers, json: { code: 'version_conflict' } }); return; }
      Object.assign(plan, req.postDataJSON()); plan.version++;
    } else if (req.method() === 'POST') {
      const command = path.split('/').at(-1)!; state.commands.push(command);
      plan.status = command === 'confirm' ? 'confirmed' : command === 'cancel' ? 'cancelled' : 'completed'; plan.version++;
    }
    await route.fulfill({ headers, json: path === '/v1/plans' ? { items: [] } : plan });
  });
  await page.goto('/'); await page.getByLabel('Email', { exact: true }).fill('ban@example.com'); await page.getByLabel('Mật khẩu', { exact: true }).fill('fixture-password'); await page.getByRole('button', { name: 'Đăng nhập', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Kèo của bạn.' })).toBeVisible();
  await page.goto('/plans/' + id); await expect(page.getByRole('heading', { name: 'Kèo của hội' })).toBeVisible();
  return state;
}
test('owner edits preserve conflict draft and require explicit refresh, then confirm/cancel', async ({ page }) => {
  const state = await fixture(page);
  await page.getByRole('button', { name: 'Sửa kèo', exact: true }).click();
  await page.getByLabel('Sửa tên kèo', { exact: true }).fill('Tên đã sửa');
  await page.getByRole('button', { name: 'Lưu thay đổi', exact: true }).click();
  await expect(page.getByRole('alert')).toContainText('vừa thay đổi');
  await expect(page.getByLabel('Sửa tên kèo', { exact: true })).toHaveValue('Tên đã sửa');
  expect(state.patches[0]).toEqual({ body: { title: 'Tên đã sửa' }, version: '"3"' });
  await page.getByRole('button', { name: 'Bỏ bản sửa, lấy thông tin mới' }).click();
  await expect(page.getByLabel('Sửa địa điểm', { exact: true })).toHaveValue('Địa điểm đã đổi');
  await page.getByLabel('Sửa tên kèo', { exact: true }).fill('Tên đã kiểm tra');
  await page.getByRole('button', { name: 'Lưu thay đổi', exact: true }).click();
  await expect(page.getByRole('heading', { name: 'Tên đã kiểm tra' })).toBeVisible();
  expect(state.patches[1]).toEqual({ body: { title: 'Tên đã kiểm tra' }, version: '"4"' });
  await page.getByRole('button', { name: 'Chốt kèo', exact: true }).click();
  await expect(page.getByText('Đã chốt kèo', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Hủy kèo', exact: true }).click();
  await page.getByRole('button', { name: 'Giữ nguyên kèo', exact: true }).click();
  expect(state.commands).toEqual(['confirm']);
  await page.getByRole('button', { name: 'Hủy kèo', exact: true }).click();
  await page.getByRole('button', { name: 'Xác nhận hủy kèo', exact: true }).click();
  await expect(page.getByText('Đã hủy', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Chưa chắc', exact: true })).toHaveCount(0);
  expect(state.commands).toEqual(['confirm', 'cancel']);
});
test('member sees RSVP but no owner actions', async ({ page }) => {
  await fixture(page, member);
  await expect(page.getByRole('button', { name: 'Chưa chắc', exact: true })).toBeVisible();
  for (const name of ['Sửa kèo', 'Chốt kèo', 'Hủy kèo', 'Kết thúc kèo']) await expect(page.getByRole('button', { name, exact: true })).toHaveCount(0);
});
test('failed conflict refresh preserves original edit snapshot until explicit rebase', async ({ page }) => {
  const state = await fixture(page); state.failNextRefresh = true;
  await page.getByRole('button', { name: 'Sửa kèo', exact: true }).click();
  await page.getByLabel('Sửa tên kèo', { exact: true }).fill('Bản sửa chưa được lưu');
  const failed = page.waitForResponse(r => r.url().endsWith('/v1/plans/' + id) && r.status() === 503);
  await page.getByRole('button', { name: 'Lưu thay đổi', exact: true }).click(); await failed;
  await expect(page.getByLabel('Sửa tên kèo', { exact: true })).toHaveValue('Bản sửa chưa được lưu');
  await page.getByRole('button', { name: 'Tải lại kèo', exact: true }).first().click();
  await expect(page.getByRole('button', { name: 'Bỏ bản sửa, lấy thông tin mới' })).toBeVisible();
  await expect(page.getByLabel('Sửa tên kèo', { exact: true })).toHaveValue('Bản sửa chưa được lưu');
  expect(state.patches[0].version).toBe('"3"');
  await page.getByRole('button', { name: 'Bỏ bản sửa, lấy thông tin mới' }).click();
  await expect(page.getByLabel('Sửa tên kèo', { exact: true })).toHaveValue('Kèo của hội');
});
test('confirmed past plan can be completed only after explicit confirmation', async ({ page }) => {
  const state = await fixture(page, owner, true);
  await page.getByRole('button', { name: 'Kết thúc kèo', exact: true }).click();
  expect(state.commands).toEqual([]);
  await page.getByRole('button', { name: 'Xác nhận kết thúc', exact: true }).click();
  await expect(page.getByText('Đã kết thúc', { exact: true })).toBeVisible();
  expect(state.commands).toEqual(['complete']);
});
