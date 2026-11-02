import type { Plan } from './api.ts';
import { planDetails, vietnamTime } from './plan-input.ts';
import type { PlanDraft } from './plan-input.ts';
export type OwnerAction = 'edit' | 'confirm' | 'cancel' | 'complete';
export function ownerActions(plan: Plan, actor: string, now: Date): OwnerAction[] {
  if (actor !== plan.owner_id || ['completed', 'cancelled'].includes(plan.status)) return [];
  const future = Date.parse(plan.starts_at) > now.getTime();
  const actions: OwnerAction[] = [];
  if (future) actions.push('edit');
  if (future && plan.status === 'inviting' && plan.location.trim()) actions.push('confirm');
  if (!future && plan.status === 'confirmed') actions.push('complete');
  actions.push('cancel'); return actions;
}
export function editDraft(plan: Plan): PlanDraft {
  const local = new Date(Date.parse(plan.starts_at) + 7 * 3600000).toISOString();
  return { title: plan.title, activity: plan.activity, date: local.slice(0, 10), time: local.slice(11, 16), capacity: plan.capacity ? String(plan.capacity) : '', location: plan.location, group_id: '' };
}
export function editPatch(plan: Plan, draft: PlanDraft, now = new Date()) {
  const fields = planDetails(draft); const original = editDraft(plan);
  const startsAt = draft.date === original.date && draft.time === original.time ? new Date(plan.starts_at) : vietnamTime(draft.date, draft.time);
  if (startsAt <= now) throw new Error('Ngày giờ phải ở tương lai.');
  if (fields.capacity > 0 && fields.capacity < Object.values(plan.members).filter(v => v === 'going').length) throw new Error('Sức chứa không thể thấp hơn số người đang đi.');
  if (plan.status === 'confirmed' && !fields.location) throw new Error('Kèo đã chốt cần địa điểm cụ thể.');
  const patch: { title?: string; starts_at?: string; capacity?: number; location?: string } = {};
  if (fields.title !== plan.title) patch.title = fields.title;
  if (startsAt.getTime() !== Date.parse(plan.starts_at)) patch.starts_at = startsAt.toISOString();
  if (fields.capacity !== plan.capacity) patch.capacity = fields.capacity;
  if (fields.location !== plan.location) patch.location = fields.location;
  return patch;
}
