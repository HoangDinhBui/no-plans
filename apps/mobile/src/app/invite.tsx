import { useEffect, useRef, useState, useSyncExternalStore } from 'react';
import { View } from 'react-native';
import { router } from 'expo-router';
import { APIError } from '../lib/api';
import { parseIncoming } from '../lib/incoming';
import { incomingInvite } from '../lib/pending-invite';
import { useSession } from '../lib/session';
import { AuthScreen } from '../features/AuthScreen';
import { Action, Card, Copy, Heading, Loading, Screen } from '../ui/components';
type Preview = { title: string; activity: string; starts_at: string };
const activities: Record<string, string> = { food: 'Ăn uống', coffee: 'Cà phê', drinks: 'Đi uống', cinema: 'Xem phim', sports: 'Thể thao', other: 'Đi chơi' };
export default function InviteScreen() {
  const incoming = useSyncExternalStore(incomingInvite.subscribe, incomingInvite.getSnapshot, incomingInvite.getSnapshot);
  const { session, api, ready } = useSession(); const token = incoming.token;
  const [retry, setRetry] = useState(0); const [loginOpen, setLoginOpen] = useState(false); const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [joined, setJoined] = useState<{ token: string; id: string } | null>(null);
  const [preview, setPreview] = useState<{ token: string; data?: Preview; error?: Error; loading: boolean } | null>(null);
  const inFlight = useRef(false); const mounted = useRef(false);
  useEffect(() => { mounted.current = true; void incomingInvite.load(); return () => { mounted.current = false; }; }, []);
  useEffect(() => {
    if (!token) return; const controller = new AbortController(); const currentToken = token;
    async function load() {
      setPreview({ token: currentToken, loading: true });
      try {
        const data = await api.request<Preview>('/invite-preview', { public: true, inviteToken: currentToken, signal: controller.signal });
        if (!data || typeof data.title !== 'string' || !(data.activity in activities) || !Number.isFinite(Date.parse(data.starts_at))) throw new APIError('invalid_response');
        if (!controller.signal.aborted) setPreview({ token: currentToken, data: { title: data.title, activity: data.activity, starts_at: data.starts_at }, loading: false });
      } catch (e) { if (!controller.signal.aborted) setPreview({ token: currentToken, error: e instanceof APIError ? e : new APIError('network_error'), loading: false }); }
    }
    void load(); return () => controller.abort();
  }, [api, token, retry, session?.access_token]);
  async function join() {
    if (!token || inFlight.current) return;
    if (!session) { setLoginOpen(true); return; }
    inFlight.current = true; setBusy(true); setError(''); const scope = api.captureScope(); let established = joined?.token === token;
    try {
      let id = joined?.token === token ? joined.id : '';
      if (!id) {
        const result = await api.request<{ plan_id: string }>('/invite-redeem', { method: 'POST', inviteToken: token, scope });
        if (!mounted.current || incomingInvite.getSnapshot().token !== token) return;
        const target = parseIncoming('/plans/' + result?.plan_id);
        if (target.type !== 'route' || !target.path.startsWith('/plans/')) throw new APIError('invalid_response');
        id = result.plan_id; established = true; setJoined({ token, id });
      }
      if (await incomingInvite.consume(token) && mounted.current && scope === api.captureScope()) router.replace({ pathname: '/plans/[id]', params: { id } });
    } catch (e) { if (mounted.current && incomingInvite.getSnapshot().token === token) setError(established ? 'Bạn đã vào kèo, nhưng chưa đóng được lời mời lưu trên thiết bị. Bấm tiếp tục để thử lại.' : e instanceof APIError ? e.message : 'Chưa vào được kèo. Hãy thử lại.'); }
    finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }
  const current = preview?.token === token ? preview : null;
  if (loginOpen && !session) return <View style={{ flex: 1 }}><AuthScreen /><View style={{ padding: 20 }}><Action secondary label="Quay lại lời mời" onPress={() => setLoginOpen(false)} /></View></View>;
  return <Screen><Copy>NO PLANS ↗</Copy><Heading>Hội mình đang chờ.</Heading>
    {!incoming.ready || !ready ? <Loading /> : incoming.invalid ? <Card><Heading small>Lời mời không hợp lệ</Heading><Copy>Nhờ người tạo gửi lại link mới nhé.</Copy></Card> : !token ? <Card><Heading small>Có link mời rồi chứ?</Heading><Copy>Mở link từ người tạo kèo để xem cuộc hẹn và tham gia.</Copy></Card> : <Card>
      {(!current || current.loading) && <Loading />}
      {current?.data && <><Copy>{activities[current.data.activity]}</Copy><Heading small>{current.data.title}</Heading><Copy>{new Date(current.data.starts_at).toLocaleString('vi-VN', { timeZone: 'Asia/Ho_Chi_Minh' })} · giờ Việt Nam</Copy><Copy>Địa điểm và bạn bè chỉ hiện sau khi bạn tham gia.</Copy></>}
      {current?.error && <Copy error>{current.error.message}</Copy>}
      <Action label={busy ? 'Đang mở kèo…' : joined?.token === token ? 'Tiếp tục mở kèo đã tham gia' : !session ? 'Đăng nhập để tham gia' : current?.error instanceof APIError && current.error.code === 'invite_unavailable' ? 'Đã tham gia? Mở lại kèo' : 'Tham gia kèo'} disabled={busy} onPress={() => { void join(); }} />
      <Copy>Link hết lượt vẫn có thể mở lại cho người đã tham gia. Link hết hạn hoặc bị thu hồi cần người tạo gửi mới.</Copy><Action secondary label="Tải lại lời mời" disabled={busy} onPress={() => setRetry(n => n + 1)} />
    </Card>}
    {!!incoming.error && <Copy>{incoming.error}</Copy>}{!!error && <Copy error>{error}</Copy>}
    <Action secondary label="Về kèo của bạn" disabled={busy} onPress={() => router.replace('/')} />
  </Screen>;
}
