import test from 'node:test';
import assert from 'node:assert/strict';
import { recipientID, canAccept, availableMembers } from './social.ts';
const self = '00000000-0000-4000-8000-000000000001';
const peer = '00000000-0000-4000-8000-000000000002';
const incoming = { id: 'request', low_id: self, high_id: peer, requester_id: peer, status: 'pending' as const, peer: { id: peer, display_name: 'Bạn' } };
test('friend code normalization rejects malformed IDs and self requests', () => {
  assert.equal(recipientID(' ' + peer.toUpperCase() + ' ', self), peer);
  assert.throws(() => recipientID('email@example.com', self)); assert.throws(() => recipientID(self, self));
});
test('only recipient of a pending request can accept', () => {
  assert.equal(canAccept(incoming, self), true);
  assert.equal(canAccept({ ...incoming, requester_id: self }, self), false);
  assert.equal(canAccept({ ...incoming, status: 'accepted' }, self), false);
  assert.equal(canAccept(incoming, 'stranger'), false);
});
test('group invitations offer accepted friends outside current membership only', () => {
  assert.deepEqual(availableMembers([incoming], []), []);
  const accepted = { ...incoming, status: 'accepted' as const };
  assert.equal(availableMembers([accepted], []).length, 1);
  assert.deepEqual(availableMembers([accepted], [accepted.peer]), []);
});
