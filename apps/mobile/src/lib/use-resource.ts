import { useEffect, useRef, useState } from 'react';
import { AppState } from 'react-native';
import { APIError } from './api';
import { useSession } from './session';
export function useResource<T>(path: string, poll = false) {
  const { api, session } = useSession(); const [retry, setRetry] = useState(0);
  const signature = path + ':' + (session?.access_token ?? '');
  const [state, setState] = useState<{ signature: string; data?: T; error?: Error; loading: boolean }>({ signature: '', loading: true });
  const generation = useRef(0);
  useEffect(() => {
    let alive = true; let inFlight = false; let delay = 15000; let timer: ReturnType<typeof setTimeout> | undefined;
    const controller = new AbortController();
    const load = async () => {
      if (!alive || inFlight) return; inFlight = true;
      const requestGeneration = ++generation.current;
      if (timer) clearTimeout(timer);
      setState(old => ({ signature, data: old.signature === signature ? old.data : undefined, loading: true }));
      try { const data = await api.request<T>(path, { signal: controller.signal }); if (alive && requestGeneration === generation.current) setState({ signature, data, loading: false }); delay = 15000; }
      catch (e) {
        if (alive && requestGeneration === generation.current && !(e instanceof APIError && e.code === 'cancelled')) {
          const denied = e instanceof APIError && [401, 403, 404, 410].includes(e.status);
          setState(old => ({ signature, data: !denied && old.signature === signature ? old.data : undefined, error: e instanceof Error ? e : new Error('Chưa tải được dữ liệu.'), loading: false }));
        }
        delay = Math.min(60000, delay * 2);
      }
      finally { inFlight = false; if (alive && poll) timer = setTimeout(() => { if (AppState.currentState === 'active') void load(); }, delay); }
    };
    void load();
    const foreground = AppState.addEventListener('change', state => { if (state === 'active') void load(); });
    return () => { alive = false; controller.abort(); if (timer) clearTimeout(timer); foreground.remove(); };
  }, [api, path, signature, retry, poll]);
  const current = state.signature === signature ? state : { loading: true, data: undefined, error: undefined };
  return { ...current, reload: () => { generation.current++; setRetry(n => n + 1); }, reject: (error: Error) => { generation.current++; setState({ signature, error, loading: false }); } };
}
