import { useState } from 'react';
import type { FormEvent } from 'react';
import type { Session, SupabaseClient } from '@supabase/supabase-js';

export function AuthPanel({ auth, onComplete, onBack }: { auth: SupabaseClient; onComplete: (session: Session) => Promise<void>; onBack: () => void }) {
  const [mode, setMode] = useState<'login' | 'signup'>('login');
  const [email, setEmail] = useState(''); const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [info, setInfo] = useState('');
  async function submit(event: FormEvent) {
    event.preventDefault(); if (busy) return; setBusy(true); setError(''); setInfo('');
    try {
      const { data, error } = mode === 'login'
        ? await auth.auth.signInWithPassword({ email: email.trim(), password })
        : await auth.auth.signUp({ email: email.trim(), password, options: { emailRedirectTo: window.location.origin } });
      if (error) {
        setError(error.code === 'invalid_credentials' ? 'Email hoặc mật khẩu chưa đúng.' : error.code === 'email_not_confirmed' ? 'Xác minh email trước khi đăng nhập nhé.' : 'Chưa đăng nhập được. Kiểm tra thông tin rồi thử lại.');
      } else if (data.session) { await onComplete(data.session); }
      else { setInfo('Kiểm tra email để xác minh tài khoản, rồi quay lại link mời này để đăng nhập.'); }
    } catch { setError('Chưa kết nối được dịch vụ đăng nhập. Thử lại sau nhé.'); }
    finally { setBusy(false); }
  }
  return <section className="auth-panel" aria-labelledby="auth-title">
    <button className="text-button" onClick={onBack} disabled={busy}>← Quay lại lời mời</button>
    <h3 id="auth-title">{mode === 'login' ? 'Gặp nhau thôi.' : 'Chào người bạn mới.'}</h3>
    <p className="muted">Đăng nhập để lưu phản hồi của bạn với cả nhóm.</p>
    <form onSubmit={submit} aria-busy={busy}>
      <label htmlFor="email">Email</label>
      <input id="email" name="email" type="email" autoComplete="email" autoFocus required maxLength={254} value={email} onChange={e => setEmail(e.target.value)} disabled={busy} />
      <label htmlFor="password">Mật khẩu</label>
      <input id="password" name="password" type="password" autoComplete={mode === 'login' ? 'current-password' : 'new-password'} required minLength={mode === 'signup' ? 8 : 1} value={password} onChange={e => setPassword(e.target.value)} disabled={busy} />
      {mode === 'signup' && <p className="field-note">Ít nhất 8 ký tự. Bạn sẽ xác minh tài khoản qua email.</p>}
      {error && <p className="error" role="alert">{error}</p>}
      {info && <p className="notice" role="status">{info}</p>}
      <button className="primary-button" type="submit" disabled={busy}>{busy ? 'Đang kết nối…' : mode === 'login' ? 'Đăng nhập & tham gia' : 'Tạo tài khoản'}</button>
    </form>
    <button className="text-button switch-auth" disabled={busy} onClick={() => { setMode(mode === 'login' ? 'signup' : 'login'); setError(''); setInfo(''); }}>{mode === 'login' ? 'Chưa có tài khoản? Đăng ký miễn phí' : 'Đã có tài khoản? Đăng nhập'}</button>
  </section>;
}
