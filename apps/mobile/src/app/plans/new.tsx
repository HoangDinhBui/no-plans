import { useEffect, useRef, useState } from 'react';
import { router } from 'expo-router';
import type { Page, Group, Plan } from '../../lib/api';
import { planInput } from '../../lib/plan-input';
import type { PlanDraft } from '../../lib/plan-input';
import { emptyDraft } from '../../lib/drafts';
import { planDrafts } from '../../lib/plan-drafts';
import { useResource } from '../../lib/use-resource';
import { useSession } from '../../lib/session';
import { Action, Card, Copy, Field, Heading, Loading, Screen } from '../../ui/components';

const activities = { food: 'Ăn uống', coffee: 'Cà phê', drinks: 'Đi uống', cinema: 'Xem phim', sports: 'Thể thao', other: 'Đi chơi' };
export default function NewPlan() {
  const { api, session } = useSession(); const actor = session!.user.id;
  const [groupCursor, setGroupCursor] = useState('');
  const groups = useResource<Page<Group>>('/groups?limit=100' + (groupCursor ? '&cursor=' + encodeURIComponent(groupCursor) : ''));
  const [draft, setDraft] = useState<PlanDraft>(emptyDraft);
  const [ready, setReady] = useState(false); const [busy, setBusy] = useState(false);
  const [error, setError] = useState(''); const [saveError, setSaveError] = useState('');
  const [confirmDiscard, setConfirmDiscard] = useState(false); const [createdID, setCreatedID] = useState('');
  const [restore, setRestore] = useState(0);
  const mounted = useRef(false);
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, [actor]);
  useEffect(() => {
    let alive = true;
    void planDrafts.load(actor).then(record => { if (alive) { setDraft(record.draft); setReady(true); setSaveError(''); } }).catch(() => { if (alive) setSaveError('Chưa đọc được bản nháp. Thử tải lại hoặc kiểm tra kèo đã tạo trước khi xóa bản nháp.'); });
    return () => { alive = false; };
  }, [actor, restore]);
  const update = (key: keyof PlanDraft, value: string) => {
    const next = { ...draft, [key]: value }; setDraft(next);
    void planDrafts.saveDraft(actor, next).then(() => setSaveError('')).catch(() => setSaveError('Chưa lưu được bản nháp trên thiết bị. Giữ màn hình này và thử lại.'));
  };
  function open(id: string) { router.replace({ pathname: '/plans/[id]', params: { id } }); }
  async function create() {
    if (busy || !ready || createdID) return; setError('');
    try {
      const body = planInput(draft); const scope = api.captureScope(); setBusy(true);
      const key = await planDrafts.prepare(actor, draft, body);
      if (!mounted.current) return;
      const result = await api.request<Plan>('/plans', { method: 'POST', body, key, scope });
      if (!mounted.current) return;
      setCreatedID(result.id);
      try { await planDrafts.complete(actor, key); if (mounted.current && scope === api.captureScope()) open(result.id); }
      catch { setSaveError('Kèo đã được tạo. Chưa xóa được bản nháp trên thiết bị; bạn có thể mở kèo bên dưới.'); }
    } catch (e) { setError(e instanceof Error ? e.message : 'Chưa tạo được kèo.'); }
    finally { setBusy(false); }
  }
  async function discard() {
    setBusy(true);
    try { await planDrafts.discard(actor); setDraft(emptyDraft()); setReady(true); setSaveError(''); setError(''); setConfirmDiscard(false); setCreatedID(''); }
    catch { setSaveError('Chưa xóa được bản nháp. Hãy thử lại.'); }
    finally { setBusy(false); }
  }
  const disabled = busy || !ready || !!createdID;
  return <Screen>
    <Heading>Hội mình làm gì?</Heading>
    {!ready && !saveError && <Loading />}
    {!!saveError && <Card><Copy error>{saveError}</Copy>{!ready && <Action secondary label="Tải lại bản nháp" onPress={() => setRestore(n => n + 1)} />}</Card>}
    {!!createdID && <Action label="Mở kèo đã tạo" onPress={() => open(createdID)} />}
    <Field label="Tên kèo" value={draft.title} onChangeText={v => update('title', v)} maxLength={120} editable={!disabled} />
    <Card><Heading small>Hoạt động</Heading>{Object.entries(activities).map(([key, label]) => <Action key={key} secondary label={label} selected={draft.activity === key} disabled={disabled} onPress={() => update('activity', key)} />)}</Card>
    <Field label="Ngày (YYYY-MM-DD)" placeholder="2030-01-01" value={draft.date} onChangeText={v => update('date', v)} editable={!disabled} />
    <Field label="Giờ Việt Nam (HH:mm)" value={draft.time} onChangeText={v => update('time', v)} editable={!disabled} />
    <Card><Heading small>Rủ nhóm</Heading>
      <Action secondary label="Chỉ mình tôi, mời thêm sau" selected={!draft.group_id} disabled={disabled} onPress={() => update('group_id', '')} />
      {groups.data?.items.map(group => <Action key={group.id} secondary label={group.name} selected={draft.group_id === group.id} disabled={disabled} onPress={() => update('group_id', group.id)} />)}
      {!!draft.group_id && !groups.data?.items.some(g => g.id === draft.group_id) && <Copy>Đang giữ nhóm đã chọn từ bản nháp hoặc trang trước.</Copy>}
      {groups.error && <><Copy error>{groups.error.message}</Copy><Action secondary label="Tải lại nhóm" onPress={groups.reload} /></>}
      {!!groups.data?.next_cursor && <Action secondary label="Xem thêm nhóm" disabled={disabled} onPress={() => setGroupCursor(groups.data!.next_cursor!)} />}
      {!!groupCursor && <Action secondary label="Về trang nhóm đầu" disabled={disabled} onPress={() => setGroupCursor('')} />}
    </Card>
    <Field label="Địa điểm (có thể chọn sau)" value={draft.location} onChangeText={v => update('location', v)} maxLength={500} editable={!disabled} />
    <Field label="Giới hạn người (để trống nếu không giới hạn)" value={draft.capacity} onChangeText={v => update('capacity', v)} keyboardType="number-pad" editable={!disabled} />
    {!!error && <Copy error>{error}</Copy>}
    <Action label={busy ? 'Đang tạo…' : 'Tạo kèo ↗'} disabled={disabled} onPress={() => { void create(); }} />
    <Copy>Bản nháp lưu riêng cho tài khoản này trên thiết bị. Nếu mất mạng, giữ nguyên thông tin và thử lại.</Copy>
    {confirmDiscard ? <Card><Copy>Xóa bản nháp không hủy kèo đã tạo. Nếu lần gửi trước mất kết nối, hãy kiểm tra danh sách kèo trước để tránh tạo trùng.</Copy><Action secondary label="Về danh sách kiểm tra" disabled={busy} onPress={() => router.replace('/')} /><Action label="Xóa bản nháp trên thiết bị" disabled={busy} onPress={() => { void discard(); }} /><Action secondary label="Giữ bản nháp" disabled={busy} onPress={() => setConfirmDiscard(false)} /></Card> : <Action secondary label="Bỏ bản nháp" disabled={busy} onPress={() => setConfirmDiscard(true)} />}
  </Screen>;
}
