import { useEffect, useRef, useState } from 'react';
import { router, useLocalSearchParams } from 'expo-router';
import { APIError } from '../../lib/api';
import type { Friendship, GroupDetail, Page, Profile } from '../../lib/api';
import { availableMembers } from '../../lib/social';
import { useSession } from '../../lib/session';
import { useResource } from '../../lib/use-resource';
import { Action, Card, Copy, Heading, Loading, Screen } from '../../ui/components';
type Confirmation = { action: 'remove' | 'transfer'; member: Profile } | { action: 'leave' };
function GroupScreen({ id }: { id: string }) {
  const { api, session } = useSession(); const actor = session!.user.id;
  const group = useResource<GroupDetail>('/groups/' + encodeURIComponent(id ?? ''), true);
  const [cursor, setCursor] = useState(''); const friends = useResource<Page<Friendship>>('/friend-requests' + (cursor ? '?cursor=' + encodeURIComponent(cursor) : ''));
  const [busy, setBusy] = useState(false); const inFlight = useRef(false); const mounted = useRef(false);
  const [confirm, setConfirm] = useState<Confirmation | null>(null); const [error, setError] = useState(''); const [notice, setNotice] = useState('');
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, [id, actor]);
  const data = group.data; const owner = data?.owner_id === actor;
  const confirmationAllowed = confirm && data && (confirm.action === 'leave' ? !owner : owner && data.members.some(member => member.id === confirm.member.id && member.id !== actor));
  async function mutate(action: 'add' | 'remove' | 'transfer' | 'leave', target: string) {
    if (inFlight.current || !data) return; inFlight.current = true; setBusy(true); setError(''); setNotice('');
    try {
      if (action === 'transfer') await api.request('/groups/' + data.id + '/owner', { method: 'PUT', body: { user_id: target } });
      else await api.request('/groups/' + data.id + '/members/' + target, { method: action === 'add' ? 'POST' : 'DELETE' });
      if (!mounted.current) return;
      setConfirm(null);
      if (action === 'leave') { router.replace('/groups'); return; }
      setNotice(action === 'add' ? 'Đã thêm bạn vào nhóm.' : action === 'remove' ? 'Đã xóa thành viên khỏi nhóm.' : 'Đã chuyển quyền chủ nhóm.'); group.reload();
    } catch (e) {
      if (!mounted.current) return;
      if (e instanceof APIError && (e.status === 401 || (e.status === 403 && e.code === 'forbidden'))) { group.reject(e); setConfirm(null); }
      else { setError(e instanceof Error ? e.message : 'Chưa thực hiện được thao tác.'); group.reload(); }
    } finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }
  return <Screen>{group.loading && <Loading />}{group.error && <Card><Copy error>{group.error.message}</Copy><Action secondary label="Thử tải lại nhóm" onPress={group.reload} /></Card>}
    {data && <><Heading>{data.name}</Heading><Copy>{owner ? 'Bạn đang quản lý nhóm này.' : 'Hội mình ở đây.'}</Copy>{!!error && <Copy error>{error}</Copy>}{!!notice && <Copy>{notice}</Copy>}
      <Heading small>Thành viên</Heading>{data.members.map(member => <Card key={member.id}><Heading small>{member.display_name}</Heading><Copy>{member.id === data.owner_id ? 'Chủ nhóm' : member.id === actor ? 'Bạn' : 'Thành viên'}</Copy>
        {owner && member.id !== actor && <><Action secondary label={'Chuyển quyền cho ' + member.display_name} disabled={busy || group.loading} onPress={() => setConfirm({ action: 'transfer', member })} /><Action secondary label={'Xóa ' + member.display_name + ' khỏi nhóm'} disabled={busy || group.loading} onPress={() => setConfirm({ action: 'remove', member })} /></>}
      </Card>)}
      {owner ? <Card><Heading small>Rủ thêm bạn</Heading><Copy>Chỉ thêm người đã chấp nhận kết bạn. Người vào sau không tự được mời vào kèo cũ.</Copy>
        {friends.loading && <Loading />}{friends.error && <Copy error>{friends.error.message}</Copy>}
        {availableMembers(friends.data?.items ?? [], data.members).map(friend => <Action key={friend.id} secondary label={'Thêm ' + friend.peer.display_name} disabled={busy || group.loading} onPress={() => { void mutate('add', friend.peer.id); }} />)}
        {friends.data && availableMembers(friends.data.items, data.members).length === 0 && <Copy>Không có bạn mới để thêm ở trang này.</Copy>}
        {!!friends.data?.next_cursor && <Action secondary label="Xem thêm bạn để mời" disabled={busy} onPress={() => setCursor(friends.data!.next_cursor!)} />}
        {!!cursor && <Action secondary label="Về trang mời bạn đầu" disabled={busy} onPress={() => setCursor('')} />}
        <Action secondary label="Tải lại bạn để mời" disabled={busy} onPress={friends.reload} /><Action secondary label="Kết nối bạn bè" disabled={busy} onPress={() => router.push('/friends')} />
      </Card> : <Action secondary label="Rời nhóm" disabled={busy || group.loading} onPress={() => setConfirm({ action: 'leave' })} />}
      {owner && <Copy>Muốn rời nhóm? Chuyển quyền chủ nhóm cho một thành viên trước nhé.</Copy>}
      {confirm && confirmationAllowed && <Card><Heading small>{confirm.action === 'leave' ? 'Rời hội bạn này?' : confirm.action === 'transfer' ? 'Chuyển quyền quản lý?' : 'Xóa thành viên?'}</Heading>
        <Copy>{confirm.action === 'leave' ? 'Bạn vẫn giữ lời mời vào những kèo cũ. Bạn sẽ không có mặt trong các kèo nhóm mới.' : confirm.action === 'transfer' ? confirm.member.display_name + ' sẽ quản lý thành viên. Bạn vẫn ở trong nhóm.' : confirm.member.display_name + ' sẽ rời nhóm. Quyền truy cập kèo cũ không tự bị thu hồi.'}</Copy>
        <Action label={confirm.action === 'leave' ? 'Xác nhận rời nhóm' : confirm.action === 'transfer' ? 'Xác nhận chuyển quyền' : 'Xác nhận xóa thành viên'} disabled={busy || group.loading} onPress={() => { void mutate(confirm.action, confirm.action === 'leave' ? actor : confirm.member.id); }} />
        <Action secondary label="Giữ nguyên nhóm" disabled={busy} onPress={() => setConfirm(null)} />
      </Card>}
      <Action secondary label="Tải lại nhóm" disabled={busy} onPress={group.reload} />
    </>}
  </Screen>;
}
export default function GroupRoute() {
  const { id } = useLocalSearchParams<{ id: string }>();
  if (typeof id !== 'string') return <Screen><Copy error>Chưa mở được nhóm này.</Copy></Screen>;
  return <GroupScreen key={id} id={id} />;
}
