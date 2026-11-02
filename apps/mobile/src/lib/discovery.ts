export type Filters = { activity: string; kind: string; area: string; budget: string; people: string };
export const defaultFilters = (): Filters => ({ activity: '', kind: '', area: '', budget: '', people: '1' });
export function catalogueQuery(filters: Filters, cursor = ''): string {
  if ((filters.activity && !['food', 'coffee', 'drinks', 'cinema', 'sports', 'other'].includes(filters.activity)) || (filters.kind && !['place', 'idea'].includes(filters.kind))) throw new Error('Chọn bộ lọc hợp lệ.');
  const people = Number(filters.people.trim()); const budget = filters.budget.trim();
  if (!/^\d+$/.test(filters.people.trim()) || people < 1 || people > 1000) throw new Error('Số người cần từ1 đến1000.');
  if (budget && (!/^\d+$/.test(budget) || Number(budget) > 2147483647)) throw new Error('Nhập ngân sách nguyên không âm, tối đa2.147.483.647đ.');
  const area = filters.area.trim(); if ([...area].length > 80) throw new Error('Tên khu vực tối đa80 ký tự.');
  const query = new URLSearchParams({ people: String(people), limit: '20' });
  if (filters.activity) query.set('activity', filters.activity);
  if (filters.kind) query.set('kind', filters.kind);
  if (area) query.set('area', area);
  if (budget) query.set('max_budget', String(Number(budget)));
  if (cursor) query.set('cursor', cursor);
  return '/places?' + query.toString();
}
export function budgetLabel(min: number | null, max: number | null) {
  if (min === null || max === null) return 'Chi phí chưa xác minh';
  const format = (n: number) => new Intl.NumberFormat('vi-VN').format(n);
  return 'Dự trù ' + (min === max ? format(min) : format(min) + '–' + format(max)) + 'đ';
}
