import type { Calculation, Descriptor, Geometry, StopFocus, Reading } from '../api/types';
import { frameWindows, sameWindow } from '../features/time/dates';
export function selectReading(c: Calculation, index: number, routeId: string, focus?: StopFocus): Reading | undefined {
  const route = c.frames[index]?.routes.find(r => r.route_id === routeId);
  if (!focus) return route;
  return route?.stop_readings.find(s => s.route_id === focus.route_id && s.route_pattern_id === focus.route_pattern_id && s.route_stop_id === focus.route_stop_id);
}
export function assertCalculation(c: Calculation, requested: Descriptor, geometry?: Geometry) {
  const a = c.descriptor.selection, b = requested.selection;
  if (c.descriptor.kind !== requested.kind || a.forecast_snapshot_id !== b.forecast_snapshot_id || a.view_mode !== b.view_mode || a.resolution !== b.resolution || a.spatial_detail !== b.spatial_detail || !sameWindow(a.window,b.window) || [...a.route_ids].sort().join() !== [...b.route_ids].sort().join() || c.provenance.forecast_snapshot_id !== b.forecast_snapshot_id) throw new Error('Ответ не соответствует запрошенному расчёту.');
  if (c.descriptor.kind === 'scenario' && requested.kind === 'scenario') {
    const x = c.descriptor.overrides, y = requested.overrides;
    if (!sameWindow(x.effective_window, y.effective_window) || JSON.stringify(x.fleet) !== JSON.stringify(y.fleet) || JSON.stringify(x.factors ?? {}) !== JSON.stringify(y.factors ?? {})) throw new Error('Ответ содержит другой сценарий.');
  }
  const windows = frameWindows(b);
  if (c.frames.length !== windows.length || c.frames.some((f,i) => !sameWindow(f.window,windows[i]) || [...f.routes.map(r=>r.route_id)].sort().join() !== [...b.route_ids].sort().join()) || [...c.totals.map(r=>r.route_id)].sort().join() !== [...b.route_ids].sort().join()) throw new Error('Получен неполный расчёт. Период не был применён.');
  if (geometry && geometry.network_version !== c.provenance.network_version) throw new Error('Версия геометрии не совпадает с расчётом.');
}
export class LatestRequest {
  private version = 0;
  private controller?: AbortController;
  next() { this.controller?.abort(); const version = ++this.version; const controller = this.controller = new AbortController(); return { signal: controller.signal, isCurrent: () => version === this.version && !controller.signal.aborted }; }
  cancel() { ++this.version; this.controller?.abort(); }
}
