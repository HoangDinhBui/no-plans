import { useEffect, useRef, useState } from 'react';
import { Share } from 'react-native';
import { randomUUID } from 'expo-crypto';
import { APIError, CreationIntent } from '../lib/api';
import type { Friendship, InviteLink, IssuedInvite, Page, Plan } from '../lib/api';
import { inviteLimit, inviteURL, linkStatus, validateInviteBase } from '../lib/invites';
import { useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import { accessFailure, useAccessFailure } from '../lib/use-access-failure';
import { Action, Card, Copy, Field, Heading, Loading } from '../ui/components';
const labels = { active: 'Còn hiệu lực', full: 'Đã hết lượt', expired: 'Đã hết hạn', revoked: 'Đã thu hồi' };
const webBase = process.env.EXPO_PUBLIC_INVITE_WEB_URL ?? '';
export function PlanInviteOwner({ plan, actor, now, disabled, onChanged, onDenied }: { plan: Plan; actor: string; now: number; disabled: boolean; onChanged: () => void; onDenied: (e: Error) => void }) {
  const { api } = useSession(); const [friendCursor, setFriendCursor] = useState(''); const [linkCursor, setLinkCursor] = useState('');
  const friends = useResource<Page<Friendship>>('/friend-requests' + (friendCursor ? '?cursor=' + encodeURIComponent(friendCursor) : ''));
  const links = useResource<Page<InviteLink>>(`/plans/${plan.id}/invite-links${linkCursor ? '?cursor=' + encodeURIComponent(linkCursor) : ''}`, true, plan.version);
  useAccessFailure(links.error, onDenied);
  const [maximum, setMaximum] = useState('100'); const [issued, setIssued] = useState<(IssuedInvite & { url: string }) | null>(null);
  const [revokeID, setRevokeID] = useState(''); const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [notice, setNotice] = useState('');
  const inFlight = useRef(false); const mounted = useRef(false); const intent = useRef(new CreationIntent(randomUUID));
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const future = ['inviting', 'confirmed'].includes(plan.status) && Date.parse(plan.starts_at) > now;
  const issuedMetadata = issued && (links.data?.items.find(link => link.id === issued.id) ?? issued);
  const visibleURL = issued && issuedMetadata && future && linkStatus(issuedMetadata, now) === 'active' ? issued.url : '';
  async function run(action: 'create' | 'revoke' | 'invite', target = '') {
    if (inFlight.current || disabled) return; inFlight.current = true; setBusy(true); setError(''); setNotice('');
    try {
      if (action === 'create') {
        validateInviteBase(webBase, __DEV__); const body = { max_uses: inviteLimit(maximum) };
        const result = await api.request<IssuedInvite>(`/plans/${plan.id}/invite-links`, { method: 'POST', body, key: intent.current.key(actor, { plan_id: plan.id, ...body }) });
        if (!mounted.current) return;
        if (!result.token?.startsWith(result.id + '.')) throw new APIError('invalid_response');
        const url = inviteURL(webBase, result.token, __DEV__); setIssued({ ...result, url }); intent.current.clear(); setLinkCursor(''); links.reload();
      } else if (action === 'revoke') {
        await api.request('/invite-links/' + target, { method: 'DELETE' });
        if (!mounted.current) return;
        if (issued?.id === target) setIssued(null); setRevokeID(''); setNotice('Đã thu hồi link. Những người đã vào kèo vẫn giữ quyền; thu hồi từng thành viên nếu cần.'); links.reload();
      } else {
        await api.request(`/plans/${plan.id}/invitations`, { method: 'POST', body: { user_id: target } });
        if (!mounted.current) return;
        setNotice('Đã mời bạn vào kèo.'); onChanged();
      }
    } catch (e) { if (mounted.current) { if (accessFailure(e)) onDenied(e); else setError(e instanceof Error ? e.message : 'Chưa thực hiện được thao tác.'); } }
    finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }
  async function share() { if (!visibleURL) return; try { await Share.share({ message: visibleURL }); } catch { if (mounted.current) setError('Chưa mở được chia sẻ. Bạn có thể nhấn giữ link để sao chép.'); } }
  const candidates = friends.data?.items.filter(f => f.status === 'accepted' && !(f.peer.id in plan.members)) ?? [];
  return <Card><Heading small>Rủ hội vào kèo</Heading>{!!error && <Copy error>{error}</Copy>}{!!notice && <Copy>{notice}</Copy>}
    {future && <><Heading small>Mời bạn đã kết nối</Heading>{friends.loading && <Loading />}{friends.error && <Copy error>{friends.error.message}</Copy>}
      {candidates.map(friend => <Action key={friend.id} secondary label={'Mời ' + friend.peer.display_name + ' vào kèo'} disabled={busy || disabled} onPress={() => { void run('invite', friend.peer.id); }} />)}
      {friends.data && candidates.length === 0 && <Copy>Không có bạn mới để mời ở trang này.</Copy>}
      {!!friends.data?.next_cursor && <Action secondary label="Xem thêm bạn để mời kèo" disabled={busy} onPress={() => setFriendCursor(friends.data!.next_cursor!)} />}
      {!!friendCursor && <Action secondary label="Về trang mời kèo đầu" disabled={busy} onPress={() => setFriendCursor('')} />}
      <Action secondary label="Tải lại bạn để mời kèo" disabled={busy} onPress={friends.reload} />
      <Heading small>Gửi một link lời mời</Heading><Field label="Số lượt cho link mới (1–1000)" value={maximum} onChangeText={setMaximum} keyboardType="number-pad" editable={!busy && !disabled} />
      <Action label="Tạo link lời mời" disabled={busy || disabled} onPress={() => { void run('create'); }} /><Copy>Link hết hạn sau7 ngày. Người nhận cần đăng nhập để vào kèo. Chỉ chia sẻ với người bạn muốn rủ.</Copy>
    </>}
    {!!visibleURL && <Card><Heading small>Link vừa tạo</Heading><Copy selectable>{visibleURL}</Copy><Action label="Chia sẻ link vừa tạo" disabled={busy || disabled} onPress={() => { void share(); }} /><Copy>Link này chỉ hiện trong phiên mở màn hình. Bạn có thể thu hồi link từ danh sách bên dưới.</Copy></Card>}
    <Heading small>Link đã phát hành</Heading>{links.loading && <Loading />}{links.error && <Copy error>{links.error.message}</Copy>}
    {links.data?.items.map((link, index) => <Card key={link.id}><Copy>Link {index + 1} · {labels[linkStatus(link, now)]}</Copy><Copy>{link.use_count}/{link.max_uses} lượt · Hết hạn {new Date(link.expires_at).toLocaleString('vi-VN')}</Copy>{!link.revoked_at && <Action secondary label={'Thu hồi link ' + (index + 1)} disabled={busy || disabled} onPress={() => setRevokeID(link.id)} />}</Card>)}
    {!!revokeID && links.data?.items.some(link => link.id === revokeID && !link.revoked_at) && <Card><Copy>Thu hồi link sẽ chặn người mới vào bằng link này. Người đã tham gia vẫn ở trong kèo.</Copy><Action label="Xác nhận thu hồi link" disabled={busy || disabled} onPress={() => { void run('revoke', revokeID); }} /><Action secondary label="Giữ link" disabled={busy} onPress={() => setRevokeID('')} /></Card>}
    {!!links.data?.next_cursor && <Action secondary label="Trang link tiếp" disabled={busy} onPress={() => { setRevokeID(''); setLinkCursor(links.data!.next_cursor!); }} />}
    {!!linkCursor && <Action secondary label="Về trang link đầu" disabled={busy} onPress={() => { setRevokeID(''); setLinkCursor(''); }} />}
    <Action secondary label="Tải lại link lời mời" disabled={busy} onPress={links.reload} />
  </Card>;
}
