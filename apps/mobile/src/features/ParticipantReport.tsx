import { useRef, useState } from 'react';
import { useSession } from '../lib/session';
import { Action, Card, Copy, Heading } from '../ui/components';

const reasons = {
  harassment: 'Quấy rối',
  spam: 'Tin rác',
  unsafe: 'Hành vi không an toàn',
  other: 'Lý do khác',
};

export function ParticipantReport({ planId, targetId, name }: { planId: string; targetId: string; name: string }) {
  const { api } = useSession();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<keyof typeof reasons>();
  const [status, setStatus] = useState<'idle' | 'sending' | 'sent' | 'failed'>('idle');
  const inFlight = useRef(false);
  async function submit() {
    if (!reason || inFlight.current) return;
    inFlight.current = true;
    setStatus('sending');
    try {
      await api.request('/reports', { method: 'POST', body: { plan_id: planId, target_id: targetId, reason } });
      setStatus('sent');
      setOpen(false);
    } catch {
      setStatus('failed');
    } finally {
      inFlight.current = false;
    }
  }
  if (status === 'sent') return <Copy announce>Đã gửi báo cáo.</Copy>;
  if (!open) return <Action secondary label={'Báo cáo ' + name} onPress={() => setOpen(true)} />;
  const busy = status === 'sending';
  return <Card>
    <Heading small>Báo cáo {name}</Heading>
    <Copy>Chọn lý do báo cáo.</Copy>
    {(Object.keys(reasons) as (keyof typeof reasons)[]).map(key => <Action key={key} secondary label={reasons[key]} selected={reason === key} disabled={busy} onPress={() => setReason(key)} />)}
    {status === 'failed' && <Copy error>Chưa gửi được báo cáo. Hãy thử lại.</Copy>}
    <Action label="Gửi báo cáo" disabled={!reason || busy} onPress={() => { void submit(); }} />
    <Action secondary label="Hủy báo cáo" disabled={busy} onPress={() => { setOpen(false); setReason(undefined); setStatus('idle'); }} />
  </Card>;
}
