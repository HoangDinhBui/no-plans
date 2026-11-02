export const inviteStorageKey = 'no-plans.pending-invite';
export const activePlanKey = 'no-plans.active-plan';
export type KeyStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>;
const invitePattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.[A-Za-z0-9_-]{43}$/i;
export const validID = (value: unknown): value is string => typeof value === 'string' && /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(value);
export function saved(storage: KeyStorage, key: string): string | null { try { return storage.getItem(key); } catch { return null; } }
export function save(storage: KeyStorage, key: string, value: string): void { try { storage.setItem(key, value); } catch { /* Current in-memory flow still works when storage is blocked. */ } }
export function forget(storage: KeyStorage, key: string): void { try { storage.removeItem(key); } catch { /* Private browsing may deny storage. */ } }

export function captureInvite(storage: KeyStorage, location: Pick<Location, 'hash' | 'pathname' | 'search'>, replace: (url: string) => void): { token: string | null; invalid: boolean } {
  const fragment = new URLSearchParams(location.hash.slice(1));
  const incoming = fragment.has('token');
  const token = incoming ? fragment.get('token') : saved(storage, inviteStorageKey);
  if (incoming) { replace(location.pathname + location.search); forget(storage, activePlanKey); }
  if (token === null && !incoming) return { token: null, invalid: false };
  if (token === null || !invitePattern.test(token)) { forget(storage, inviteStorageKey); return { token: null, invalid: true }; }
  save(storage, inviteStorageKey, token);
  return { token, invalid: false };
}
