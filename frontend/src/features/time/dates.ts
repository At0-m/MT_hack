import { DateTime } from 'luxon';
import type { Bootstrap, Selection, Window, Schema } from '../../api/types';
export const ZONE = 'Europe/Moscow';
export const moscow = (value: string) => DateTime.fromISO(value, { zone: ZONE }).setZone(ZONE);
export const iso = (date: DateTime) => date.toISO({ suppressMilliseconds: true })!;
export function makeWindow(mode: Schema['ViewMode'], start: string, end = start): Window {
  const selected = moscow(start).startOf('day');
  const from = mode === 'month' ? selected.startOf('month') : selected;
  if (!from.isValid) throw new Error('Укажите корректную дату.');
  const to = mode === 'month' ? from.plus({ months: 1 }) : mode === 'week' ? from.plus({ days: 7 }) : mode === 'day' ? from.plus({ days: 1 }) : moscow(end).startOf('day').plus({ days: 1 });
  if (!to.isValid || to <= from) throw new Error('Конечная дата не может быть раньше начальной.');
  return { from: iso(from), to: iso(to) };
}
export function frameWindows(selection: Selection): Window[] {
  const frames: Window[] = [];
  let cursor = moscow(selection.window.from);
  const end = moscow(selection.window.to);
  if (!cursor.isValid || !end.isValid || end <= cursor) throw new Error('Некорректные границы периода.');
  while (cursor < end && frames.length <= 8784) {
    const next = cursor.plus(selection.resolution === 'hour' ? { hours: 1 } : { days: 1 });
    if (next > end) throw new Error('Неполный временной кадр.');
    frames.push({ from: iso(cursor), to: iso(next) }); cursor = next;
  }
  return frames;
}
export function validateSelection(s: Selection, b: Bootstrap, stopCount?: number): void {
  const from = moscow(s.window.from), to = moscow(s.window.to);
  if (!from.isValid || !to.isValid || to <= from || from.toMillis() !== from.startOf('day').toMillis() || to.toMillis() !== to.startOf('day').toMillis()) throw new Error('Период должен начинаться и заканчиваться в полночь по Москве.');
  const days = to.diff(from, 'days').days, hours = to.diff(from, 'hours').hours;
  if ((s.view_mode === 'day' && (days !== 1 || s.resolution !== 'hour')) || (s.view_mode !== 'day' && s.resolution !== 'day') || (s.view_mode === 'week' && days !== 7) || (s.view_mode === 'month' && (from.day !== 1 || to.toMillis() !== from.plus({months:1}).toMillis()))) throw new Error('Период не соответствует выбранному режиму.');
  if (s.view_mode === 'custom' && days > b.limits.max_custom_days) throw new Error(`Допустимо не более ${b.limits.max_custom_days} дней.`);
  const count = s.resolution === 'hour' ? hours : days;
  if (!s.route_ids.length || s.route_ids.length > b.limits.max_routes || new Set(s.route_ids).size !== s.route_ids.length) throw new Error('Недопустимое количество маршрутов.');
  if (hours > b.limits.max_window_hours || hours * s.route_ids.length > b.limits.max_hourly_cells || count * s.route_ids.length > b.limits.max_response_cells) throw new Error('Период превышает лимит расчёта. Выберите меньший период или меньше маршрутов.');
  if (s.spatial_detail === 'route_stop' && (!b.capabilities.stop_forecasts || !b.capabilities.forecast_scopes.includes('route_stop'))) throw new Error('Остановочные прогнозы недоступны.');
  if (s.spatial_detail === 'route_stop' && stopCount !== undefined && count * stopCount > b.limits.max_stop_cells) throw new Error(`Превышен лимит остановочных кадров: ${b.limits.max_stop_cells}.`);
  for (const id of s.route_ids) {
    const c = b.active_snapshot.coverage.find(c => c.route_id === id);
    if (!c || from < moscow(c.window.from) || to > moscow(c.window.to) || !c.resolutions.includes(s.resolution) || !c.prediction_scopes.includes(s.spatial_detail)) throw new Error(`Маршрут ${id}: выбранный период или детализация вне доступного покрытия snapshot.`);
  }
}
export { sameWindow } from '../../state/contractTime';
export const dateLabel = (value: string, format = 'dd.MM.yy') => moscow(value).setLocale('ru').toFormat(format);
