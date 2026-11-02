import { useState } from 'react';
import { Linking, View } from 'react-native';
import type { Page, Place } from '../lib/api';
import { budgetLabel, catalogueQuery, defaultFilters } from '../lib/discovery';
import type { Filters } from '../lib/discovery';
import { useResource } from '../lib/use-resource';
import { Action, Card, Copy, Field, Heading, Loading } from '../ui/components';
const activities = { '': 'Tất cả hoạt động', food: 'Ăn uống', coffee: 'Cà phê', drinks: 'Đi uống', cinema: 'Xem phim', sports: 'Thể thao', other: 'Đi chơi' };
export function PlaceExplorer({ onChoose, disabled = false }: { onChoose?: (place: Place) => void; disabled?: boolean }) {
  const [filters, setFilters] = useState(defaultFilters); const [applied, setApplied] = useState(defaultFilters); const [cursor, setCursor] = useState(''); const [error, setError] = useState('');
  const results = useResource<Page<Place>>(catalogueQuery(applied, cursor));
  const update = (field: keyof Filters, value: string) => setFilters(old => ({ ...old, [field]: value }));
  function apply() { try { catalogueQuery(filters); setApplied({ ...filters }); setCursor(''); setError(''); results.reload(); } catch (e) { setError(e instanceof Error ? e.message : 'Kiểm tra lại bộ lọc.'); } }
  async function source(url: string) { try { const parsed = new URL(url); if (parsed.protocol !== 'https:') throw new Error(); await Linking.openURL(url); } catch { setError('Chưa mở được nguồn thông tin. Hãy thử lại.'); } }
  return <><Card><Heading small>Tìm ý tưởng cho hội mình</Heading><Copy>Chọn bộ lọc rồi bấm Tìm gợi ý. Có ngân sách sẽ chỉ hiện mục có chi phí dự trù phù hợp.</Copy>
    <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>{Object.entries(activities).map(([key, label]) => <Action key={key} secondary label={label} selected={filters.activity === key} disabled={disabled} onPress={() => update('activity', key)} />)}</View>
    <View style={{ gap: 8 }}>{Object.entries({ '': 'Cả địa điểm và ý tưởng', place: 'Địa điểm', idea: 'Ý tưởng' }).map(([key, label]) => <Action key={key} secondary label={label} selected={filters.kind === key} disabled={disabled} onPress={() => update('kind', key)} />)}</View>
    <Field label="Khu vực" value={filters.area} onChangeText={v => update('area', v)} maxLength={80} editable={!disabled} />
    <Field label="Ngân sách tối đa (đ, có thể để trống)" keyboardType="number-pad" value={filters.budget} onChangeText={v => update('budget', v)} editable={!disabled} />
    <Field label="Số người đi" keyboardType="number-pad" value={filters.people} onChangeText={v => update('people', v)} editable={!disabled} />
    <Action label="Tìm gợi ý" disabled={disabled} onPress={apply} />{!!error && <Copy error>{error}</Copy>}
  </Card>{results.loading && <Loading />}{results.error && <Copy error>{results.error.message}</Copy>}
    {results.data?.items.map(place => <Card key={place.id}><Copy>{place.kind === 'idea' ? 'Ý tưởng hoạt động · Chưa phải địa điểm' : 'Địa điểm'} · {place.area}</Copy><Heading small>{place.name}</Heading><Copy>{place.description}</Copy>{!!place.address && <Copy>{place.address}</Copy>}<Copy>{budgetLabel(place.budget_min, place.budget_max)}</Copy>{!!place.price_note && <Copy>{place.price_note}</Copy>}<Copy>Gợi ý cho {place.people_min}{place.people_max ? '–' + place.people_max : '+'} người; không phải số chỗ còn trống.</Copy>
      {!!place.source_url && <><Action secondary label={'Xem nguồn: ' + place.name} onPress={() => { void source(place.source_url); }} />{!!place.source_checked_at && <Copy>Kiểm tra nguồn: {new Date(place.source_checked_at).toLocaleDateString('vi-VN')}</Copy>}</>}
      {onChoose && <Action label={'Đề xuất ' + place.name} disabled={disabled} onPress={() => onChoose(place)} />}
    </Card>)}
    {results.data?.items.length === 0 && <Copy>Chưa có gợi ý phù hợp. Thử nới bộ lọc hoặc tự thêm nơi bạn biết.</Copy>}
    {!!results.data?.next_cursor && <Action secondary label="Trang gợi ý tiếp" disabled={disabled} onPress={() => setCursor(results.data!.next_cursor!)} />}
    {!!cursor && <Action secondary label="Về trang gợi ý đầu" disabled={disabled} onPress={() => setCursor('')} />}
    <Action secondary label="Tải lại gợi ý" disabled={disabled} onPress={results.reload} />
  </>;
}
