import test from 'node:test';
import assert from 'node:assert/strict';
import { editDraft, editPatch, ownerActions } from './owner-plan.ts';
import type { Plan } from './api.ts';
const now = new Date('2029-01-01T00:00:00Z');
const plan: Plan = { id: 'plan', owner_id: 'owner', title: 'Coffee', activity: 'coffee', starts_at: '2030-01-01T12:30:45.123Z', location: 'Quán', capacity: 5, status: 'inviting', version: 3, members: { owner: 'going', friend: 'going' } };
test('owner actions respect role, schedule, location and terminal states', () => {
  assert.deepEqual(ownerActions(plan, 'stranger', now), []);
  assert.deepEqual(ownerActions(plan, 'owner', now), ['edit', 'confirm', 'cancel']);
  assert.deepEqual(ownerActions({ ...plan, location: '' }, 'owner', now), ['edit', 'cancel']);
  assert.deepEqual(ownerActions({ ...plan, status: 'confirmed', starts_at: '2020-01-01T00:00:00Z' }, 'owner', now), ['complete', 'cancel']);
  for (const status of ['completed', 'cancelled'] as const) assert.deepEqual(ownerActions({ ...plan, status }, 'owner', now), []);
});
test('edit uses Vietnam calendar, preserves untouched timestamp precision and sends only changes', () => {
  const draft = editDraft(plan);
  assert.equal(draft.time, '19:30'); assert.equal(draft.date, '2030-01-01');
  assert.deepEqual(editPatch(plan, { ...draft, title: ' New name ' }, now), { title: 'New name' });
  assert.deepEqual(editPatch(plan, { ...draft, time: '20:30' }, now), { starts_at: '2030-01-01T13:30:00.000Z' });
  assert.deepEqual(editPatch(plan, draft, now), {});
});
test('editing rejects invalid date, capacity below attendance and empty confirmed location', () => {
  assert.throws(() => editPatch(plan, { ...editDraft(plan), date: '2030-02-30' }, now));
  assert.throws(() => editPatch(plan, { ...editDraft(plan), capacity: '1' }, now));
  const confirmed = { ...plan, status: 'confirmed' as const };
  assert.throws(() => editPatch(confirmed, { ...editDraft(confirmed), location: '' }, now));
});
