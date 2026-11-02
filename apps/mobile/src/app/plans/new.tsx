import { useRef, useState } from 'react';
import { router } from 'expo-router';
import { randomUUID } from 'expo-crypto';
import { CreationIntent } from '../../lib/api';
import type { Page, Group, Plan } from '../../lib/api';
import { planInput } from '../../lib/plan-input';
import type { PlanDraft } from '../../lib/plan-input';
import { useResource } from '../../lib/use-resource';
import { useSession } from '../../lib/session';
import { Action, Card, Copy, Field, Heading, Screen } from '../../ui/components';
const activities = { food: 'Ăn uống', coffee: 'Cà phê', drinks: 'Đi uống', cinema: 'Xem phim', sports: 'Thể thao', other: 'Đi chơi' };
export default function NewPlan() {
  const { api, session } = useSession(); const groups = useResource<Page<Group>>('/groups?limit=100');
  const [draft, setDraft] = useState<PlanDraft>({ title: '', activity: 'coffee', date: '', time: '19:30', location: '', capacity: '', group_id: '' });
  const intent = useRef(new CreationIntent(randomUUID)); const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  const update = (key: keyof PlanDraft, value: string) => setDraft(old => ({ ...old, [key]: value }));
  async function create() {
    if (busy || !session) return; setError('');
    try { const body = planInput(draft); setBusy(true); const result = await api.request<Plan>('/plans', { method: 'POST', body, key: intent.current.key(session.user.id, body) }); intent.current.clear(); router.replace({ pathname: '/plans/[id]', params: { id: result.id } }); }
    catch (e) { setError(e instanceof Error ? e.message : 'Chưa tạo được kèo.'); }
    finally { setBusy(false); }
  }
  return <Screen><Heading>Hội mình làm gì?</Heading><Field label="Tên kèo" value={draft.title} onChangeText={v => update('title', v)} maxLength={120} editable={!busy} /><Card><Heading small>Hoạt động</Heading>{Object.entries(activities).map(([key, label]) => <Action key={key} secondary label={label} selected={draft.activity === key} disabled={busy} onPress={() => update('activity', key)} />)}</Card><Field label="Ngày (YYYY-MM-DD)" placeholder="2030-01-01" value={draft.date} onChangeText={v => update('date', v)} editable={!busy} /><Field label="Giờ Việt Nam (HH:mm)" value={draft.time} onChangeText={v => update('time', v)} editable={!busy} /><Card><Heading small>Rủ nhóm</Heading><Action secondary label="Chỉ mình tôi, mời thêm sau" selected={!draft.group_id} disabled={busy} onPress={() => update('group_id', '')} />{groups.data?.items.map(group => <Action key={group.id} secondary label={group.name} selected={draft.group_id === group.id} disabled={busy} onPress={() => update('group_id', group.id)} />)}{groups.error && <><Copy error>{groups.error.message}</Copy><Action secondary label="Tải lại nhóm" onPress={groups.reload} /></>}{groups.data?.next_cursor && <Copy>Danh sách đang hiển thị100 nhóm đầu; tìm nhóm khác sẽ được bổ sung.</Copy>}</Card><Field label="Địa điểm (có thể chọn sau)" value={draft.location} onChangeText={v => update('location', v)} maxLength={500} editable={!busy} /><Field label="Giới hạn người (để trống nếu không giới hạn)" value={draft.capacity} onChangeText={v => update('capacity', v)} keyboardType="number-pad" editable={!busy} />{error && <Copy error>{error}</Copy>}<Action label={busy ? 'Đang tạo…' : 'Tạo kèo ↗'} disabled={busy} onPress={() => { void create(); }} /><Copy>Nếu mất mạng, giữ nguyên thông tin và thử lại để tránh tạo trùng.</Copy></Screen>;
}
