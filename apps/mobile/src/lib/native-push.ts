import AsyncStorage from '@react-native-async-storage/async-storage';
import Constants from 'expo-constants';
import { randomUUID } from 'expo-crypto';
import { Platform } from 'react-native';
import type { APIClient } from './api';
import { registerPushDevice, revokePushDevice } from './push-devices';

const installationKey = 'no-plans.push.installation';
let installation: Promise<string> | undefined;

function getInstallationId() {
  installation ??= AsyncStorage.getItem(installationKey).then(async (existing) => {
    if (existing) return existing;
    const id = randomUUID();
    await AsyncStorage.setItem(installationKey, id);
    return id;
  }).catch((error: unknown) => {
    installation = undefined;
    throw error;
  });
  return installation;
}

export async function enableNativePush(api: APIClient) {
  const platform = Platform.OS;
  if (platform !== 'android' && platform !== 'ios') return false;
  const projectId = Constants.easConfig?.projectId;
  if (!projectId) return false;

  const scope = api.captureScope();
  const notifications = await import('expo-notifications');
  if (platform === 'android') {
    await notifications.setNotificationChannelAsync('default', {
      name: 'Thông báo NO PLANS',
      importance: notifications.AndroidImportance.DEFAULT,
    });
  }
  let permission = await notifications.getPermissionsAsync();
  if (!permission.granted && permission.canAskAgain) permission = await notifications.requestPermissionsAsync();
  if (!permission.granted) return false;

  const { data: token } = await notifications.getExpoPushTokenAsync({ projectId });
  await registerPushDevice(api, await getInstallationId(), platform, token, scope);
  return true;
}

export async function disableNativePush(api: APIClient) {
  if (Platform.OS !== 'android' && Platform.OS !== 'ios') return;
  const scope = api.captureScope();
  const id = await AsyncStorage.getItem(installationKey);
  if (id) await revokePushDevice(api, id, scope);
}
