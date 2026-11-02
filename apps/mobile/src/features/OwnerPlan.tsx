import { useState } from 'react';
import type { Plan } from '../lib/api';
import type { PlanDraft } from '../lib/plan-input';
import { editDraft, editPatch, ownerActions } from '../lib/owner-plan';
import type { OwnerAction } from '../lib/owner-plan';
import { Action, Card, Copy, Field, Heading } from '../ui/components';
type Change = { action: 'confirm' | 'cancel' | 'complete' } | { action: 'edit'; body: ReturnType<typeof editPatch>; version: number };
export function OwnerPlan({ plan, actor, now, busy, mutate }: { plan: Plan; actor: string; now: number; busy: boolean; mutate: (change: Change) => Promise<boolean> }) {
  const [editing, setEditing] = useState<{ original: Plan; draft: PlanDraft } | null>(null);
  const [confirm, setConfirm] = useState<OwnerAction | null>(null);
  const [error, setError] = useState('');
  const actions = ownerActions(plan, actor, new Date(now));
  if (!actions.length) return null;
  const update = (field: keyof PlanDraft, value: string) => setEditing(old => old ? { ...old, draft: { ...old.draft, [field]: value } } : null);
  async function save() {
    if (!editing || busy) return; setError('');
    try {
      const body = editPatch(editing.original, editing.draft);
      if (!Object.keys(body).length) { setEditing(null); return; }
      if (await mutate({ action: 'edit', body, version: editing.original.version })) setEditing(null);
    } catch (e) { setError(e instanceof Error ? e.message : 'Kiểm tra lại thông tin.'); }
  }
  return <Card><Heading small>Chủ kèo</Heading>
    {editing ? <>
      {editing.original.version !== plan.version && <Copy>Kèo đã thay đổi. Bản sửa của bạn vẫn được giữ; tải thông tin mới để kiểm tra trước khi lưu.</Copy>}
      <Field label="Sửa tên kèo" value={editing.draft.title} maxLength={120} editable={!busy} onChangeText={v => update('title', v)} />
      <Field label="Sửa ngày (YYYY-MM-DD)" value={editing.draft.date} editable={!busy} onChangeText={v => update('date', v)} />
      <Field label="Sửa giờ Việt Nam (HH:mm)" value={editing.draft.time} editable={!busy} onChangeText={v => update('time', v)} />
      <Field label="Sửa địa điểm" value={editing.draft.location} maxLength={500} editable={!busy} onChangeText={v => update('location', v)} />
      <Field label="Sửa giới hạn người (trống nếu không giới hạn)" value={editing.draft.capacity} keyboardType="number-pad" editable={!busy} onChangeText={v => update('capacity', v)} />
      {!!error && <Copy error>{error}</Copy>}
      <Action label="Lưu thay đổi" disabled={busy} onPress={() => { void save(); }} />
      {editing.original.version !== plan.version && <Action secondary label="Bỏ bản sửa, lấy thông tin mới" disabled={busy} onPress={() => { setEditing({ original: plan, draft: editDraft(plan) }); setError(''); }} />}
      <Action secondary label="Đóng phần sửa" disabled={busy} onPress={() => { setEditing(null); setError(''); }} />
    </> : <>
      {actions.includes('edit') && <Action secondary label="Sửa kèo" disabled={busy} onPress={() => setEditing({ original: plan, draft: editDraft(plan) })} />}
      {actions.includes('confirm') && <Action label="Chốt kèo" disabled={busy} onPress={() => { void mutate({ action: 'confirm' }); }} />}
      {plan.status === 'inviting' && !plan.location.trim() && <Copy>Thêm địa điểm cụ thể trước khi chốt kèo.</Copy>}
      {actions.includes('complete') && <Action label="Kết thúc kèo" disabled={busy} onPress={() => setConfirm('complete')} />}
      {actions.includes('cancel') && <Action secondary label="Hủy kèo" disabled={busy} onPress={() => setConfirm('cancel')} />}
    </>}
    {confirm && <Card><Heading small>{confirm === 'cancel' ? 'Hủy cuộc hẹn này?' : 'Hội mình đã gặp nhau rồi?'}</Heading><Copy>{confirm === 'cancel' ? 'Cả nhóm sẽ thấy kèo đã hủy và không thể phản hồi thêm.' : 'Kèo sẽ chuyển vào lịch sử và không nhận thay đổi nữa.'}</Copy><Action label={confirm === 'cancel' ? 'Xác nhận hủy kèo' : 'Xác nhận kết thúc'} disabled={busy} onPress={() => { void mutate({ action: confirm === 'cancel' ? 'cancel' : 'complete' }).then(ok => { if (ok) setConfirm(null); }); }} /><Action secondary label="Giữ nguyên kèo" disabled={busy} onPress={() => setConfirm(null)} /></Card>}
  </Card>;
}
export type OwnerChange = Change;
