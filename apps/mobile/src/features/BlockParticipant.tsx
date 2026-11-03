import { useRef, useState } from 'react';
import { useSession } from '../lib/session';
import { Action, Card, Copy, Heading } from '../ui/components';

export function BlockParticipant({ id, name }: { id: string; name: string }) {
  const { api } = useSession();
  const [confirm, setConfirm] = useState(false);
  const [state, setState] = useState<'idle' | 'saving' | 'blocked' | 'error'>('idle');
  const saving = useRef(false);

  async function block() {
    if (saving.current) return;
    saving.current = true;
    setState('saving');
    try {
      await api.request('/blocks', { method: 'POST', body: { user_id: id } });
      setState('blocked');
    } catch {
      setState('error');
    } finally {
      saving.current = false;
    }
  }

  if (state === 'blocked') return <Copy announce>Đã chặn {name}.</Copy>;
  if (!confirm) return <Action secondary label={'Chặn ' + name} onPress={() => setConfirm(true)} />;
  return <Card>
    <Heading small>Chặn {name}?</Heading>
    <Copy>Quan hệ kết bạn sẽ bị xóa và hai bạn sẽ không thể gửi lời mời mới cho nhau.</Copy>
    {state === 'error' && <Copy error>Chưa chặn được. Hãy thử lại.</Copy>}
    <Action label="Xác nhận chặn" disabled={state === 'saving'} onPress={() => { void block(); }} />
    <Action secondary label="Hủy chặn" disabled={state === 'saving'} onPress={() => setConfirm(false)} />
  </Card>;
}
