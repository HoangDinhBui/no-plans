import { expect, test } from 'vitest';
import { act, render, screen } from '@testing-library/react';
import type { Session, SupabaseClient } from '@supabase/supabase-js';
import userEvent from '@testing-library/user-event';
import { App } from './App';
import { APIClient, APIError } from './api';
import type { Plan } from './api';
import { activePlanKey } from './invite-state';

const token = '00000000-0000-4000-8000-000000000001.' + 'a'.repeat(43);
const api = () => new APIClient('', async () => new Response(JSON.stringify({ title: 'Cà phê cuối tuần', activity: 'coffee', starts_at: '2030-01-01T12:00:00Z' }), { status: 200 }));

test('guest sees minimal invitation and cannot fabricate a sign-in when auth is unconfigured', async () => {
  render(<App api={api()} auth={null} invite={{ token, invalid: false }} />);
  expect(await screen.findByRole('heading', { name: 'Cà phê cuối tuần' })).toBeInTheDocument();
  expect(screen.getByText(/Địa điểm và danh sách bạn bè/)).toBeInTheDocument();
  await userEvent.click(screen.getByRole('button', { name: /Mình tham gia/ }));
  expect(screen.getByRole('alert')).toHaveTextContent('Đăng nhập chưa được cấu hình');
  expect(screen.queryByLabelText('Email')).not.toBeInTheDocument();
});

test('unavailable invite never displays stale private details', async () => {
  const missing = new APIClient('', async () => new Response(JSON.stringify({ code: 'invite_unavailable' }), { status: 404 }));
  render(<App api={missing} auth={null} invite={{ token, invalid: false }} />);
  expect(await screen.findByRole('heading', { name: 'Link mời không còn khả dụng' })).toBeInTheDocument();
  expect(screen.queryByRole('button', { name: /Mình tham gia/ })).not.toBeInTheDocument();
});

test('without an invite the page explains the next action instead of showing sample plans', () => {
  render(<App api={api()} auth={null} invite={{ token: null, invalid: false }} />);
  expect(screen.getByRole('heading', { name: 'Có link mời rồi chứ?' })).toBeInTheDocument();
  expect(screen.queryByText('Cà phê cuối tuần')).not.toBeInTheDocument();
});

test.each(['signed-out', 'revoked'])('a delayed RSVP refresh cannot restore private content after %s', async reason => {
  const userID = '00000000-0000-4000-8000-000000000002';
  const planID = '00000000-0000-4000-8000-000000000003';
  const session = { access_token: 'access', user: { id: userID } } as Session;
  const plan: Plan = { id: planID, owner_id: userID, title: 'Kèo riêng tư', location: 'Địa điểm riêng tư', activity: 'coffee', starts_at: '2030-01-01T12:00:00Z', status: 'inviting', version: 1, members: { [userID]: 'going' } };
  let authChanged: (event: string, session: Session | null) => void = () => {};
  const auth = { auth: { getSession: async () => ({ data: { session } }), onAuthStateChange: (callback: typeof authChanged) => { authChanged = callback; return { data: { subscription: { unsubscribe() {} } } }; } } } as unknown as SupabaseClient;
  let release: (plan: Plan) => void = () => {}; let reads = 0;
  const client = new APIClient();
  client.plan = async () => {
    reads++;
    if (reads === 1) return plan;
    if (reads === 3) throw new APIError('forbidden', 403);
    return new Promise<Plan>(resolve => { release = resolve; });
  };
  client.rsvp = async () => {};
  sessionStorage.setItem(activePlanKey, planID);
  render(<App api={client} auth={auth} invite={{ token: null, invalid: false }} />);
  expect(await screen.findByText('Địa điểm riêng tư')).toBeInTheDocument();
  await userEvent.click(screen.getByRole('button', { name: 'Chưa chắc' }));
  await act(async () => { if (reason === 'signed-out') authChanged('SIGNED_OUT', null); else window.dispatchEvent(new Event('focus')); });
  expect(screen.queryByText('Địa điểm riêng tư')).not.toBeInTheDocument();
  await act(async () => { release(plan); });
  expect(screen.queryByText('Địa điểm riêng tư')).not.toBeInTheDocument();
  sessionStorage.clear();
});
