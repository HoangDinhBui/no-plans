import { router } from 'expo-router';
import { Action, Copy, Heading, Screen } from '../ui/components';
export default function LinkError() { return <Screen><Heading>Chưa mở được link này.</Heading><Copy>Link có thể không hợp lệ hoặc không còn được hỗ trợ. Nhờ người gửi kiểm tra lại nhé.</Copy><Action label="Về kèo của bạn" onPress={() => router.replace('/')} /></Screen>; }
