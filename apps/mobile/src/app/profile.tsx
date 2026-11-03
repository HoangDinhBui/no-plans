import { Platform } from 'react-native';
import { disableNativePush, enableNativePush } from '../lib/native-push';
import { useState } from 'react';
import { auth, useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import type { Profile } from '../lib/api';
import { Action, Card, Copy, Field, Heading, Loading, Screen } from '../ui/components';
export default function ProfileScreen() {
  const { session, api } = useSession(); const profile = useResource<Profile>('/me');
  const [editedName, setName] = useState<string | null>(null); const [busy, setBusy] = useState(false); const [message, setMessage] = useState('');
  const [confirmDelete, setConfirmDelete] = useState(false);
  const name = editedName ?? profile.data?.display_name ?? '';
  async function save() { if (busy) return; setBusy(true); setMessage(''); try { await api.request('/me', { method: 'PATCH', body: { display_name: name.trim() } }); setMessage('Đã lưu hồ sơ.'); profile.reload(); } catch (e) { setMessage(e instanceof Error ? e.message : 'Chưa lưu được.'); } finally { setBusy(false); } }
  async function enablePush() {
    if (busy) return;
    setBusy(true);
    try {
      const enabled = await enableNativePush(api);
      setMessage(enabled ? 'Đã bật thông báo trên thiết bị này.' : 'Thông báo chưa khả dụng hoặc chưa được cấp quyền.');
    } catch {
      setMessage('Chưa bật được thông báo. Hãy thử lại.');
    } finally {
      setBusy(false);
    }
  }
  async function logout() { if (!auth || busy) return; setBusy(true); try { await disableNativePush(api); const { error } = await auth.auth.signOut({ scope: 'local' }); if (error) setMessage('Chưa đăng xuất được. Hãy thử lại.'); } catch { setMessage('Chưa đăng xuất được. Hãy thử lại.'); } finally { setBusy(false); } }
  async function deleteProfile() {
    if (!auth || busy || !confirmDelete) return;
    setBusy(true);
    setMessage('');
    try {
      await api.request('/me', { method: 'DELETE' });
      setConfirmDelete(false);
      const { error } = await auth.auth.signOut({ scope: 'local' });
      if (error) setMessage('Đã xóa hồ sơ. Hãy đăng xuất khỏi thiết bị này.');
    } catch {
      setMessage('Chưa xác nhận được việc xóa hồ sơ. Hãy thử lại.');
    } finally {
      setBusy(false);
    }
  }
  return <Screen><Heading>Đây là bạn.</Heading>{profile.loading && <Loading />}{profile.error && <Copy error>{profile.error.message}</Copy>}<Card><Field label="Tên hiển thị" value={name} maxLength={80} onChangeText={setName} /><Action label="Lưu tên" onPress={() => { void save(); }} disabled={busy || !name.trim()} />{!!message && <Copy>{message}</Copy>}<Copy>{session?.user.email}</Copy><Copy selectable>Mã của bạn: {session?.user.id}</Copy><Copy>Chia sẻ mã để bạn bè gửi lời mời kết bạn.</Copy></Card>{Platform.OS !== 'web' && <Action secondary label="Bật thông báo" onPress={() => { void enablePush(); }} disabled={busy} />}<Action secondary label="Đăng xuất thiết bị này" onPress={() => { void logout(); }} disabled={busy} />{confirmDelete ? <Card>
      <Heading small>Xóa hồ sơ NO PLANS?</Heading>
      <Copy>Tên và ảnh đại diện sẽ bị xóa. Thiết bị sẽ ngừng nhận thông báo và hồ sơ này sẽ mất quyền truy cập NO PLANS.</Copy>
      <Action label="Xác nhận xóa hồ sơ" onPress={() => { void deleteProfile(); }} disabled={busy} />
      <Action secondary label="Giữ hồ sơ" onPress={() => setConfirmDelete(false)} disabled={busy} />
    </Card> : <Action secondary label="Xóa hồ sơ NO PLANS" onPress={() => setConfirmDelete(true)} disabled={busy} />}</Screen>;
}
