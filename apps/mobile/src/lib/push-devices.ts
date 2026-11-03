import type { APIClient } from './api';

type PushPlatform = 'ios' | 'android';

export function registerPushDevice(api: APIClient, deviceId: string, platform: PushPlatform, token: string, scope: number) {
  return api.request<{ status: 'ok' }>(`/devices/${encodeURIComponent(deviceId)}`, {
    method: 'PUT', body: { platform, push_token: token }, scope,
  });
}

export function revokePushDevice(api: APIClient, deviceId: string, scope: number) {
  return api.request<{ status: 'ok' }>(`/devices/${encodeURIComponent(deviceId)}`, {
    method: 'DELETE', scope,
  });
}
