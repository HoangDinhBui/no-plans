import { useEffect, useState } from 'react';
import { useLocalSearchParams } from 'expo-router';
import type { Plan, RSVP } from '../../lib/api';
import { APIError } from '../../lib/api';
import { useSession } from '../../lib/session';
import { useResource } from '../../lib/use-resource';
import { OwnerPlan } from '../../features/OwnerPlan';
import type { OwnerChange } from '../../features/OwnerPlan';
import { PlanVoting } from '../../features/PlanVoting';
import { PlanParticipants } from '../../features/PlanParticipants';
import { PlanInviteOwner } from '../../features/PlanInviteOwner';
import { Action, Card, Copy, Heading, Loading, Screen } from '../../ui/components';
export default function PlanDetail() {
  const params = useLocalSearchParams<{ id: string }>(); const id = params.id;
  const { api, session } = useSession(); const resource = useResource<Plan>('/plans/' + encodeURIComponent(id ?? ''), true);
  const [busy, setBusy] = useState(false); const [notice, setNotice] = useState(''); const [error, setError] = useState('');
  const [now, setNow] = useState(Date.now);
  useEffect(() => { const timer = setInterval(() => setNow(Date.now()), 15000); return () => clearInterval(timer); }, []);
  const plan = resource.data; const open = plan && ['inviting', 'confirmed'].includes(plan.status) && Date.parse(plan.starts_at) > now;
  async function respond(response: RSVP) {
    if (busy || !plan) return; setBusy(true); setNotice(''); setError('');
    try { await api.request(`/plans/${plan.id}/rsvp`, { method: 'PUT', body: { response } }); setNotice('Đã lưu phản hồi của bạn.'); resource.reload(); }
    catch (e) { if (e instanceof APIError && [401, 403].includes(e.status)) { resource.reject(e); setError(''); } else { setError(e instanceof Error ? e.message : 'Chưa lưu được.'); resource.reload(); } }
    finally { setBusy(false); }
  }
  async function ownerMutation(change: OwnerChange): Promise<boolean> {
    if (busy || !plan) return false; setBusy(true); setNotice(''); setError('');
    try {
      if (change.action === 'edit') await api.request(`/plans/${plan.id}`, { method: 'PATCH', body: change.body, version: change.version });
      else await api.request(`/plans/${plan.id}/${change.action}`, { method: 'POST' });
      setNotice(change.action === 'edit' ? 'Đã lưu thay đổi.' : change.action === 'confirm' ? 'Đã chốt kèo.' : change.action === 'cancel' ? 'Đã hủy kèo.' : 'Đã kết thúc kèo.');
      resource.reload(); return true;
    } catch (e) {
      if (e instanceof APIError && [401, 403].includes(e.status)) { resource.reject(e); setError(''); }
      else { setError(e instanceof Error ? e.message : 'Chưa thực hiện được thao tác.'); resource.reload(); }
      return false;
    } finally { setBusy(false); }
  }
  return <Screen>{resource.loading && <Loading />}{resource.error && <Card><Copy error>{resource.error.message}</Copy><Action secondary label="Tải lại kèo" onPress={resource.reload} /></Card>}{plan && <><Copy>{plan.status === 'confirmed' ? 'Đã chốt kèo' : plan.status === 'cancelled' ? 'Đã hủy' : plan.status === 'completed' ? 'Đã kết thúc' : 'Đang rủ hội'}</Copy><Heading>{plan.title}</Heading><Card><Heading small>{new Date(plan.starts_at).toLocaleString('vi-VN')}</Heading><Copy>{plan.location || 'Cả nhóm đang chọn địa điểm'}</Copy><Copy>{Object.values(plan.members).filter(v => v === 'going').length} người đi · {Object.keys(plan.members).length} trong kèo</Copy></Card><Heading small>Bạn có đi không?</Heading>{open ? (Object.entries({ going: 'Đi ✓', maybe: 'Chưa chắc', not_going: 'Không đi' }) as [RSVP, string][]).map(([response, label]) => <Action key={response} secondary label={label} selected={plan.members[session!.user.id] === response} disabled={busy} onPress={() => { void respond(response); }} />) : <Copy>Kèo đã bắt đầu, kết thúc hoặc được hủy. Không thể đổi phản hồi.</Copy>}{!!error && <Copy error>{error}</Copy>}{!!notice && <Copy>{notice}</Copy>}<PlanVoting key={"voting:" + plan.id} plan={plan} actor={session!.user.id} now={now} disabled={busy || resource.loading} onChanged={resource.reload} onDenied={resource.reject} /><PlanParticipants key={"members:" + plan.id} plan={plan} actor={session!.user.id} disabled={busy || resource.loading} onChanged={resource.reload} onDenied={resource.reject} />{plan.owner_id === session!.user.id && <PlanInviteOwner key={"invites:" + plan.id} plan={plan} actor={session!.user.id} now={now} disabled={busy || resource.loading} onChanged={resource.reload} onDenied={resource.reject} />}<OwnerPlan key={plan.id} plan={plan} actor={session!.user.id} now={now} busy={busy || resource.loading} mutate={ownerMutation} /><Action secondary label="Tải lại kèo" onPress={resource.reload} /></>}</Screen>;
}
