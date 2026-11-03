import { useEffect } from 'react';
import { Platform } from 'react-native';
import { useRouter } from 'expo-router';

const uuid = /^[0-9a-f]{8}(?:-[0-9a-f]{4}){3}-[0-9a-f]{12}$/i;

export function usePushNavigation(userId: string | undefined) {
  const router = useRouter();

  useEffect(() => {
    if (!userId || (Platform.OS !== 'ios' && Platform.OS !== 'android')) return;
    let active = true;
    let unsubscribe: (() => void) | undefined;
    void import('expo-notifications').then((notifications) => {
      if (!active) return;
      const open = (response: ReturnType<typeof notifications.getLastNotificationResponse>) => {
        if (!active || response?.actionIdentifier !== notifications.DEFAULT_ACTION_IDENTIFIER) return;
        const data = response.notification.request.content.data;
        notifications.clearLastNotificationResponse();
        if (data?.recipientId === userId && typeof data.planId === 'string' && uuid.test(data.planId)) {
          router.push({ pathname: '/plans/[id]', params: { id: data.planId } });
        }
      };
      const subscription = notifications.addNotificationResponseReceivedListener(open);
      unsubscribe = () => subscription.remove();
      open(notifications.getLastNotificationResponse());
    }).catch(() => {
      // Push is optional in builds without the native module.
    });
    return () => { active = false; unsubscribe?.(); };
  }, [router, userId]);
}
