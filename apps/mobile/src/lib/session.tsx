import 'react-native-url-polyfill/auto';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { createClient } from '@supabase/supabase-js';
import type { Session } from '@supabase/supabase-js';
import { AppState, Platform } from 'react-native';
import { createContext, useContext, useEffect, useState } from 'react';
import type { ReactNode } from 'react';
import { APIClient } from './api';

const url = process.env.EXPO_PUBLIC_SUPABASE_URL;
const key = process.env.EXPO_PUBLIC_SUPABASE_PUBLISHABLE_KEY;
export const auth = url && key ? createClient(url, key, { auth: { storage: AsyncStorage, persistSession: true, autoRefreshToken: true, detectSessionInUrl: false } }) : null;
const apiURL = process.env.EXPO_PUBLIC_API_URL;
type SessionState = { session: Session | null; ready: boolean; configured: boolean; error: string; api: APIClient };
const Context = createContext<SessionState | null>(null);
export function SessionProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(null);
  const [ready, setReady] = useState(!auth); const [error, setError] = useState('');
  const [transport] = useState(() => {
    let current: Session | null = null;
    const api = new APIClient(apiURL ?? '', () => current?.access_token ?? null);
    return { api, accept(next: Session | null) { if (current?.access_token !== next?.access_token) api.invalidate(); current = next; } };
  });
  const { api } = transport;
  useEffect(() => {
    if (!auth) return;
    let alive = true; let revision = 0;
    const accept = (next: Session | null) => {
      if (!alive) return;
      transport.accept(next); setSession(next); setReady(true);
    };
    const { data } = auth.auth.onAuthStateChange((_event, next) => { revision++; accept(next); });
    const before = revision;
    void auth.auth.getSession().then(({ data, error }) => { if (!alive || revision !== before) return; if (error) setError('Chưa khôi phục được phiên. Đăng nhập lại nhé.'); accept(data.session); }).catch(() => { if (alive) { setError('Chưa khôi phục được phiên. Đăng nhập lại nhé.'); accept(null); } });
    const refresh = (state: string) => { if (Platform.OS !== 'web') { if (state === 'active') auth!.auth.startAutoRefresh(); else auth!.auth.stopAutoRefresh(); } };
    refresh(AppState.currentState ?? 'active');
    const subscription = AppState.addEventListener('change', refresh);
    return () => { alive = false; api.invalidate(); data.subscription.unsubscribe(); subscription.remove(); if (Platform.OS !== 'web') auth!.auth.stopAutoRefresh(); };
  }, [api, transport]);
  return <Context.Provider value={{ session, ready, configured: !!auth && !!apiURL, error, api }}>{children}</Context.Provider>;
}
export function useSession() { const value = useContext(Context); if (!value) throw new Error('Missing SessionProvider'); return value; }
