import { useState } from 'react';
import { View } from 'react-native';
import { auth, useSession } from '../lib/session';
import { Action, Card, Copy, Field, Heading, Loading, Screen } from '../ui/components';
export function AuthScreen() {
  const { ready, configured, error: sessionError } = useSession();
  const [email, setEmail] = useState(''); const [password, setPassword] = useState(''); const [signup, setSignup] = useState(false);
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [notice, setNotice] = useState('');
  async function submit() {
    if (!auth || busy) return;
    if (!email.trim().includes('@') || !password || (signup && password.length < 8)) { setError('Nhập email và mật khẩu hợp lệ. Mật khẩu mới cần ít nhất8 ký tự.'); return; }
    setBusy(true); setError(''); setNotice('');
    try {
      const result = signup ? await auth.auth.signUp({ email: email.trim(), password }) : await auth.auth.signInWithPassword({ email: email.trim(), password });
      if (result.error) setError(result.error.code === 'email_not_confirmed' ? 'Xác minh email trước khi đăng nhập nhé.' : 'Chưa đăng nhập được. Kiểm tra thông tin rồi thử lại.');
      else if (!result.data.session) setNotice('Kiểm tra email để xác minh tài khoản, rồi quay lại đăng nhập.');
    } catch { setError('Chưa kết nối được dịch vụ đăng nhập. Hãy thử lại.'); }
    finally { setBusy(false); }
  }
  return <Screen><Copy>NO PLANS ↗</Copy><Heading>Bớt bàn.{'\n'}Đi thôi.</Heading><Copy>Rủ hội bạn, chốt một cái hẹn và gặp nhau ngoài đời.</Copy>{!ready ? <Loading /> : !configured ? <Card><Heading small>Chưa kết nối dịch vụ</Heading><Copy>Bản chạy này chưa được cấu hình. Hãy liên hệ người cung cấp ứng dụng.</Copy></Card> : <Card><Heading small>{signup ? 'Chào người bạn mới.' : 'Hội mình đang chờ.'}</Heading><Field label="Email" value={email} onChangeText={setEmail} keyboardType="email-address" autoCapitalize="none" autoCorrect={false} autoComplete="email" editable={!busy} /><Field label="Mật khẩu" value={password} onChangeText={setPassword} secureTextEntry autoComplete={signup ? 'new-password' : 'current-password'} editable={!busy} />{signup && <Copy>Ít nhất8 ký tự. Xác minh email để gửi lời mời.</Copy>}{(error || sessionError) && <Copy error>{error || sessionError}</Copy>}{notice && <Copy>{notice}</Copy>}<View style={{ gap: 12 }}><Action label={busy ? 'Đang kết nối…' : signup ? 'Tạo tài khoản' : 'Đăng nhập'} onPress={() => { void submit(); }} disabled={busy} /><Action secondary label={signup ? 'Đã có tài khoản? Đăng nhập' : 'Chưa có tài khoản? Đăng ký'} onPress={() => { setSignup(!signup); setError(''); setNotice(''); }} disabled={busy} /></View></Card>}</Screen>;
}
