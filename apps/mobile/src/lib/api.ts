const messages: Record<string, string> = {
  unauthorized: 'Phiên đăng nhập đã hết hạn. Đăng nhập lại nhé.', forbidden: 'Bạn không có quyền thực hiện thao tác này.',
  email_unverified: 'Xác minh email trước khi gửi lời mời hoặc tạo kèo nhé.',
  plan_full: 'Kèo đã hết chỗ. Bạn vẫn có thể chọn Chưa chắc hoặc Không đi.',
  plan_closed: 'Kèo không còn nhận thay đổi lúc này.', version_conflict: 'Kèo vừa thay đổi. Tải lại và kiểm tra trước khi lưu.',
  venue_required: 'Ý tưởng chưa phải nơi gặp. Hãy thêm hoặc nhập một địa điểm cụ thể.',
  invalid_input: 'Kiểm tra lại thông tin đã nhập.', rate_limited: 'Bạn thao tác hơi nhanh. Đợi một chút rồi thử lại nhé.',
  invite_unavailable: 'Link mời hết hạn, hết lượt hoặc đã được thu hồi.',
  network_error: 'Chưa kết nối được. Kiểm tra mạng rồi thử lại.', timeout: 'Kết nối mất nhiều thời gian. Hãy thử lại.',
  cancelled: 'Đã dừng yêu cầu.', invalid_response: 'Chưa đọc được dữ liệu. Hãy tải lại.',
};
export class APIError extends Error {
  code: string; status: number;
  constructor(code: string, status = 0) { super(messages[code] ?? 'Chưa thực hiện được thao tác. Hãy thử lại.'); this.code = code; this.status = status; }
}
type Options = { method?: string; body?: unknown; key?: string; version?: number; signal?: AbortSignal; inviteToken?: string; public?: boolean; scope?: number };
export class APIClient {
  private generation = 0;
  private base: string; private accessToken: () => string | null; private fetcher: typeof fetch;
  constructor(base: string, accessToken: () => string | null, fetcher: typeof fetch = fetch) {
    this.base = base.replace(/\/$/, ''); this.accessToken = accessToken; this.fetcher = fetcher;
  }
  invalidate() { this.generation++; }
  captureScope() { return this.generation; }
  async request<T>(path: string, options: Options = {}): Promise<T> {
    if (options.scope !== undefined && options.scope !== this.generation) throw new APIError('cancelled');
    const generation = this.generation;
    const headers = new Headers({ Accept: 'application/json' });
    if (!options.public) { const token = this.accessToken(); if (!token) throw new APIError('unauthorized', 401); headers.set('Authorization', `Bearer ${token}`); }
    if (options.key) headers.set('Idempotency-Key', options.key);
    if (options.version !== undefined) headers.set('If-Match', `"${options.version}"`);
    if (options.inviteToken) headers.set('X-Invite-Token', options.inviteToken);
    if (options.body !== undefined) headers.set('Content-Type', 'application/json');
    const controller = new AbortController(); let timedOut = false;
    const abort = () => controller.abort();
    options.signal?.addEventListener('abort', abort, { once: true }); if (options.signal?.aborted) abort();
    const timer = setTimeout(() => { timedOut = true; controller.abort(); }, 15000);
    try {
      const response = await this.fetcher.call(globalThis, this.base + '/v1' + path, { method: options.method ?? 'GET', headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), signal: controller.signal, credentials: 'omit', referrerPolicy: 'no-referrer' });
      let data: unknown; try { data = await response.json(); } catch { throw new APIError('invalid_response', response.status); }
      if (generation !== this.generation || options.signal?.aborted) throw new APIError('cancelled');
      if (!response.ok) { const code = (data as { code?: unknown } | null)?.code; throw new APIError(typeof code === 'string' ? code : 'unknown', response.status); }
      return data as T;
    } catch (e) {
      if (e instanceof APIError) throw e;
      throw new APIError(timedOut ? 'timeout' : options.signal?.aborted ? 'cancelled' : 'network_error');
    } finally { clearTimeout(timer); options.signal?.removeEventListener('abort', abort); }
  }
}

// A transport failure leaves this intent intact; editing input or switching actor rotates it.
export class CreationIntent {
  private signature = ''; private value = ''; private create: () => string;
  constructor(create: () => string) { this.create = create; }
  key(actor: string, payload: unknown): string {
    const signature = JSON.stringify([actor, payload]);
    if (signature !== this.signature || !this.value) { this.signature = signature; this.value = this.create(); }
    return this.value;
  }
  clear() { this.signature = ''; this.value = ''; }
}

export type Page<T> = { items: T[]; next_cursor?: string };
export type Profile = { id: string; display_name: string; avatar_url?: string };
export type Group = { id: string; name: string; owner_id: string };
export type GroupDetail = Group & { members: Profile[] };
export type Friendship = { id: string; low_id: string; high_id: string; requester_id: string; status: 'pending' | 'accepted'; peer: Profile };
export type Participant = Profile & { response: RSVP };
export type InviteLink = { id: string; plan_id: string; expires_at: string; max_uses: number; use_count: number; revoked_at?: string };
export type IssuedInvite = InviteLink & { token: string };
export type Place = { id: string; name: string; activity: string; kind: 'place' | 'idea'; area: string; address: string; description: string; budget_min: number | null; budget_max: number | null; price_note: string; people_min: number; people_max: number; source_url: string; source_checked_at?: string; active: boolean };
export type PlaceOption = { id: string; plan_id: string; place_id?: string; kind: 'place' | 'idea'; name: string; address: string; vote_count: number; my_vote: boolean };
export type RSVP = 'going' | 'not_going' | 'maybe' | 'invited';
export type Plan = { id: string; owner_id: string; title: string; activity: string; starts_at: string; location: string; capacity: number; status: 'inviting' | 'confirmed' | 'cancelled' | 'completed'; version: number; members: Record<string, RSVP>; selected_option_id?: string };
export type PlanSummary = Plan & { my_response: RSVP; going_count: number; pending_count: number };
