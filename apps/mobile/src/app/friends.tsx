import { useRef, useState } from 'react';
import { randomUUID } from 'expo-crypto';
import { CreationIntent } from '../lib/api';
import type { Friendship, Page } from '../lib/api';
import { canAccept, recipientID } from '../lib/social';
import { useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import { Action, Card, Copy, Field, Heading, Loading, Screen } from '../ui/components';
export default function Friends() {
  const { api, session } = useSession(); const actor = session!.user.id;
  const [cursor, setCursor] = useState(''); const list = useResource<Page<Friendship>>('/friend-requests' + (cursor ? '?cursor=' + encodeURIComponent(cursor) : ''));
  const [code, setCode] = useState(''); const [busy, setBusy] = useState(false); const inFlight = useRef(false);
  const [error, setError] = useState(''); const [notice, setNotice] = useState('');
  const intent = useRef(new CreationIntent(randomUUID));
  async function send() {
    if (inFlight.current) return; setError(''); setNotice('');
    try {
      const body = { recipient_id: recipientID(code, actor) }; inFlight.current = true; setBusy(true);
      await api.request('/friend-requests', { method: 'POST', body, key: intent.current.key(actor, body) });
      intent.current.clear(); setCode(''); setNotice('Đã gửi lời mời kết bạn.'); setCursor(''); list.reload();
    } catch (e) { setError(e instanceof Error ? e.message : 'Chưa gửi được lời mời.'); }
    finally { inFlight.current = false; setBusy(false); }
  }
  async function accept(id: string) {
    if (inFlight.current) return; inFlight.current = true; setBusy(true); setError(''); setNotice('');
    try { await api.request('/friend-requests/' + id + '/accept', { method: 'POST' }); setNotice('Đã kết bạn. Giờ có thể thêm bạn vào nhóm.'); list.reload(); }
    catch (e) { setError(e instanceof Error ? e.message : 'Chưa chấp nhận được lời mời.'); list.reload(); }
    finally { inFlight.current = false; setBusy(false); }
  }
  return <Screen><Heading>Hội bạn bắt đầu từ đây.</Heading><Copy selectable>Mã của bạn: {actor}</Copy><Copy>Xin mã bạn bè từ trang Hồ sơ của người bạn muốn kết nối.</Copy>
    <Card><Field label="Mã bạn bè" value={code} onChangeText={setCode} autoCapitalize="none" autoCorrect={false} editable={!busy} /><Action label="Gửi lời mời kết bạn" disabled={busy || !code.trim()} onPress={() => { void send(); }} /></Card>
    {!!error && <Copy error>{error}</Copy>}{!!notice && <Copy>{notice}</Copy>}
    {list.loading && <Loading />}{list.error && <Copy error>{list.error.message}</Copy>}
    {list.data?.items.map(friend => <Card key={friend.id}><Heading small>{friend.peer.display_name}</Heading><Copy>{friend.status === 'accepted' ? 'Đã là bạn bè' : friend.requester_id === actor ? 'Đang chờ bạn ấy chấp nhận' : 'Muốn kết bạn với bạn'}</Copy>{canAccept(friend, actor) && <Action label={'Chấp nhận ' + friend.peer.display_name} disabled={busy} onPress={() => { void accept(friend.id); }} />}</Card>)}
    {list.data?.items.length === 0 && <Copy>Chưa có lời mời hay bạn bè nào. Gửi một lời chào đầu tiên nhé.</Copy>}
    {!!list.data?.next_cursor && <Action secondary label="Trang bạn bè tiếp" disabled={busy} onPress={() => setCursor(list.data!.next_cursor!)} />}
    {!!cursor && <Action secondary label="Về trang bạn bè đầu" disabled={busy} onPress={() => setCursor('')} />}
    <Action secondary label="Tải lại bạn bè" disabled={busy} onPress={list.reload} />
  </Screen>;
}
