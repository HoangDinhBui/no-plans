export type PlanDraft = { title: string; activity: string; date: string; time: string; capacity: string; location: string; group_id: string };
export function planDetails(draft: PlanDraft) {
  const title = draft.title.trim(); const location = draft.location.trim();
  if (!title || [...title].length > 120 || [...location].length > 500) throw new Error('Tên cần1–120 ký tự; địa điểm tối đa500 ký tự.');
  if (!['food', 'coffee', 'drinks', 'cinema', 'sports', 'other'].includes(draft.activity)) throw new Error('Chọn loại hoạt động hợp lệ.');
  const capacity = draft.capacity.trim() ? Number(draft.capacity) : 0;
  if ((draft.capacity.trim() && !/^\d+$/.test(draft.capacity.trim())) || !Number.isSafeInteger(capacity) || capacity < 0 || capacity > 2147483647 || (draft.capacity.trim() && capacity === 0)) throw new Error('Sức chứa cần số nguyên từ1 trở lên, tính cả bạn; để trống nếu không giới hạn.');
  if (draft.group_id && !/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(draft.group_id)) throw new Error('Chọn nhóm hợp lệ.');
  return { title, activity: draft.activity, capacity, location, ...(draft.group_id ? { group_id: draft.group_id } : {}) };
}
export function vietnamTime(date: string, time: string): Date {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(date) || !/^\d{2}:\d{2}$/.test(time)) throw new Error('Nhập ngày YYYY-MM-DD và giờ HH:mm theo giờ Việt Nam.');
  const at = new Date(`${date}T${time}:00+07:00`);
  const local = new Date(at.getTime() + 7 * 3600000);
  if (!Number.isFinite(at.getTime()) || local.toISOString().slice(0, 16) !== `${date}T${time}`) throw new Error('Ngày giờ phải hợp lệ và ở tương lai.');
  return at;
}
export function planInput(draft: PlanDraft, now = new Date()) {
  const details = planDetails(draft); const at = vietnamTime(draft.date, draft.time);
  if (at <= now) throw new Error('Ngày giờ phải hợp lệ và ở tương lai.');
  return { ...details, starts_at: at.toISOString() };
}
