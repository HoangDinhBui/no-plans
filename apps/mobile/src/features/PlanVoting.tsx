import { useEffect, useRef, useState } from 'react';
import { randomUUID } from 'expo-crypto';
import { CreationIntent } from '../lib/api';
import type { Page, Place, PlaceOption, Plan } from '../lib/api';
import { accessFailure, useAccessFailure } from '../lib/use-access-failure';
import { useSession } from '../lib/session';
import { useResource } from '../lib/use-resource';
import { Action, Card, Copy, Field, Heading, Loading } from '../ui/components';
import { PlaceExplorer } from './PlaceExplorer';
export function PlanVoting({ plan, actor, now, disabled, onChanged, onDenied }: { plan: Plan; actor: string; now: number; disabled: boolean; onChanged: () => void; onDenied: (e: Error) => void }) {
  const { api } = useSession(); const [cursor, setCursor] = useState('');
  const options = useResource<Page<PlaceOption>>(`/plans/${plan.id}/options${cursor ? '?cursor=' + encodeURIComponent(cursor) : ''}`, true, plan.version);
  useAccessFailure(options.error, onDenied);
  const [name, setName] = useState(''); const [address, setAddress] = useState(''); const [explore, setExplore] = useState(false);
  const [selection, setSelection] = useState<{ option: PlaceOption; version: number } | null>(null);
  const [busy, setBusy] = useState(false); const [error, setError] = useState(''); const [notice, setNotice] = useState('');
  const inFlight = useRef(false); const mounted = useRef(false); const intent = useRef(new CreationIntent(randomUUID));
  useEffect(() => { mounted.current = true; return () => { mounted.current = false; }; }, []);
  const owner = plan.owner_id === actor; const open = plan.status === 'inviting' && Date.parse(plan.starts_at) > now;
  async function act(action: 'add' | 'vote' | 'select', body: object, version?: number) {
    if (inFlight.current || disabled || !open) return; inFlight.current = true; setBusy(true); setError(''); setNotice('');
    try {
      const path = action === 'add' ? 'options' : action === 'vote' ? 'vote' : 'selection';
      await api.request(`/plans/${plan.id}/${path}`, { method: action === 'add' ? 'POST' : 'PUT', body, version, ...(action === 'add' ? { key: intent.current.key(actor, { plan_id: plan.id, ...body }) } : {}) });
      if (!mounted.current) return;
      if (action === 'add') { intent.current.clear(); setName(''); setAddress(''); setExplore(false); setCursor(''); }
      if (action === 'select') setSelection(null);
      setNotice(action === 'add' ? 'Đã thêm lựa chọn để cả nhóm bình chọn.' : action === 'vote' ? 'Đã lưu phiếu của bạn.' : 'Đã chọn địa điểm. Bạn có thể chốt kèo bên dưới.'); options.reload(); onChanged();
    } catch (e) { if (mounted.current) { if (accessFailure(e)) onDenied(e); else { setError(e instanceof Error ? e.message : 'Chưa thực hiện được thao tác.'); options.reload(); onChanged(); } } }
    finally { inFlight.current = false; if (mounted.current) setBusy(false); }
  }
  function custom() {
    const trimmed = name.trim(); const location = address.trim();
    if (!trimmed || [...trimmed].length > 120 || [...(trimmed + (location ? ' · ' + location : ''))].length > 500) { setError('Nhập tên nơi gặp (tối đa120 ký tự) và địa chỉ phù hợp, tổng tối đa500 ký tự.'); return; }
    void act('add', { name: trimmed, address: location });
  }
  const choose = (place: Place) => { void act('add', { place_id: place.id }); };
  return <Card><Heading small>Hội mình gặp ở đâu?</Heading><Copy>Mỗi người một phiếu, có thể đổi khi kèo còn đang mời. Kết quả là gợi ý; chủ kèo chọn nơi gặp và chốt.</Copy>
    {options.loading && <Loading />}{options.error && <Copy error>{options.error.message}</Copy>}{!!error && <Copy error>{error}</Copy>}{!!notice && <Copy>{notice}</Copy>}
    {!open && <Copy>Bình chọn đã khóa vì kèo đã chốt, đã bắt đầu hoặc không còn hoạt động.</Copy>}
    {options.data?.items.map(option => <Card key={option.id}><Copy>{option.kind === 'idea' ? 'Ý tưởng · Cần chọn nơi gặp cụ thể sau' : 'Địa điểm'}{plan.selected_option_id === option.id ? ' · Đã chọn làm nơi gặp' : ''}</Copy><Heading small>{option.name}</Heading>{!!option.address && <Copy>{option.address}</Copy>}<Copy>{option.vote_count} phiếu{option.my_vote ? ' · Bạn đã chọn' : ''}</Copy>
      {open && <Action secondary label={'Bình chọn ' + option.name} selected={option.my_vote} disabled={busy || disabled || options.loading} onPress={() => { void act('vote', { option_id: option.id }); }} />}
      {open && owner && option.kind === 'place' && <Action secondary label={'Chọn làm nơi gặp: ' + option.name} selected={plan.selected_option_id === option.id} disabled={busy || disabled || options.loading} onPress={() => setSelection({ option, version: plan.version })} />}
    </Card>)}
    {options.data?.items.length === 0 && <Copy>Chưa có lựa chọn. Chủ kèo có thể đề xuất địa điểm hoặc ý tưởng để cả nhóm cùng chọn.</Copy>}
    {selection && open && owner && <Card><Heading small>Chọn {selection.option.name}?</Heading><Copy>{selection.option.address || selection.option.name}</Copy><Copy>Đây là địa điểm gặp, chưa phải thao tác chốt kèo.</Copy>{selection.version !== plan.version && <Copy>Kèo vừa thay đổi. Kiểm tra lại và bấm Chọn làm nơi gặp một lần nữa trước khi xác nhận.</Copy>}<Action label="Xác nhận địa điểm gặp" disabled={busy || disabled || selection.version !== plan.version} onPress={() => { void act('select', { option_id: selection.option.id }, selection.version); }} /><Action secondary label="Chọn sau" disabled={busy} onPress={() => setSelection(null)} /></Card>}
    {!!options.data?.next_cursor && <Action secondary label="Trang lựa chọn tiếp" disabled={busy} onPress={() => { setSelection(null); setCursor(options.data!.next_cursor!); }} />}
    {!!cursor && <Action secondary label="Về trang lựa chọn đầu" disabled={busy} onPress={() => { setSelection(null); setCursor(''); }} />}
    {owner && open && <Card><Heading small>Thêm nơi hội mình biết</Heading><Field label="Tên địa điểm đề xuất" value={name} onChangeText={setName} maxLength={120} editable={!busy && !disabled} /><Field label="Địa chỉ địa điểm đề xuất" value={address} onChangeText={setAddress} maxLength={500} editable={!busy && !disabled} /><Action label="Thêm địa điểm để bình chọn" disabled={busy || disabled || !name.trim()} onPress={custom} /><Copy>Tối đa20 lựa chọn. Ghi rõ nơi gặp; tên món ăn hoặc ý tưởng không thay thế địa điểm cụ thể.</Copy><Action secondary label={explore ? 'Đóng khám phá' : 'Tìm trong gợi ý NO PLANS'} disabled={busy || disabled} onPress={() => setExplore(!explore)} />{explore && <PlaceExplorer disabled={busy || disabled} onChoose={choose} />}</Card>}
    <Action secondary label="Tải lại bình chọn" disabled={busy} onPress={options.reload} />
  </Card>;
}
