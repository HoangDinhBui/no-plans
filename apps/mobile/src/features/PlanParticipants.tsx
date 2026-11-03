import { ParticipantReport } from './ParticipantReport';
import { useEffect, useRef, useState } from 'react';
import type { Page, Participant, Plan } from '../lib/api';
import { useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import { accessFailure, useAccessFailure } from '../lib/use-access-failure';
import { Action, Card, Copy, Heading, Loading } from '../ui/components';
const responses: Record<string, string> = { invited: 'Chưa phản hồi', going: 'Đi', maybe: 'Chưa chắc', not_going: 'Không đi' };
export function PlanParticipants({ plan, actor, disabled, onChanged, onDenied }: { plan: Plan; actor: string; disabled: boolean; onChanged: () => void; onDenied: (e: Error) => void }) {
  const { api } = useSession(); const [cursor, setCursor] = useState('');
  const members = useResource<Page<Participant>>(`/plans/${plan.id}/members${cursor ? '?cursor=' + encodeURIComponent(cursor) : ''}`, true, plan.version);
  useAccessFailure(members.error, onDenied);
  const [target, setTarget] = useState<Participant | null>(null); const [busy, setBusy] = useState(false); const [error, setError] = useState('');
  const mounted = useRef(false); const inFlight = useRef(false);
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const owner = plan.owner_id === actor; const active = ['inviting', 'confirmed'].includes(plan.status);
  async function revoke() {
    if (!target || inFlight.current || disabled) return; inFlight.current = true; setBusy(true); setError('');
    try { await api.request(`/plans/${plan.id}/invitations/${target.id}`, { method: 'DELETE' }); if (mounted.current) { setTarget(null); members.reload(); onChanged(); } }
    catch (e) { if (mounted.current) { if (accessFailure(e)) onDenied(e); else setError(e instanceof Error ? e.message : 'Chưa thu hồi được lời mời.'); } }
    finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }
  return <Card><Heading small>Hội mình có ai?</Heading>{members.loading && <Loading />}{members.error && <Copy error>{members.error.message}</Copy>}{!!error && <Copy error>{error}</Copy>}
    {members.data?.items.map(member => <Card key={member.id}><Heading small>{member.display_name}</Heading><Copy>{responses[member.response]}{member.id === plan.owner_id ? ' · Chủ kèo' : member.id === actor ? ' · Bạn' : ''}</Copy>{owner && active && member.id !== actor && <Action secondary label={'Thu hồi lời mời của ' + member.display_name} disabled={busy || disabled || members.loading} onPress={() => setTarget(member)} />}{member.id !== actor && <ParticipantReport planId={plan.id} targetId={member.id} name={member.display_name} />}</Card>)}
    {target && owner && active && members.data?.items.some(m => m.id === target.id) && <Card><Heading small>Thu hồi quyền vào kèo?</Heading><Copy>{target.display_name} sẽ không còn xem hay phản hồi kèo. Link cũ không cho người này tự vào lại; bạn có thể mời trực tiếp lại sau.</Copy><Action label="Xác nhận thu hồi thành viên" disabled={busy || disabled} onPress={() => { void revoke(); }} /><Action secondary label="Giữ lời mời" disabled={busy} onPress={() => setTarget(null)} /></Card>}
    {!!members.data?.next_cursor && <Action secondary label="Trang người tham gia tiếp" disabled={busy} onPress={() => { setTarget(null); setCursor(members.data!.next_cursor!); }} />}
    {!!cursor && <Action secondary label="Về trang người tham gia đầu" disabled={busy} onPress={() => { setTarget(null); setCursor(''); }} />}
    <Action secondary label="Tải lại người tham gia" disabled={busy} onPress={members.reload} />
  </Card>;
}
