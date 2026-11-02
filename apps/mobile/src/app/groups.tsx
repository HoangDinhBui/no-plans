import { useRef, useState } from 'react';
import { randomUUID } from 'expo-crypto';
import { CreationIntent } from '../lib/api';
import type { Page, Group } from '../lib/api';
import { useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import { Action, Card, Copy, Field, Heading, Loading, Screen } from '../ui/components';
export default function Groups() {
  const { api, session } = useSession(); const [cursor, setCursor] = useState('');
  const groups = useResource<Page<Group>>('/groups' + (cursor ? '?cursor=' + encodeURIComponent(cursor) : ''));
  const [name, setName] = useState(''); const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const intent = useRef(new CreationIntent(randomUUID));
  async function create() { if (busy || !name.trim() || !session) return; setBusy(true); setError(''); const body = { name: name.trim() }; try { await api.request('/groups', { method: 'POST', body, key: intent.current.key(session.user.id, body) }); intent.current.clear(); setName(''); setCursor(''); groups.reload(); } catch (e) { setError(e instanceof Error ? e.message : 'Chưa tạo được nhóm.'); } finally { setBusy(false); } }
  return <Screen><Heading>Hội của bạn.</Heading><Copy>Những người cùng bạn biến lời rủ thành cuộc hẹn.</Copy><Card><Field label="Tên nhóm mới" value={name} onChangeText={setName} maxLength={80} editable={!busy} /><Action label="Tạo nhóm" disabled={busy || !name.trim()} onPress={() => { void create(); }} />{!!error && <Copy error>{error}</Copy>}</Card>{groups.loading && <Loading />}{groups.error && <Copy error>{groups.error.message}</Copy>}{groups.data?.items.map(group => <Card key={group.id}><Heading small>{group.name}</Heading><Copy>{group.owner_id === session?.user.id ? 'Bạn là chủ nhóm' : 'Thành viên'}</Copy></Card>)}{groups.data?.items.length === 0 && <Copy>Chưa có nhóm. Tạo một hội bạn đầu tiên nhé.</Copy>}{!!groups.data?.next_cursor && <Action secondary label="Trang nhóm tiếp" onPress={() => setCursor(groups.data!.next_cursor!)} />}{!!cursor && <Action secondary label="Về trang đầu" onPress={() => setCursor('')} />}<Action secondary label="Tải lại nhóm" onPress={groups.reload} /></Screen>;
}
