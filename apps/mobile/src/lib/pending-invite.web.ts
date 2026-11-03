import { IncomingInvite } from './incoming';
export const incomingInvite = new IncomingInvite({
  async getItem(key) { return window.sessionStorage.getItem(key); },
  async setItem(key, value) { window.sessionStorage.setItem(key, value); },
  async removeItem(key) { window.sessionStorage.removeItem(key); },
});
