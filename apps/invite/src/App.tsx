import { useEffect, useRef, useState } from 'react';
import type { Session, SupabaseClient } from '@supabase/supabase-js';
import { APIClient, APIError } from './api';
import type { Plan, Preview, RSVP } from './api';
import { activePlanKey, forget, inviteStorageKey, save, saved, validID } from './invite-state';
import { browserStorage } from './browser-storage';
import { AuthPanel } from './AuthPanel';

const activityLabels: Record<string, string> = { food: 'Ăn uống', coffee: 'Cà phê', drinks: 'Đi uống', cinema: 'Xem phim', sports: 'Thể thao', other: 'Đi chơi' };
const dateLabel = (at: string) => new Intl.DateTimeFormat('vi-VN', { weekday: 'long', day: 'numeric', month: 'numeric', timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date(at));
const timeLabel = (at: string) => new Intl.DateTimeFormat('vi-VN', { hour: '2-digit', minute: '2-digit', hour12: false, timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date(at));

export function App({ api, auth, invite }: { api: APIClient; auth: SupabaseClient | null; invite: { token: string | null; invalid: boolean } }) {
  const [token, setToken] = useState(invite.token);
  const [activeId, setActiveId] = useState<string | null>(() => { const id = saved(browserStorage, activePlanKey); return !invite.token && !invite.invalid && validID(id) ? id : null; });
  const [session, setSession] = useState<Session | null>(null);
  const privateGeneration = useRef(0);
  const currentAccessToken = useRef<string | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null); const [plan, setPlan] = useState<Plan | null>(null);
  const [busy, setBusy] = useState<string | null>(invite.token ? 'preview' : null); const [authOpen, setAuthOpen] = useState(false);
  const [error, setError] = useState<Error | null>(null); const [notice, setNotice] = useState(''); const [retry, setRetry] = useState(0);
  const [dark, setDark] = useState(() => saved(browserStorage, 'no-plans.theme') === 'dark' || (saved(browserStorage, 'no-plans.theme') === null && window.matchMedia?.('(prefers-color-scheme: dark)').matches));

  useEffect(() => { document.documentElement.dataset.theme = dark ? 'dark' : 'light'; save(browserStorage, 'no-plans.theme', dark ? 'dark' : 'light'); }, [dark]);
  useEffect(() => {
    if (!auth) return; let alive = true;
    void auth.auth.getSession().then(({ data }) => { if (alive) acceptSession(data.session); }).catch(() => { if (alive) setError(new Error('Chưa khôi phục được phiên đăng nhập. Hãy thử đăng nhập lại.')); });
    const { data } = auth.auth.onAuthStateChange((event, next) => { if (alive) { acceptSession(next); if (event === 'SIGNED_OUT') { privateGeneration.current++; setPlan(null); setPreview(null); } } });
    return () => { alive = false; data.subscription.unsubscribe(); };
  }, [auth]);
  useEffect(() => {
    if (!token) return; const controller = new AbortController(); setBusy('preview'); setError(null);
    void api.preview(token, controller.signal).then(setPreview).catch(e => { if (!controller.signal.aborted) setError(e); }).finally(() => { if (!controller.signal.aborted) setBusy(null); });
    return () => controller.abort();
  }, [api, token, retry]);
  useEffect(() => {
    if (!activeId || !session) return; const controller = new AbortController(); const generation = privateGeneration.current; setBusy('plan');
    void api.plan(activeId, session.access_token, controller.signal).then(p => { if (!controller.signal.aborted && generation === privateGeneration.current) { setPlan(p); setError(null); } }).catch(e => { if (!controller.signal.aborted && generation === privateGeneration.current) { handlePrivateError(e); } }).finally(() => { if (!controller.signal.aborted && generation === privateGeneration.current) setBusy(null); });
    return () => controller.abort();
  }, [api, activeId, session, retry]);
  useEffect(() => {
    if (!plan || !session) return;
    const controller = new AbortController(); const generation = privateGeneration.current; let disposed = false; let inFlight = false; let timer: ReturnType<typeof setTimeout>; let delay = 15000;
    const refresh = async () => {
      if (disposed || inFlight) return; clearTimeout(timer);
      if (document.visibilityState === 'visible') {
        inFlight = true;
        try { const fresh = await api.plan(plan.id, session.access_token, controller.signal); if (!disposed && generation === privateGeneration.current) setPlan(fresh); delay = 15000; }
        catch (e) { if (!disposed && generation === privateGeneration.current && e instanceof APIError && [401, 403].includes(e.status)) handlePrivateError(e); delay = Math.min(delay * 2, 60000); }
        finally { inFlight = false; }
      }
      if (!disposed) timer = setTimeout(refresh, delay);
    };
    timer = setTimeout(refresh, delay); const onFocus = () => { void refresh(); };
    window.addEventListener('focus', onFocus); window.addEventListener('online', onFocus); document.addEventListener('visibilitychange', onFocus);
    return () => { disposed = true; clearTimeout(timer); controller.abort(); window.removeEventListener('focus', onFocus); window.removeEventListener('online', onFocus); document.removeEventListener('visibilitychange', onFocus); };
  }, [api, plan?.id, session?.access_token]);

  function acceptSession(next: Session | null) {
    if (currentAccessToken.current !== (next?.access_token ?? null)) {
      privateGeneration.current++;
      currentAccessToken.current = next?.access_token ?? null;
      setPlan(null); setBusy(null); setNotice('');
    }
    setSession(next);
  }
  function handlePrivateError(e: unknown) {
    if (e instanceof APIError && [401, 403].includes(e.status)) {
      privateGeneration.current++;
      setPlan(null); setPreview(null); setBusy(null); setNotice('');
      if (e.status === 403) { setActiveId(null); forget(browserStorage, activePlanKey); }
    }
    setError(e instanceof Error ? e : new APIError('network_error'));
  }
  async function join(next = session) {
    if (!token) return;
    if (!auth) { setError(new Error('Đăng nhập chưa được cấu hình cho bản chạy này.')); return; }
    if (!next) { setAuthOpen(true); return; }
    setBusy('join'); setError(null);
    const generation = privateGeneration.current;
    try { const id = await api.redeem(token, next.access_token); if (generation !== privateGeneration.current) return; save(browserStorage, activePlanKey, id); forget(browserStorage, inviteStorageKey); setToken(null); setActiveId(id); setAuthOpen(false); }
    catch (e) { if (generation === privateGeneration.current) { setError(e instanceof Error ? e : new APIError('network_error')); setBusy(null); } }
  }
  async function respond(response: RSVP) {
    if (!plan || !session || busy) return; setBusy('rsvp'); setError(null); setNotice('');
    const generation = privateGeneration.current;
    try { await api.rsvp(plan.id, response, session.access_token); if (generation !== privateGeneration.current) return; const fresh = await api.plan(plan.id, session.access_token); if (generation === privateGeneration.current) { setPlan(fresh); setNotice('Đã lưu phản hồi của bạn.'); } }
    catch (e) { if (generation === privateGeneration.current) handlePrivateError(e); }
    finally { if (generation === privateGeneration.current) setBusy(null); }
  }
  async function logout() {
    if (!auth || busy) return; setBusy('logout');
    const { error } = await auth.auth.signOut({ scope: 'local' }); setBusy(null);
    if (error) { setError(new Error('Đăng xuất chưa hoàn tất. Hãy thử lại.')); return; }
    acceptSession(null); setPlan(null); setActiveId(null); forget(browserStorage, activePlanKey); setNotice('Đã đăng xuất.');
  }

  const unavailable = invite.invalid || (error instanceof APIError && error.code === 'invite_unavailable');
  const content = plan ?? preview;
  const canRespond = !!plan && ['inviting', 'confirmed'].includes(plan.status) && Date.parse(plan.starts_at) > Date.now();
  const myResponse = session && plan ? plan.members[session.user.id] : undefined;
  const going = plan ? Object.values(plan.members).filter(r => r === 'going').length : 0;

  return <div className="site-shell">
    <a className="skip-link" href="#invitation">Đến lời mời</a>
    <header className="site-header">
      <div className="wordmark" aria-label="NO PLANS"><span className="brand-symbol" aria-hidden="true">↗</span>NO PLANS<span className="brand-dot">.</span></div>
      <div className="header-actions">{session && <button className="text-button" onClick={() => { void logout(); }} disabled={!!busy}>Đăng xuất</button>}<button className="theme-button" onClick={() => setDark(!dark)} aria-label={dark ? 'Bật giao diện sáng' : 'Bật giao diện tối'}><span aria-hidden="true">{dark ? '☀' : '◐'}</span></button></div>
    </header>
    <main className="invite-layout">
      <section className="intro" aria-labelledby="intro-title">
        <p className="eyebrow"><span className="pulse-dot" aria-hidden="true" /> ÍT LÊN KẾ HOẠCH. THÊM CHUYỆN ĐỂ KỂ.</p>
        <h1 id="intro-title">Hội mình<br />đang chờ <span className="highlight">bạn.<svg viewBox="0 0 220 18" aria-hidden="true"><path d="M3 12Q100-3 215 9M15 16Q109 6 203 13" /></svg></span></h1>
        <p className="intro-copy">Một lời rủ rê, một cái gật đầu.<br />Còn lại, để những cuộc gặp làm nên câu chuyện.</p>
        <div className="illustration" aria-hidden="true"><div className="orbit orbit-one" /><div className="orbit orbit-two" /><span className="spark">✳</span><div className="mini-ticket"><span>HÔM NAY</span><strong>Có kèo<br />là đi.</strong><i>↗</i><div className="ticket-dots">● ● ● ● ● ● ● ●</div></div><span className="floating-note">Hẹn nhau ngoài đời nhé!</span></div>
        <ol className="steps"><li><span>01</span> Mở lời mời</li><li><span>02</span> Gật đầu cái đã</li><li><span>03</span> Gặp nhau thôi</li></ol>
      </section>
      <section id="invitation" className="invite-card" aria-labelledby="card-title" aria-busy={!!busy}>
        <div className="card-top"><span className="eyebrow">MỘT CUỘC HẸN NGOÀI ĐỜI</span><span className="ticket-star" aria-hidden="true">✳</span></div>
        {unavailable ? <div className="empty-state"><span className="empty-symbol" aria-hidden="true">↗</span><h2 id="card-title">Link mời không còn khả dụng</h2><p className="muted">Link có thể hết lượt, hết hạn hoặc được thu hồi. Nhờ người tạo gửi bạn một link mới nhé.</p>{token && auth && (authOpen ? <AuthPanel auth={auth} onBack={() => setAuthOpen(false)} onComplete={async next => { acceptSession(next); setAuthOpen(false); await join(next); }} /> : <button className="primary-button" disabled={!!busy} onClick={() => { void join(); }}>Đã tham gia? Mở lại kèo</button>)}</div>
          : busy === 'preview' || (busy === 'plan' && !plan) || (busy === 'join' && !content) ? <div className="empty-state" role="status"><span className="loading-ring" aria-hidden="true" /><h2 id="card-title">Đang mở lời mời…</h2><p className="muted">Một chút thôi, hội bạn sắp ở đây rồi.</p></div>
          : content ? <>
            <div className="activity-badge">{activityLabels[content.activity]}<span aria-hidden="true"> · </span>{plan ? plan.status === 'confirmed' ? 'Đã chốt kèo' : plan.status === 'cancelled' ? 'Đã hủy' : plan.status === 'completed' ? 'Đã kết thúc' : 'Đang rủ hội' : 'Bạn được mời'}</div>
            <h2 id="card-title" className="plan-title">{content.title}</h2>
            <div className="schedule"><div className="calendar-icon" aria-hidden="true"><span>{new Intl.DateTimeFormat('vi-VN', { month: 'numeric', timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date(content.starts_at))}</span><strong>{new Intl.DateTimeFormat('vi-VN', { day: 'numeric', timeZone: 'Asia/Ho_Chi_Minh' }).format(new Date(content.starts_at))}</strong></div><div><p className="schedule-time">{timeLabel(content.starts_at)} <span>· giờ Việt Nam</span></p><p className="muted schedule-date">{dateLabel(content.starts_at)}</p></div></div>
            {plan ? <><div className="location-block"><span className="detail-label">ĐỊA ĐIỂM</span><p>{plan.location || 'Cả nhóm đang chọn địa điểm'}</p>{plan.location && <a className="map-link" href={'https://www.google.com/maps/search/?api=1&query=' + encodeURIComponent(plan.location)} target="_blank" rel="noreferrer noopener">Mở chỉ đường ↗</a>}</div><p className="group-summary"><span className="small-dot" aria-hidden="true" />{going} người xác nhận đi · {Object.keys(plan.members).length} người trong kèo</p></>
              : <div className="privacy-note"><span aria-hidden="true">◎</span><p>Địa điểm và danh sách bạn bè sẽ hiện sau khi bạn tham gia.</p></div>}
            {authOpen && auth ? <AuthPanel auth={auth} onBack={() => setAuthOpen(false)} onComplete={async next => { acceptSession(next); setAuthOpen(false); if (token) await join(next); }} />
              : plan ? <div className="response-section"><h3>Bạn có đi không?</h3>{canRespond ? <div className="rsvp-buttons">{([['going', 'Đi', '✓'], ['maybe', 'Chưa chắc', '?'], ['not_going', 'Không đi', '−']] as const).map(([value, label, icon]) => <button key={value} aria-pressed={myResponse === value} className={'rsvp-button ' + (myResponse === value ? 'selected' : '')} disabled={!!busy} onClick={() => { void respond(value); }}><span aria-hidden="true">{icon}</span>{label}</button>)}</div> : <p className="notice">Kèo đã bắt đầu, kết thúc hoặc được hủy nên không thể đổi phản hồi.</p>}<p className="response-help muted">Bạn có thể đổi ý trước giờ hẹn. Cả nhóm sẽ thấy phản hồi mới nhất.</p></div>
                : <><button className="primary-button join-button" disabled={!!busy} onClick={() => { void join(); }}>{busy === 'join' ? 'Đang tham gia…' : <>Mình tham gia <span aria-hidden="true">↗</span></>}</button><p className="card-caption">Miễn phí tham gia. Không cần cài ứng dụng.</p></>}
          </>
            : activeId ? <div className="empty-state"><h2 id="card-title">Kèo của bạn vẫn ở đây.</h2><p className="muted">Đăng nhập để xem lại kèo đã tham gia.</p>{authOpen && auth ? <AuthPanel auth={auth} onBack={() => setAuthOpen(false)} onComplete={async next => { acceptSession(next); setAuthOpen(false); }} /> : <button className="primary-button" onClick={() => auth ? setAuthOpen(true) : setError(new Error('Đăng nhập chưa được cấu hình cho bản chạy này.'))}>Đăng nhập</button>}</div>
              : <div className="empty-state"><span className="empty-symbol" aria-hidden="true">↗</span><h2 id="card-title">{error ? 'Chưa mở được lời mời' : 'Có link mời rồi chứ?'}</h2><p className="muted">Mở link từ người tạo kèo để xem cuộc hẹn và trả lời với cả nhóm.{session ? ' Bạn đã đăng nhập sẵn sàng rồi.' : ''}</p></div>}
        {error && !unavailable && <div className="error-box"><p role="alert">{error.message}</p>{!(error instanceof APIError) || error.status !== 403 ? <button className="text-button" onClick={() => { setRetry(value => value + 1); setError(null); }} disabled={!!busy}>Thử lại</button> : null}{error instanceof APIError && error.status === 401 && auth && <button className="text-button" onClick={() => setAuthOpen(true)}>Đăng nhập lại</button>}</div>}
        {notice && <p className="notice" role="status">{notice}</p>}
        <div className="card-bottom"><span aria-hidden="true">◎</span> Lời mời riêng tư. Cuộc vui của hội mình.</div>
      </section>
    </main>
    <footer className="site-footer"><p>Stop scrolling. <strong>Start going.</strong></p><p>Ít thời gian bàn bạc. Nhiều thời gian bên nhau.</p></footer>
  </div>;
}
