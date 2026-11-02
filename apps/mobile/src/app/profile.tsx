import { useState } from 'react';
import { auth, useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import type { Profile } from '../lib/api';
import { Action, Card, Copy, Field, Heading, Loading, Screen } from '../ui/components';
export default function ProfileScreen() {
  const { session, api } = useSession(); const profile = useResource<Profile>('/me');
  const [editedName, setName] = useState<string | null>(null); const [busy, setBusy] = useState(false); const [message, setMessage] = useState('');
  const name = editedName ?? profile.data?.display_name ?? '';
  async function save() { if (busy) return; setBusy(true); setMessage(''); try { await api.request('/me', { method: 'PATCH', body: { display_name: name.trim() } }); setMessage('Đã lưu hồ sơ.'); profile.reload(); } catch (e) { setMessage(e instanceof Error ? e.message : 'Chưa lưu được.'); } finally { setBusy(false); } }
  async function logout() { if (!auth || busy) return; setBusy(true); try { const { error } = await auth.auth.signOut({ scope: 'local' }); if (error) setMessage('Chưa đăng xuất được. Hãy thử lại.'); } catch { setMessage('Chưa đăng xuất được. Hãy thử lại.'); } finally { setBusy(false); } }
  return <Screen><Heading>Đây là bạn.</Heading>{profile.loading && <Loading />}{profile.error && <Copy error>{profile.error.message}</Copy>}<Card><Field label="Tên hiển thị" value={name} maxLength={80} onChangeText={setName} /><Action label="Lưu tên" onPress={() => { void save(); }} disabled={busy || !name.trim()} />{!!message && <Copy>{message}</Copy>}<Copy>{session?.user.email}</Copy><Copy selectable>Mã của bạn: {session?.user.id}</Copy><Copy>Chia sẻ mã để bạn bè gửi lời mời kết bạn.</Copy></Card><Action secondary label="Đăng xuất thiết bị này" onPress={() => { void logout(); }} disabled={busy} /></Screen>;
}
