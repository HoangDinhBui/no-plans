import type { PlanDraft } from './plan-input.ts';
type Storage = { getItem(key: string): Promise<string | null>; setItem(key: string, value: string): Promise<void>; removeItem(key: string): Promise<void> };
type Record = { version: 1; draft: PlanDraft; pending?: { signature: string; key: string; preparedAt: number; draftSignature?: string } };
export const emptyDraft = (): PlanDraft => ({ title: '', activity: 'coffee', date: '', time: '19:30', capacity: '', location: '', group_id: '' });
export class DraftStore {
  private storage: Storage; private uuid: () => string; private now: () => number;
  private queues = new Map<string, Promise<void>>();
  constructor(storage: Storage, uuid: () => string, now: () => number = Date.now) { this.storage = storage; this.uuid = uuid; this.now = now; }
  private key(actor: string) { return 'no-plans.plan-draft.' + encodeURIComponent(actor); }
  private async read(actor: string): Promise<Record> {
    const raw = await this.storage.getItem(this.key(actor));
    if (raw === null) return { version: 1, draft: emptyDraft() };
    try {
      if (raw.length > 16000) throw new Error();
      const record = JSON.parse(raw) as Record;
      if (record.version !== 1 || !record.draft || Object.keys(emptyDraft()).some(k => typeof record.draft[k as keyof PlanDraft] !== 'string')) throw new Error();
      if (record.pending && (typeof record.pending.signature !== 'string' || typeof record.pending.key !== 'string' || !/^[\x21-\x7e]{1,128}$/.test(record.pending.key) || !Number.isFinite(record.pending.preparedAt))) throw new Error();
      return record;
    } catch { throw new Error('Chưa đọc được bản nháp. Kiểm tra kèo đã tạo trước khi xóa bản nháp trên thiết bị.'); }
  }
  private serial<T>(actor: string, operation: () => Promise<T>): Promise<T> {
    const previous = this.queues.get(actor) ?? Promise.resolve();
    const run = previous.then(operation); const tail = run.then(() => {}, () => {});
    this.queues.set(actor, tail);
    void tail.then(() => { if (this.queues.get(actor) === tail) this.queues.delete(actor); });
    return run;
  }
  load(actor: string) { return this.serial(actor, () => this.read(actor)); }
  saveDraft(actor: string, draft: PlanDraft) {
    const copy = { ...draft };
    return this.serial(actor, async () => { const record = await this.read(actor); record.draft = copy; await this.storage.setItem(this.key(actor), JSON.stringify(record)); });
  }
  prepare(actor: string, draft: PlanDraft, payload: unknown): Promise<string> {
    const copy = { ...draft }; const signature = JSON.stringify(payload);
    return this.serial(actor, async () => {
      const record = await this.read(actor); const now = this.now();
      // Server replay records live24h. Leave a safety margin and require reconciliation after this window.
      if (record.pending && (now - record.pending.preparedAt >= 23 * 3600000 || now < record.pending.preparedAt)) throw new Error('Lần tạo trước đã quá lâu. Hãy kiểm tra danh sách kèo trước khi xóa bản nháp và tạo lại.');
      if (!record.pending || record.pending.signature !== signature) record.pending = { signature, key: this.uuid(), preparedAt: now };
      record.pending.draftSignature = JSON.stringify(copy);
      record.draft = copy;
      await this.storage.setItem(this.key(actor), JSON.stringify(record));
      return record.pending.key;
    });
  }
  complete(actor: string, key: string) {
    return this.serial(actor, async () => {
      const record = await this.read(actor);
      if (record.pending?.key !== key) return;
      if (record.pending.draftSignature === JSON.stringify(record.draft)) await this.storage.removeItem(this.key(actor));
      else { delete record.pending; await this.storage.setItem(this.key(actor), JSON.stringify(record)); }
    });
  }
  discard(actor: string) { return this.serial(actor, () => this.storage.removeItem(this.key(actor))); }
}
