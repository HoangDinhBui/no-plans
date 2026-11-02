import { useState } from 'react';
import { router } from 'expo-router';
import { View } from 'react-native';
import type { Page, PlanSummary } from '../lib/api';
import { useResource } from '../lib/use-resource';
import { Action, Card, Copy, Heading, Loading, Screen } from '../ui/components';
const labels = { upcoming: 'Sắp tới', invited: 'Lời mời', past: 'Đã qua' };
const statuses: Record<string, string> = { inviting: 'Đang rủ hội', confirmed: 'Đã chốt', cancelled: 'Đã hủy', completed: 'Đã kết thúc' };
export default function Plans() {
  const [filter, setFilter] = useState<keyof typeof labels>('upcoming'); const [cursor, setCursor] = useState('');
  const { data, error, loading, reload } = useResource<Page<PlanSummary>>(`/plans?status=${filter}${cursor ? '&cursor=' + encodeURIComponent(cursor) : ''}`);
  return <Screen><Copy>ÍT LÊN KẾ HOẠCH. THÊM CHUYỆN ĐỂ KỂ.</Copy><Heading>Kèo của bạn.</Heading><Action label="↗ Lên kèo" onPress={() => router.push('/plans/new')} /><View style={{ gap: 8 }}>{(Object.entries(labels) as [keyof typeof labels, string][]).map(([key, label]) => <Action key={key} secondary selected={filter === key} label={label} onPress={() => { setFilter(key); setCursor(''); }} />)}</View>{loading && <Loading />}{!!error && <Card><Copy error>{error.message}</Copy><Action secondary label="Thử lại" onPress={reload} /></Card>}{data?.items.map(plan => <Card key={plan.id}><Copy>{statuses[plan.status]}</Copy><Heading small>{plan.title}</Heading><Copy>{new Date(plan.starts_at).toLocaleString('vi-VN')} · {plan.location || 'Chưa chốt địa điểm'}</Copy><Copy>{plan.going_count} đi · {plan.pending_count} chưa phản hồi</Copy><Action secondary label={'Xem kèo: ' + plan.title} onPress={() => router.push({ pathname: '/plans/[id]', params: { id: plan.id } })} /></Card>)}{data && data.items.length === 0 && <Card><Heading small>Chưa có kèo nào.</Heading><Copy>Rủ hội bạn làm gì đó nhé?</Copy></Card>}{!!data?.next_cursor && <Action secondary label="Trang tiếp" onPress={() => setCursor(data.next_cursor!)} />}{!!cursor && <Action secondary label="Về trang đầu" onPress={() => setCursor('')} />}<View style={{ gap: 12 }}><Action secondary label="Nhóm của bạn" onPress={() => router.push('/groups')} /><Action secondary label="Bạn bè" onPress={() => router.push('/friends')} /><Action secondary label="Hồ sơ" onPress={() => router.push('/profile')} /><Action secondary label="Tải lại kèo" onPress={reload} /></View></Screen>;
}
