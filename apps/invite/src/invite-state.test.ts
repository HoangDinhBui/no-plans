import { describe, expect, test } from 'vitest';
import { captureInvite, inviteStorageKey } from './invite-state';

const token = '00000000-0000-4000-8000-000000000001.' + 'a'.repeat(43);
const memory = () => {
  const values = new Map<string, string>();
  return { getItem: (key: string) => values.get(key) ?? null, setItem: (key: string, value: string) => { values.set(key, value); }, removeItem: (key: string) => { values.delete(key); } };
};

describe('private invite continuation', () => {
  test('captures fragment token, strips it from history and survives same-tab auth/reload', () => {
    const storage = memory(); let replaced = '';
    const result = captureInvite(storage, { hash: '#token=' + token, pathname: '/', search: '?code=auth-code' }, (url) => { replaced = url; });
    expect(result).toEqual({ token, invalid: false });
    expect(replaced).toBe('/?code=auth-code');
    expect(storage.getItem(inviteStorageKey)).toBe(token);
    expect(captureInvite(storage, { hash: '', pathname: '/', search: '' }, () => {})).toEqual({ token, invalid: false });
  });
  test('a malformed new link never falls back to a different saved invite', () => {
    const storage = memory(); storage.setItem(inviteStorageKey, token);
    expect(captureInvite(storage, { hash: '#token=bad', pathname: '/', search: '' }, () => {})).toEqual({ token: null, invalid: true });
    expect(storage.getItem(inviteStorageKey)).toBeNull();
  });
  test('blocked browser storage does not break the current invite', () => {
    const blocked = { getItem() { throw Error('blocked'); }, setItem() { throw Error('blocked'); }, removeItem() { throw Error('blocked'); } };
    expect(captureInvite(blocked, { hash: '#token=' + token, pathname: '/', search: '' }, () => {}).token).toBe(token);
  });
});
