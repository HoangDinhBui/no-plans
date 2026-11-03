import { Platform } from 'react-native';
import { parseIncoming } from './incoming';
import { incomingInvite } from './pending-invite';
if (Platform.OS === 'web' && typeof window !== 'undefined') {
  const capture = () => {
    if (window.location.pathname.replace(/\/$/, '') !== '/invite') return;
    if (!window.location.hash && !new URLSearchParams(window.location.search).has('token')) return;
    const target = parseIncoming('/invite' + window.location.search + window.location.hash);
    // Remove bearer material before Expo Router initializes or observes a warm link.
    window.history.replaceState(window.history.state, '', '/invite');
    void incomingInvite.receive(target.type === 'invite' ? target : { token: null, invalid: true });
  };
  capture(); window.addEventListener('popstate', capture); window.addEventListener('hashchange', capture);
}
void incomingInvite.load();
