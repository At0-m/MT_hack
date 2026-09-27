import type { Schema, Selection, Overrides, Calculation, Bootstrap, Geometry, Descriptor } from './types';
export const MOCK = import.meta.env.VITE_API_MODE === 'mock';
export const DEV_TOOLS = import.meta.env.DEV && new URLSearchParams(location.search).get('dev') === '1';
if (MOCK && !DEV_TOOLS) document.cookie='mt_mock_fault=; Path=/; Max-Age=0; SameSite=Strict';
export class ApiError extends Error {
  constructor(public problem: Schema['Problem']) { super(problem.detail || problem.title); }
}
export async function request<T>(path: string, body?: unknown, signal?: AbortSignal, blob = false): Promise<T> {
  const response = await fetch('/api/v1'+path, { method: body === undefined ? 'GET' : 'POST', credentials: 'include', headers: body === undefined ? undefined : {'Content-Type':'application/json'}, body: body === undefined ? undefined : JSON.stringify(body), signal });
  if (!response.ok) {
    let problem: Schema['Problem'];
    try { problem = await response.json(); } catch { problem = { type:'about:blank', title:'Ошибка сервера', detail:`Запрос завершился с кодом ${response.status}.`,status:response.status,code:'HTTP_ERROR',request_id:response.headers.get('X-Request-ID') ?? 'unknown' }; }
    if (response.status === 401 && path !== '/auth/login' && path !== '/auth/session') window.dispatchEvent(new Event('session-expired'));
    throw new ApiError(problem);
  }
  if (response.status === 204) return undefined as T;
  return (blob ? response.blob() : response.json()) as Promise<T>;
}
export const api = {
  session: (signal?: AbortSignal) => request<Schema['AuthSession']>('/auth/session',undefined,signal),
  login: (body: Schema['LoginRequest']) => request<Schema['AuthSession']>('/auth/login',body),
  logout: () => request<void>('/auth/logout',{}),
  bootstrap: (signal?: AbortSignal) => request<Bootstrap>('/bootstrap',undefined,signal),
  route: (id: string, version: string, signal?: AbortSignal) => request<Schema['RouteDetail']>(`/routes/${encodeURIComponent(id)}?network_version=${encodeURIComponent(version)}`,undefined,signal),
  geometry: (id: string, version: string, signal?: AbortSignal) => request<Geometry>(`/routes/${encodeURIComponent(id)}/geometry?network_version=${encodeURIComponent(version)}`,undefined,signal),
  calculate: (selection: Selection, overrides?: Overrides, signal?: AbortSignal) => request<Calculation>(overrides?'/scenarios/evaluate':'/forecasts/query', overrides ? {selection,overrides}:{selection},signal),
  summary: (calculation: Descriptor, expected_calculation_id: string, focus: Schema['SummaryFocus'], signal?: AbortSignal) => request<Schema['SummaryResponse']>('/summaries/query',{calculation,expected_calculation_id,focus},signal),
  export: (calculation: Descriptor, expected_calculation_id: string, signal?: AbortSignal) => request<Blob>('/exports',{calculation,expected_calculation_id,format:'csv'},signal,true),
};
export function errorText(error: unknown) {
  if (error instanceof ApiError) {
    const known: Record<string,string> = {CALCULATION_MISMATCH:'Версия расчёта изменилась. Повторите расчёт вручную.',SNAPSHOT_EXPIRED:'Snapshot больше недоступен. Обновите данные вручную.',INVALID_CREDENTIALS:'Неверный логин или пароль.',SESSION_EXPIRED:'Сессия истекла. Войдите снова.'};
    return `${known[error.problem.code] ?? error.message} (${error.problem.code}; request_id: ${error.problem.request_id})`;
  }
  return error instanceof Error ? error.message : 'Не удалось выполнить запрос.';
}
