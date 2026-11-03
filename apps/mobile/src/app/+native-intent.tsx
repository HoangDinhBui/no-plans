import { parseIncoming } from '../lib/incoming';
import { incomingInvite } from '../lib/pending-invite';
let arrival = 0;
export async function redirectSystemPath({ path }: { path: string; initial: boolean }) {
  try {
    const target = parseIncoming(path, process.env.EXPO_PUBLIC_INVITE_WEB_URL ?? '', __DEV__);
    if (target.type === 'invite') {
      if (target.token || target.invalid) await incomingInvite.receive(target); else await incomingInvite.load();
      return '/invite?arrival=' + Date.now() + '-' + (++arrival);
    }
    return target.type === 'route' ? target.path : '/link-error';
  } catch { return '/link-error'; }
}
