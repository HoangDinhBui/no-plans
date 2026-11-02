import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import type { Place, PlaceOption, Plan } from '../src/lib/api';
const owner = '00000000-0000-4000-8000-000000000001';
const member = '00000000-0000-4000-8000-000000000002';
const id = '00000000-0000-4000-8000-000000000010';
const ideaID = '00000000-0000-4000-8000-000000000020';
const venueID = '00000000-0000-4000-8000-000000000021';
const customID = '00000000-0000-4000-8000-000000000022';
async function fixture(page: Page, actor = owner) {
  const plan: Plan = { id, owner_id: owner, title: 'Kèo bình chọn', activity: 'food', starts_at: '2030-01-01T12:30:00Z', location: '', capacity: 0, status: 'inviting', version: 1, members: { [owner]: 'going', [member]: 'invited' } };
  const options: PlaceOption[] = [{ id: ideaID, plan_id: id, kind: 'idea', name: 'Ăn bánh mì', address: '', vote_count: 0, my_vote: false }, { id: venueID, plan_id: id, kind: 'place', name: 'Công viên', address: 'Cổng chính', vote_count: 0, my_vote: false }];
  const idea: Place = { id: ideaID, name: 'Ăn bánh mì', activity: 'food', kind: 'idea', area: 'TP.HCM', address: '', description: 'Ý tưởng cho cả nhóm', budget_min: 30000, budget_max: 60000, price_note: 'Ngân sách gợi ý, không phải báo giá quán.', people_min: 1, people_max: 6, source_url: '', active: true };
  const venue: Place = { ...idea, id: venueID, name: 'Công viên', activity: 'other', kind: 'place', address: 'Cổng chính', budget_min: null, budget_max: null, price_note: 'Chưa xác minh chi phí.', source_url: 'https://example.com/venue', source_checked_at: '2026-11-01T00:00:00Z' };
  const state = { plan, options, denyOptions: false, conflict: true, versions: [] as string[], requests: [] as string[], bodies: [] as { path: string; body: unknown }[] };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const user = { id: actor, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: {}, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z' };
  const jwt = [Buffer.from('{"alg":"ES256"}').toString('base64url'), Buffer.from(JSON.stringify({ sub: actor, exp })).toString('base64url'), 'fixture'].join('.');
  const headers = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,PATCH,PUT,DELETE' };
  await page.route('https://auth.no-plans.test/**', async route => { if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; } await route.fulfill({ headers, json: { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user } }); });
  await page.route('https://api.no-plans.test/**', async route => {
    const req = route.request(); const url = new URL(req.url()); const path = url.pathname;
    if (req.method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; }
    expect(req.headers().authorization).toBe('Bearer ' + jwt); state.requests.push(req.url());
    if (path === '/v1/places') { await route.fulfill({ headers, json: url.searchParams.get('kind') === 'idea' || url.searchParams.has('cursor') ? { items: [idea] } : { items: [venue], next_cursor: venueID } }); return; }
    if (path.endsWith('/options')) {
      if (state.denyOptions) { await route.fulfill({ status: 403, headers, json: { code: 'forbidden' } }); return; }
      if (req.method() === 'POST') { const body = req.postDataJSON(); state.bodies.push({ path, body }); const record: PlaceOption = body.place_id ? { ...options[0], id: customID } : { id: customID, plan_id: id, kind: 'place', name: body.name, address: body.address, vote_count: 0, my_vote: false }; options.push(record); plan.version++; await route.fulfill({ status: 201, headers, json: record }); return; }
      await route.fulfill({ headers, json: { items: options } }); return;
    }
    if (path.endsWith('/vote')) { const body = req.postDataJSON(); state.bodies.push({ path, body }); for (const option of options) { option.my_vote = option.id === body.option_id; option.vote_count = option.my_vote ? 1 : 0; } plan.version++; await route.fulfill({ headers, json: { changed: true } }); return; }
    if (path.endsWith('/selection')) {
      state.versions.push(req.headers()['if-match']); state.bodies.push({ path, body: req.postDataJSON() });
      if (state.conflict) { state.conflict = false; plan.version++; await route.fulfill({ status: 409, headers, json: { code: 'version_conflict' } }); return; }
      const option = options.find(o => o.id === req.postDataJSON().option_id)!;
      plan.location = option.name + ' · ' + option.address; plan.selected_option_id = option.id; plan.version++; await route.fulfill({ headers, json: plan }); return;
    }
    if (path.endsWith('/confirm')) { plan.status = 'confirmed'; plan.version++; await route.fulfill({ headers, json: plan }); return; }
    await route.fulfill({ headers, json: path === '/v1/plans/' + id ? plan : { items: [] } });
  });
  await page.goto('/'); await page.getByLabel('Email', { exact: true }).fill('ban@example.com'); await page.getByLabel('Mật khẩu', { exact: true }).fill('fixture-password'); await page.getByRole('button', { name: 'Đăng nhập', exact: true }).click(); await expect(page.getByRole('heading', { name: 'Kèo của bạn.' })).toBeVisible();
  return state;
}
test('owner votes on ideas, adds venue, rechecks selection conflict and confirms', async ({ page }) => {
  const state = await fixture(page); await page.goto('/plans/' + id);
  await expect(page.getByRole('button', { name: 'Chọn làm nơi gặp: Ăn bánh mì' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Bình chọn Ăn bánh mì', exact: true }).click();
  await expect(page.getByText('1 phiếu · Bạn đã chọn', { exact: true })).toBeVisible();
  await page.getByLabel('Tên địa điểm đề xuất', { exact: true }).fill('Quán C');
  await page.getByLabel('Địa chỉ địa điểm đề xuất', { exact: true }).fill('Số 12');
  await page.getByRole('button', { name: 'Thêm địa điểm để bình chọn' }).click();
  await page.getByRole('button', { name: 'Chọn làm nơi gặp: Quán C' }).click();
  await page.getByRole('button', { name: 'Xác nhận địa điểm gặp' }).click();
  await expect(page.getByRole('alert')).toContainText('vừa thay đổi');
  await expect(page.getByRole('button', { name: 'Xác nhận địa điểm gặp' })).toBeDisabled();
  await page.getByRole('button', { name: 'Chọn làm nơi gặp: Quán C' }).click();
  await page.getByRole('button', { name: 'Xác nhận địa điểm gặp' }).click();
  await expect(page.getByText('Quán C · Số 12', { exact: true })).toBeVisible();
  expect(state.versions).toEqual(['"3"', '"4"']);
  await page.getByRole('button', { name: 'Chốt kèo', exact: true }).click();
  await expect(page.getByText('Đã chốt kèo', { exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /Bình chọn / })).toHaveCount(0);
  expect(state.bodies[0].body).toEqual({ option_id: ideaID });
});
test('member can change vote but cannot propose or select, and denial clears plan', async ({ page }) => {
  const state = await fixture(page, member); await page.goto('/plans/' + id);
  await page.getByRole('button', { name: 'Bình chọn Công viên', exact: true }).click();
  await expect(page.getByText('1 phiếu · Bạn đã chọn', { exact: true })).toBeVisible();
  await expect(page.getByLabel('Tên địa điểm đề xuất', { exact: true })).toHaveCount(0);
  await expect(page.getByRole('button', { name: /Chọn làm nơi gặp:/ })).toHaveCount(0);
  state.denyOptions = true; await page.getByRole('button', { name: 'Tải lại bình chọn' }).click();
  await expect(page.getByRole('heading', { name: 'Kèo bình chọn' })).toHaveCount(0);
  await expect(page.getByRole('alert')).toContainText('không có quyền');
});
test('discovery filters and pagination preserve unknown-price and idea distinction', async ({ page }) => {
  const state = await fixture(page); await page.getByRole('button', { name: 'Khám phá', exact: true }).click();
  await expect(page.getByText('Chi phí chưa xác minh', { exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Trang gợi ý tiếp' }).click();
  await expect(page.getByRole('heading', { name: 'Ăn bánh mì', exact: true })).toBeVisible();
  await page.getByRole('button', { name: 'Ăn uống', exact: true }).click();
  await page.getByRole('button', { name: 'Ý tưởng', exact: true }).click();
  await page.getByLabel('Khu vực', { exact: true }).fill('TP.HCM');
  await page.getByLabel('Ngân sách tối đa (đ, có thể để trống)').fill('80000');
  await page.getByLabel('Số người đi', { exact: true }).fill('4');
  await page.getByRole('button', { name: 'Tìm gợi ý', exact: true }).click();
  await expect.poll(() => state.requests.filter(url => url.includes('/places?')).at(-1)).toContain('max_budget=80000');
  const query = new URL(state.requests.filter(url => url.includes('/places?')).at(-1)!);
  expect(query.searchParams.get('activity')).toBe('food'); expect(query.searchParams.get('kind')).toBe('idea'); expect(query.searchParams.get('people')).toBe('4'); expect(query.searchParams.has('cursor')).toBe(false);
  await expect(page.getByText('Ngân sách gợi ý, không phải báo giá quán.', { exact: true })).toBeVisible();
});
test('owner can propose a catalogue idea without treating it as a venue', async ({ page }) => {
  const state = await fixture(page); await page.goto('/plans/' + id);
  await page.getByRole('button', { name: 'Tìm trong gợi ý NO PLANS' }).click();
  await page.getByRole('button', { name: 'Ý tưởng', exact: true }).click();
  await page.getByRole('button', { name: 'Tìm gợi ý', exact: true }).click();
  await page.getByRole('button', { name: 'Đề xuất Ăn bánh mì', exact: true }).click();
  await expect(page.getByText('Đã thêm lựa chọn để cả nhóm bình chọn.', { exact: true })).toBeVisible();
  expect(state.bodies.at(-1)?.body).toEqual({ place_id: ideaID });
  await expect(page.getByRole('button', { name: 'Chọn làm nơi gặp: Ăn bánh mì' })).toHaveCount(0);
  await expect(page.getByRole('button', { name: 'Đóng khám phá' })).toHaveCount(0);
});
