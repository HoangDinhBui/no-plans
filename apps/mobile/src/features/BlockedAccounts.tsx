import { useRef, useState } from 'react';
import type { Page, Profile } from '../lib/api';
import { useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import { Action, Card, Copy, Heading, Loading } from '../ui/components';

export function BlockedAccounts() {
  const { api } = useSession();
  const [cursor, setCursor] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState(false);
  const inFlight = useRef(false);
  const list = useResource<Page<Profile>>('/blocks' + (cursor ? '?cursor=' + encodeURIComponent(cursor) : ''));

  async function unblock(id: string) {
    if (inFlight.current) return;
    inFlight.current = true;
    setBusy(true);
    setError(false);
    try {
      await api.request('/blocks/' + encodeURIComponent(id), { method: 'DELETE' });
      if (cursor) setCursor('');
      else list.reload();
    } catch {
      setError(true);
    } finally {
      inFlight.current = false;
      setBusy(false);
    }
  }

  return <Card>
    <Heading small>Người đã chặn</Heading>
    <Copy>Bỏ chặn sẽ cho phép gửi lời mời mới. Hai bạn cần kết bạn lại nếu muốn.</Copy>
    {list.loading && <Loading />}
    {!!list.error && <Copy error>Chưa tải được danh sách chặn.</Copy>}
    {error && <Copy error>Chưa bỏ chặn được. Hãy thử lại.</Copy>}
    {list.data?.items.length === 0 && <Copy>Trang này không có người đã chặn.</Copy>}
    {list.data?.items.map(person => <Action key={person.id} secondary
      label={'Bỏ chặn ' + person.display_name} disabled={busy || list.loading}
      onPress={() => { void unblock(person.id); }} />)}
    {!!list.data?.next_cursor && <Action secondary label="Xem thêm người đã chặn" disabled={busy || list.loading}
      onPress={() => setCursor(list.data!.next_cursor!)} />}
    {!!cursor && <Action secondary label="Về trang đầu danh sách chặn" disabled={busy || list.loading} onPress={() => setCursor('')} />}
    {!!list.error && <Action secondary label="Tải lại danh sách chặn" disabled={busy} onPress={list.reload} />}
  </Card>;
}
