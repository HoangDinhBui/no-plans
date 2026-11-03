const tokenPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\.[A-Za-z0-9_-]{43}$/i;
type Captured = { token: string | null; invalid: boolean };
type Target = ({ type: 'invite' } & Captured) | { type: 'route'; path: string } | { type: 'invalid' };
export function parseIncoming(raw: string, webBase = '', development = false): Target {
  try {
    if (raw.length > 4096 || raw.startsWith('//')) return { type: 'invalid' };
    const relative = raw.startsWith('/'); const url = new URL(raw, relative ? 'no-plans://app' : undefined);
    let path: string;
    if (url.protocol === 'no-plans:') {
      if (url.username || url.password || url.port) return { type: 'invalid' };
      path = url.hostname && url.hostname !== 'app' ? '/' + url.hostname + url.pathname : url.pathname;
    } else if (development && ['exp:', 'exps:'].includes(url.protocol) && !url.username && !url.password) {
      if (url.pathname === '/' || url.pathname === '') path = '/';
      else if (url.pathname.startsWith('/--/')) path = url.pathname.slice(3);
      else return { type: 'invalid' };
    } else {
      const base = new URL(webBase);
      if ((url.protocol !== 'https:' && !(development && url.protocol === 'http:')) || url.username || url.password || url.origin !== base.origin || url.pathname.replace(/\/$/, '') !== base.pathname.replace(/\/$/, '')) return { type: 'invalid' };
      path = '/invite';
    }
    path = path.replace(/\/$/, '') || '/';
    if (path === '/invite') {
      const fragment = new URLSearchParams(url.hash.slice(1)); const values = fragment.getAll('token');
      if (url.searchParams.has('token') || (url.hash && values.length !== 1) || values.length > 1 || (values.length === 1 && !tokenPattern.test(values[0]))) return { type: 'invite', token: null, invalid: true };
      return { type: 'invite', token: values[0] ?? null, invalid: false };
    }
    if (url.hash || url.searchParams.has('token')) return { type: 'invalid' };
    if (path === '/' || /^\/(plans|groups)\/[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(path)) return { type: 'route', path: path.toLowerCase() };
    return { type: 'invalid' };
  } catch { return { type: 'invalid' }; }
}
type Storage = { getItem(key: string): Promise<string | null>; setItem(key: string, value: string): Promise<void>; removeItem(key: string): Promise<void> };
export type IncomingState = Captured & { ready: boolean; error?: string };
export class IncomingInvite {
  private storage: Storage; private state: IncomingState = { ready: false, token: null, invalid: false };
  private listeners = new Set<() => void>(); private generation = 0; private writes = Promise.resolve();
  private key = 'no-plans.incoming-invite';
  constructor(storage: Storage) { this.storage = storage; }
  getSnapshot = () => this.state;
  subscribe = (listener: () => void) => { this.listeners.add(listener); return () => { this.listeners.delete(listener); }; };
  private publish(state: IncomingState) { this.state = state; for (const listener of this.listeners) listener(); }
  private queue(operation: () => Promise<void>) { const next = this.writes.then(operation); this.writes = next.catch(() => {}); return next; }
  async load() {
    if (this.state.ready) return; const generation = this.generation;
    try {
      const raw = await this.storage.getItem(this.key); const record = raw ? JSON.parse(raw) as Captured : { token: null, invalid: false };
      if ((record.token !== null && (typeof record.token !== 'string' || !tokenPattern.test(record.token))) || typeof record.invalid !== 'boolean') throw new Error();
      if (generation === this.generation) this.publish({ ...record, ready: true });
    } catch { if (generation === this.generation) this.publish({ ready: true, token: null, invalid: true, error: 'Chưa khôi phục được lời mời. Mở lại link từ người tạo kèo nhé.' }); }
  }
  async receive(input: Captured) {
    const generation = ++this.generation;
    const record: Captured = input.token && tokenPattern.test(input.token) && !input.invalid ? { token: input.token, invalid: false } : { token: null, invalid: input.invalid || !!input.token };
    this.publish({ ...record, ready: true });
    try { await this.queue(() => this.storage.setItem(this.key, JSON.stringify(record))); }
    catch { if (generation === this.generation) this.publish({ ...record, ready: true, error: 'Lời mời chưa lưu được trên thiết bị. Bạn vẫn có thể tiếp tục ở màn hình này.' }); }
  }
  async consume(expected: string): Promise<boolean> {
    if (this.state.token !== expected) return false;
    const generation = ++this.generation;
    await this.queue(() => this.storage.removeItem(this.key));
    if (generation !== this.generation || this.state.token !== expected) return false;
    this.publish({ ready: true, token: null, invalid: false }); return true;
  }
}
