import { test, expect } from '@playwright/test';
import type { Page } from '@playwright/test';
import type { Friendship, GroupDetail } from '../src/lib/api';
const alice = '00000000-0000-4000-8000-000000000001';
const bob = '00000000-0000-4000-8000-000000000002';
const carol = '00000000-0000-4000-8000-000000000003';
const dave = '00000000-0000-4000-8000-000000000004';
const groupID = '00000000-0000-4000-8000-000000000010';
const requestID = '00000000-0000-4000-8000-000000000020';
const pageCursor = '00000000-0000-4000-8000-000000000021';
async function fixture(page: Page, owner = alice) {
  const group: GroupDetail = { id: groupID, owner_id: owner, notifications_muted: false, name: 'Hội cuối tuần', members: [{ id: alice, display_name: 'Alice' }, ...(owner === bob ? [{ id: bob, display_name: 'Bob' }] : [])] };
  const incoming: Friendship = { id: requestID, low_id: alice, high_id: bob, requester_id: bob, status: 'pending', peer: { id: bob, display_name: 'Bob' } };
  const outgoing: Friendship = { id: pageCursor, low_id: alice, high_id: dave, requester_id: alice, status: 'pending', peer: { id: dave, display_name: 'Dave' } };
  const accepted: Friendship = { id: '00000000-0000-4000-8000-000000000022', low_id: alice, high_id: carol, requester_id: alice, status: 'accepted', peer: { id: carol, display_name: 'Carol' } };
  const state = { group, left: false, sentKeys: [] as string[], sentBodies: [] as unknown[], actions: [] as string[] };
  const exp = Math.floor(Date.now() / 1000) + 3600;
  const user = { id: alice, aud: 'authenticated', role: 'authenticated', email: 'ban@example.com', app_metadata: {}, user_metadata: {}, identities: [], created_at: '2026-01-01T00:00:00Z' };
  const jwt = [Buffer.from('{"alg":"ES256"}').toString('base64url'), Buffer.from(JSON.stringify({ sub: alice, exp })).toString('base64url'), 'fixture'].join('.');
  const headers = { 'access-control-allow-origin': '*', 'access-control-allow-headers': '*', 'access-control-allow-methods': 'GET,POST,PATCH,PUT,DELETE' };
  await page.route('https://auth.no-plans.test/**', async route => { if (route.request().method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; } await route.fulfill({ headers, json: { access_token: jwt, token_type: 'bearer', refresh_token: 'fixture-refresh', expires_in: 3600, expires_at: exp, user } }); });
  await page.route('https://api.no-plans.test/**', async route => {
    const req = route.request(); const url = new URL(req.url()); const path = url.pathname;
    if (req.method() === 'OPTIONS') { await route.fulfill({ status: 204, headers }); return; }
    expect(req.headers().authorization).toBe('Bearer ' + jwt);
    if (path === '/v1/friend-requests' && req.method() === 'POST') {
      state.sentBodies.push(req.postDataJSON()); state.sentKeys.push(req.headers()['idempotency-key']);
      if (state.sentKeys.length === 1) { await route.abort('failed'); return; }
      await route.fulfill({ status: 201, headers, json: outgoing }); return;
    }
    if (path === '/v1/friend-requests/' + requestID + '/accept') { incoming.status = 'accepted'; await route.fulfill({ headers, json: incoming }); return; }
    if (path === '/v1/friend-requests') { await route.fulfill({ headers, json: url.searchParams.has('cursor') ? { items: [accepted] } : { items: [incoming, outgoing], next_cursor: pageCursor } }); return; }
    if (path === '/v1/groups/' + groupID + '/notifications' && req.method() === 'PUT') {
      group.notifications_muted = req.postDataJSON().muted;
      await route.fulfill({ headers, json: { muted: group.notifications_muted } }); return;
    }
    if (path.startsWith('/v1/groups/' + groupID) && req.method() !== 'GET') {
      state.actions.push(req.method() + ' ' + path);
      const target = path.split('/').at(-1)!;
      if (path.endsWith('/owner')) group.owner_id = req.postDataJSON().user_id;
      else if (req.method() === 'POST') group.members.push({ id: target, display_name: target === bob ? 'Bob' : 'Carol' });
      else { group.members = group.members.filter(m => m.id !== target); if (target === alice) state.left = true; }
      await route.fulfill({ headers, json: { status: 'ok' } }); return;
    }
    if (path === '/v1/groups/' + groupID) { await route.fulfill(state.left ? { status: 403, headers, json: { code: 'forbidden' } } : { headers, json: group }); return; }
    await route.fulfill({ headers, json: path === '/v1/groups' ? { items: state.left ? [] : [group] } : { items: [] } });
  });
  await page.goto('/'); await page.getByLabel('Email', { exact: true }).fill('ban@example.com'); await page.getByLabel('Mật khẩu', { exact: true }).fill('fixture-password'); await page.getByRole('button', { name: 'Đăng nhập', exact: true }).click(); await expect(page.getByRole('heading', { name: 'Kèo của bạn.' })).toBeVisible();
  return state;
}
test('accept incoming friend, invite across pages, remove, transfer ownership and leave', async ({ page }) => {
  const state = await fixture(page);
  await page.getByRole('button', { name: 'Bạn bè', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Chấp nhận Dave' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Chấp nhận Bob' }).click();
  await expect(page.getByText('Đã kết bạn. Giờ có thể thêm bạn vào nhóm.', { exact: true })).toBeVisible();
  await page.goto('/groups'); await page.getByRole('button', { name: 'Mở nhóm Hội cuối tuần' }).click();
  await page.getByRole('button', { name: 'Thêm Bob', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Xóa Bob khỏi nhóm' })).toBeVisible();
  await page.getByRole('button', { name: 'Xem thêm bạn để mời' }).click();
  await page.getByRole('button', { name: 'Thêm Carol', exact: true }).click();
  await page.getByRole('button', { name: 'Xóa Carol khỏi nhóm' }).click();
  expect(state.group.members.some(m => m.id === carol)).toBe(true);
  await page.getByRole('button', { name: 'Xác nhận xóa thành viên' }).click();
  await expect(page.getByRole('button', { name: 'Xóa Carol khỏi nhóm' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Chuyển quyền cho Bob' }).click();
  await page.getByRole('button', { name: 'Giữ nguyên nhóm' }).click();
  expect(state.group.owner_id).toBe(alice);
  await page.getByRole('button', { name: 'Chuyển quyền cho Bob' }).click();
  await page.getByRole('button', { name: 'Xác nhận chuyển quyền' }).click();
  await expect(page.getByRole('button', { name: 'Rời nhóm', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Xóa Bob khỏi nhóm' })).toHaveCount(0);
  await page.getByRole('button', { name: 'Rời nhóm', exact: true }).click();
  await page.getByRole('button', { name: 'Xác nhận rời nhóm' }).click();
  await expect(page.getByRole('heading', { name: 'Hội của bạn.' })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Mở nhóm Hội cuối tuần' })).toHaveCount(0);
  expect(state.actions).toEqual(['POST /v1/groups/' + groupID + '/members/' + bob, 'POST /v1/groups/' + groupID + '/members/' + carol, 'DELETE /v1/groups/' + groupID + '/members/' + carol, 'PUT /v1/groups/' + groupID + '/owner', 'DELETE /v1/groups/' + groupID + '/members/' + alice]);
});
test('friend request failure retries same key and never submits actor ID', async ({ page }) => {
  const state = await fixture(page); await page.goto('/friends');
  await page.getByLabel('Mã bạn bè', { exact: true }).fill(alice);
  await page.getByRole('button', { name: 'Gửi lời mời kết bạn' }).click();
  await expect(page.getByRole('alert')).toContainText('mã của bạn'); expect(state.sentKeys).toHaveLength(0);
  await page.getByLabel('Mã bạn bè', { exact: true }).fill(dave);
  await page.getByRole('button', { name: 'Gửi lời mời kết bạn' }).click();
  await expect(page.getByRole('alert')).toContainText('Chưa kết nối');
  await page.getByRole('button', { name: 'Gửi lời mời kết bạn' }).click();
  await expect(page.getByText('Đã gửi lời mời kết bạn.', { exact: true })).toBeVisible();
  expect(state.sentKeys).toHaveLength(2); expect(state.sentKeys[0]).toBeTruthy(); expect(state.sentKeys[1]).toBe(state.sentKeys[0]); expect(state.sentBodies).toEqual([{ recipient_id: dave }, { recipient_id: dave }]);
});
test('ordinary member can leave but cannot manage other members', async ({ page }) => {
  await fixture(page, bob); await page.goto('/groups/' + groupID);
  await expect(page.getByRole('button', { name: 'Rời nhóm', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: /Chuyển quyền cho/ })).toHaveCount(0);
  await expect(page.getByRole('button', { name: /khỏi nhóm/ })).toHaveCount(0);
  await expect(page.getByRole('heading', { name: 'Rủ thêm bạn' })).toHaveCount(0);
});
test('ownership refresh removes an already-open owner confirmation', async ({ page }) => {
  const state = await fixture(page); state.group.members.push({ id: bob, display_name: 'Bob' });
  await page.goto('/groups/' + groupID);
  await page.getByRole('button', { name: 'Chuyển quyền cho Bob' }).click();
  await expect(page.getByRole('button', { name: 'Xác nhận chuyển quyền' })).toBeVisible();
  state.group.owner_id = bob;
  await page.getByRole('button', { name: 'Tải lại nhóm', exact: true }).click();
  await expect(page.getByRole('button', { name: 'Rời nhóm', exact: true })).toBeVisible();
  await expect(page.getByRole('button', { name: 'Xác nhận chuyển quyền' })).toHaveCount(0);
  expect(state.actions).toEqual([]);
});

test('member can mute and unmute group notifications', async ({ page }) => {
  await fixture(page, bob);
  await page.goto('/groups/' + groupID);
  await page.getByRole('button', { name: 'Tắt thông báo nhóm' }).click();
  await expect(page.getByText('Thông báo đang tắt.')).toBeVisible();
  await page.reload();
  await page.getByRole('button', { name: 'Bật thông báo nhóm' }).click();
  await expect(page.getByText('Thông báo đang bật.')).toBeVisible();
});
