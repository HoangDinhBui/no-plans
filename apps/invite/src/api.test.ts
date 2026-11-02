import { expect, test } from 'vitest';
import { APIClient, APIError } from './api';

test('preview sends invite secret only as a header and validates minimal data', async () => {
  const client = new APIClient('https://api.example', async (input, init) => {
    expect(String(input)).toBe('https://api.example/v1/invite-preview');
    expect(new Headers(init?.headers).get('X-Invite-Token')).toBe('private-token');
    return new Response(JSON.stringify({ title: 'Coffee', activity: 'coffee', starts_at: '2030-01-01T12:00:00Z' }), { status: 200 });
  });
  expect((await client.preview('private-token')).title).toBe('Coffee');
});

test('unavailable links and malformed data become safe user-facing errors', async () => {
  const unavailable = new APIClient('', async () => new Response(JSON.stringify({ code: 'invite_unavailable', request_id: 'r' }), { status: 404 }));
  await expect(unavailable.preview('token')).rejects.toMatchObject({ code: 'invite_unavailable', status: 404 });
  const malformed = new APIClient('', async () => new Response(JSON.stringify({ title: 'Coffee', starts_at: 'not-a-date' }), { status: 200 }));
  await expect(malformed.preview('token')).rejects.toBeInstanceOf(APIError);
});

test('RSVP uses the authenticated actor and never supplies a user ID', async () => {
  const client = new APIClient('', async (input, init) => {
    expect(String(input)).toBe('/v1/plans/00000000-0000-4000-8000-000000000001/rsvp');
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer access');
    expect(JSON.parse(String(init?.body))).toEqual({ response: 'maybe' });
    return new Response(JSON.stringify({ changed: true }), { status: 200 });
  });
  await expect(client.rsvp('00000000-0000-4000-8000-000000000001', 'maybe', 'access')).resolves.toBeUndefined();
});
