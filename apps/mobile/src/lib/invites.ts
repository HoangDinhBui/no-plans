import type { InviteLink } from './api.ts';
export function validateInviteBase(base: string, development = false): URL {
  const failure = () => new Error('Chưa tạo được link chia sẻ lúc này. Hãy thử lại sau.');
  let url: URL; try { url = new URL(base); } catch { throw failure(); }
  if ((url.protocol !== 'https:' && !(development && url.protocol === 'http:')) || url.username || url.password || url.search || url.hash) throw failure();
  return url;
}
export function inviteURL(base: string, token: string, development = false): string {
  const url = validateInviteBase(base, development);
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.[A-Za-z0-9_-]{43}$/i.test(token)) throw new Error('Chưa đọc được link lời mời. Hãy thử lại.');
  url.hash = 'token=' + encodeURIComponent(token); return url.toString();
}
export function linkStatus(link: InviteLink, now: number): 'active' | 'full' | 'expired' | 'revoked' {
  if (link.revoked_at) return 'revoked';
  if (!Number.isFinite(Date.parse(link.expires_at)) || Date.parse(link.expires_at) <= now) return 'expired';
  if (link.use_count >= link.max_uses) return 'full';
  return 'active';
}
export function inviteLimit(input: string): number {
  const value = input.trim(); const limit = Number(value);
  if (!/^\d+$/.test(value) || !Number.isSafeInteger(limit) || limit < 1 || limit > 1000) throw new Error('Số lượt cần là số nguyên từ1 đến1000.');
  return limit;
}
