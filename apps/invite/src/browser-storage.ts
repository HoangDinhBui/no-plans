import type { KeyStorage } from './invite-state';
export const browserStorage: KeyStorage = {
  getItem(key) { try { return window.sessionStorage.getItem(key); } catch { return null; } },
  setItem(key, value) { try { window.sessionStorage.setItem(key, value); } catch { /* In-memory flow remains usable. */ } },
  removeItem(key) { try { window.sessionStorage.removeItem(key); } catch { /* Browser may disable storage. */ } },
};
