import { validID } from './invite-state';

export type RSVP = 'going' | 'not_going' | 'maybe';
export type Preview = { title: string; activity: string; starts_at: string };
export type Plan = Preview & { id: string; owner_id: string; status: 'inviting' | 'confirmed' | 'cancelled' | 'completed'; location: string; version: number; members: Record<string, RSVP | 'invited'> };

const messages: Record<string, string> = {
  invite_unavailable: 'Link mời này không còn khả dụng.',
  forbidden: 'Bạn không còn quyền xem kèo này.',
  unauthorized: 'Chưa xác thực được phiên đăng nhập. Bạn có thể thử lại hoặc đăng nhập lại.',
  rate_limited: 'Bạn thao tác hơi nhanh. Đợi một chút rồi thử lại nhé.',
  plan_full: 'Kèo vừa hết chỗ. Bạn vẫn có thể chọn Chưa chắc hoặc Không đi.',
  plan_closed: 'Kèo đã bắt đầu, kết thúc hoặc được hủy. Không thể đổi phản hồi lúc này.',
  auth_unavailable: 'Chưa kết nối được dịch vụ đăng nhập. Thử lại sau nhé.',
  invalid_response: 'Chưa đọc được thông tin lời mời. Hãy thử tải lại.',
  network_error: 'Chưa kết nối được. Kiểm tra mạng rồi thử lại nhé.',
  timeout: 'Kết nối mất nhiều thời gian hơn dự kiến. Hãy thử lại.',
};

export class APIError extends Error {
  constructor(public code: string, public status: number = 0) { super(messages[code] ?? 'Chưa thực hiện được thao tác. Hãy thử lại.'); this.name = 'APIError'; }
}

const activities = new Set(['food', 'coffee', 'drinks', 'cinema', 'sports', 'other']);
function previewData(value: unknown): Preview {
  const p = value as Partial<Preview> | null;
  if (!p || typeof p.title !== 'string' || typeof p.activity !== 'string' || !activities.has(p.activity) || typeof p.starts_at !== 'string' || !Number.isFinite(Date.parse(p.starts_at))) throw new APIError('invalid_response');
  return { title: p.title, activity: p.activity, starts_at: p.starts_at };
}
function planData(value: unknown): Plan {
  const p = value as Plan | null; previewData(p);
  if (!p || !validID(p.id) || !validID(p.owner_id) || !['inviting', 'confirmed', 'cancelled', 'completed'].includes(p.status) || typeof p.location !== 'string' || !Number.isSafeInteger(p.version) || !p.members || Array.isArray(p.members) || typeof p.members !== 'object' || Object.values(p.members).some(v => !['going', 'not_going', 'maybe', 'invited'].includes(v))) throw new APIError('invalid_response');
  return p;
}

type RequestOptions = { method?: string; body?: unknown; accessToken?: string; inviteToken?: string; signal?: AbortSignal };
export class APIClient {
  constructor(private baseURL = '', private fetcher: typeof fetch = fetch) { this.baseURL = baseURL.replace(/\/$/, ''); }
  private async request(path: string, options: RequestOptions = {}): Promise<unknown> {
    const headers = new Headers({ Accept: 'application/json' });
    if (options.accessToken) headers.set('Authorization', `Bearer ${options.accessToken}`);
    if (options.inviteToken) headers.set('X-Invite-Token', options.inviteToken);
    if (options.body !== undefined) headers.set('Content-Type', 'application/json');
    const controller = new AbortController(); const abort = () => controller.abort();
    options.signal?.addEventListener('abort', abort, { once: true }); if (options.signal?.aborted) abort();
    const timer = setTimeout(abort, 15000);
    try {
      const response = await this.fetcher.call(globalThis, this.baseURL + '/v1' + path, { method: options.method ?? 'GET', headers, body: options.body === undefined ? undefined : JSON.stringify(options.body), signal: controller.signal, referrerPolicy: 'no-referrer', credentials: 'omit' });
      let data: unknown; try { data = await response.json(); } catch { throw new APIError('invalid_response', response.status); }
      if (!response.ok) { const code = (data as { code?: unknown } | null)?.code; throw new APIError(typeof code === 'string' ? code : 'unknown', response.status); }
      return data;
    } catch (error) {
      if (error instanceof APIError) throw error;
      throw new APIError(controller.signal.aborted ? 'timeout' : 'network_error');
    } finally { clearTimeout(timer); options.signal?.removeEventListener('abort', abort); }
  }
  async preview(token: string, signal?: AbortSignal): Promise<Preview> { return previewData(await this.request('/invite-preview', { inviteToken: token, signal })); }
  async redeem(token: string, accessToken: string): Promise<string> { const result = await this.request('/invite-redeem', { method: 'POST', inviteToken: token, accessToken }) as { plan_id?: unknown }; if (!validID(result?.plan_id)) throw new APIError('invalid_response'); return result.plan_id; }
  async plan(id: string, accessToken: string, signal?: AbortSignal): Promise<Plan> { if (!validID(id)) throw new APIError('invalid_response'); return planData(await this.request(`/plans/${id}`, { accessToken, signal })); }
  async rsvp(id: string, response: RSVP, accessToken: string): Promise<void> { if (!validID(id)) throw new APIError('invalid_response'); await this.request(`/plans/${id}/rsvp`, { method: 'PUT', body: { response }, accessToken }); }
}
