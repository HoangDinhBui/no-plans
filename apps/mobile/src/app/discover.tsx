import { router } from 'expo-router';
import { PlaceExplorer } from '../features/PlaceExplorer';
import { Action, Copy, Heading, Screen } from '../ui/components';
export default function Discover() { return <Screen><Heading>Ra ngoài một chút?</Heading><Copy>Địa điểm và ý tưởng cho cuộc gặp tiếp theo. Ngân sách là dự trù; xem ghi chú của từng mục.</Copy><Action label="Lên kèo với hội bạn" onPress={() => router.push('/plans/new')} /><PlaceExplorer /></Screen>; }
