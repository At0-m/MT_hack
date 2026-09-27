import type { Calculation, Descriptor, Geometry, StopFocus, Reading, Overrides } from '../api/types';
import { contractWindows, sameWindow } from './contractTime';
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
    if (!sameWindow(x.effective_window, y.effective_window) || !sameFleet(x.fleet, y.fleet) || !sameFactors(x.factors, y.factors)) throw new Error('Ответ содержит другой сценарий.');
  }
  const windows = contractWindows(b);
  if (c.frames.length !== windows.length || c.frames.some((f,i) => !sameWindow(f.window,windows[i]) || [...f.routes.map(r=>r.route_id)].sort().join() !== [...b.route_ids].sort().join()) || [...c.totals.map(r=>r.route_id)].sort().join() !== [...b.route_ids].sort().join()) throw new Error('Получен неполный расчёт. Период не был применён.');
  if (geometry && geometry.network_version !== c.provenance.network_version) throw new Error('Версия геометрии не совпадает с расчётом.');
  if (geometry && b.spatial_detail==='route_stop') {
    const expected=geometry.features.filter(f=>f.properties.kind==='stop').map(f=>{
      const p=f.properties; return p.kind==='stop' ? `${p.route_id}|${p.route_pattern_id}|${p.route_stop_id}|${p.stop_id}|${p.sequence}` : '';
    }).sort().join('\n');
    for(const routes of [...c.frames.map(f=>f.routes),c.totals]){
      const route=routes.find(r=>r.route_id===geometry.route_id);
      const actual=route?.stop_readings.map(s=>`${s.route_id}|${s.route_pattern_id}|${s.route_stop_id}|${s.stop_id}|${s.sequence}`).sort().join('\n');
      if(!expected || expected!==actual)throw new Error('Stop predictions do not match the pinned geometry.');
    }
  }

}
export class LatestRequest {
  private version = 0;
  private controller?: AbortController;
  next() { this.controller?.abort(); const version = ++this.version; const controller = this.controller = new AbortController(); return { signal: controller.signal, isCurrent: () => version === this.version && !controller.signal.aborted }; }
  cancel() { ++this.version; this.controller?.abort(); }
}

/** The backend normalizes omitted factors to 1. Object key order is not semantic. */
export function sameFactors(a: Overrides['factors'], b: Overrides['factors']): boolean {
  return (['weather', 'event', 'season'] as const).every(key =>
    Math.round((a?.[key] ?? 1) * 1000) === Math.round((b?.[key] ?? 1) * 1000));
}
export function sameFleet(a: Overrides['fleet'], b: Overrides['fleet']): boolean {
  if (!a || !b) return a === b;
  if (a.kind !== b.kind) return false;
  return a.kind === 'absolute' && b.kind === 'absolute'
    ? a.vehicle_count === b.vehicle_count
    : a.kind === 'delta' && b.kind === 'delta' && a.vehicle_count_delta === b.vehicle_count_delta;
}
