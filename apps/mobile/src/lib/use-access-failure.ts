import { useEffect } from 'react';
import { APIError } from './api';
export function accessFailure(error: unknown): error is APIError {
  return error instanceof APIError && (error.status === 401 || error.status === 404 || error.status === 410 || (error.status === 403 && error.code === 'forbidden'));
}
export function useAccessFailure(error: Error | undefined, reject: (error: Error) => void) {
  useEffect(() => { if (accessFailure(error)) reject(error); }, [error, reject]);
}
