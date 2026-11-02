import type { Friendship, Profile } from './api.ts';
export function recipientID(input: string, self: string): string {
  const id = input.trim().toLowerCase();
  if (!/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/.test(id)) throw new Error('Nhập mã bạn bè hợp lệ từ trang Hồ sơ của người đó.');
  if (id === self.toLowerCase()) throw new Error('Đây là mã của bạn. Hãy nhập mã của người bạn muốn kết nối.');
  return id;
}
export function canAccept(friend: Friendship, actor: string): boolean { return friend.status === 'pending' && friend.requester_id !== actor && [friend.low_id, friend.high_id].includes(actor); }
export function availableMembers(friends: Friendship[], members: Profile[]): Friendship[] { const existing = new Set(members.map(m => m.id)); return friends.filter(f => f.status === 'accepted' && !existing.has(f.peer.id)); }
