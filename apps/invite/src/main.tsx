import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { createClient } from '@supabase/supabase-js';
import { App } from './App';
import { APIClient } from './api';
import { captureInvite } from './invite-state';
import { browserStorage } from './browser-storage';
import './styles.css';

const invite = captureInvite(browserStorage, window.location, url => window.history.replaceState(null, '', url));
const auth = import.meta.env.VITE_SUPABASE_URL && import.meta.env.VITE_SUPABASE_ANON_KEY
  ? createClient(import.meta.env.VITE_SUPABASE_URL, import.meta.env.VITE_SUPABASE_ANON_KEY, { auth: { storage: browserStorage, flowType: 'pkce', persistSession: true, autoRefreshToken: true, detectSessionInUrl: true } }) : null;
const api = new APIClient(import.meta.env.VITE_API_URL || '');
const root = createRoot(document.getElementById('root')!);
let invitationVersion = 0;
const render = (next: typeof invite) => root.render(<StrictMode><App key={invitationVersion} api={api} auth={auth} invite={next} /></StrictMode>);
render(invite);
window.addEventListener('hashchange', () => {
  if (!new URLSearchParams(window.location.hash.slice(1)).has('token')) return;
  const next = captureInvite(browserStorage, window.location, url => window.history.replaceState(null, '', url));
  invitationVersion++;
  render(next);
});
