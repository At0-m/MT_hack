const HOUR = 3600000, DAY = 24 * HOUR, MOSCOW = 3 * HOUR;
export function durationSeconds(text) {
  const m = /^(\d+)(ms|s|m|h)$/.exec(String(text));
  if (!m) throw new Error(`Invalid duration ${text}; use e.g. 30s, 5m, 1h`);
  return Number(m[1]) * ({ms: 0.001, s: 1, m: 60, h: 3600}[m[2]]);
}
export function integer(value, name, min, max) {
  const n = Number(value);
  if (!Number.isInteger(n) || n < min || n > max) throw new Error(`${name} must be ${min}..${max}`);
  return n;
}
function midnightAtOrAfter(ms) { return Math.ceil((ms + MOSCOW) / DAY) * DAY - MOSCOW; }
function iso(ms) { return new Date(ms).toISOString().replace('.000Z', 'Z'); }
export function selection(bootstrap, {view = 'day', routeCount = 1, routeIds = [], dayOffset = 0} = {}) {
  if (!bootstrap || !bootstrap.active_snapshot) throw new Error('bootstrap.active_snapshot missing');
  const snap = bootstrap.active_snapshot, p = snap.provenance;
  if (!['day', 'week', 'month'].includes(view) || !snap.view_modes.includes(view)) throw new Error(`View ${view} not published`);
  integer(routeCount, 'routeCount', 1, 10); integer(dayOffset, 'dayOffset', 0, 365);
  const resolution = view === 'day' ? 'hour' : 'day';
  let coverage = snap.coverage.filter(c => c.prediction_scopes.includes('route') && c.resolutions.includes(resolution));
  if (routeIds.length) {
    if (new Set(routeIds).size !== routeIds.length) throw new Error('Duplicate route IDs');
    coverage = routeIds.map(id => {
      const c = coverage.find(v => v.route_id === id);
      if (!c) throw new Error(`Route ${id} does not support ${view}/route`);
      return c;
    });
  } else coverage = coverage.slice().sort((a,b) => a.route_id.localeCompare(b.route_id)).slice(0, routeCount);
  if (coverage.length !== (routeIds.length || routeCount) || coverage.length > 10) throw new Error('Insufficient routes; refusing to reduce route count');
  const origin = Date.parse(p.forecast_origin_at);
  const fromMin = Math.max(origin, ...coverage.map(c => Date.parse(c.window.from)));
  const toMax = Math.min(...coverage.map(c => Math.min(Date.parse(c.window.to), origin + c.max_lead_hours * HOUR)));
  if (![origin, fromMin, toMax].every(Number.isFinite)) throw new Error('Invalid coverage timestamps');
  let from = midnightAtOrAfter(fromMin) + dayOffset * DAY, to;
  if (view === 'month') {
    if (dayOffset !== 0) throw new Error('DAY_OFFSET does not apply to calendar-month benchmark');
    const local = new Date(from + MOSCOW);
    const first = Date.UTC(local.getUTCFullYear(), local.getUTCMonth(), 1) - MOSCOW;
    from = first < from ? Date.UTC(local.getUTCFullYear(), local.getUTCMonth() + 1, 1) - MOSCOW : first;
    const month = new Date(from + MOSCOW);
    to = Date.UTC(month.getUTCFullYear(), month.getUTCMonth() + 1, 1) - MOSCOW;
  } else to = from + (view === 'day' ? 1 : 7) * DAY;
  if (to > toMax || from < fromMin) throw new Error(`No complete ${view} in common coverage; refusing to shorten window`);
  const cells = ((to - from) / HOUR) * coverage.length;
  const responseCells = cells / (resolution === 'hour' ? 1 : 24);
  if (cells > 7440 || responseCells > 960) throw new Error('Selection exceeds server cell limits');
  return {forecast_snapshot_id: p.forecast_snapshot_id, route_ids: coverage.map(c => c.route_id).sort(), view_mode: view,
    window: {from: iso(from), to: iso(to)}, resolution, spatial_detail: 'route'};
}
export function framesExpected(s) { return (Date.parse(s.window.to) - Date.parse(s.window.from)) / (s.resolution === 'hour' ? HOUR : DAY); }
export function validForecast(body, s) {
  if (!body || !/^calc_[a-f0-9]{64}$/.test(body.calculation_id || '') ||
      body.provenance?.forecast_snapshot_id !== s.forecast_snapshot_id ||
      !Array.isArray(body.frames) || body.frames.length !== framesExpected(s) ||
      !Array.isArray(body.totals) || body.totals.length !== s.route_ids.length) return false;
  const expected = s.route_ids.slice().sort().join(',');
  const validReadings = readings => Array.isArray(readings) && readings.map(r => r.route_id).sort().join(',') === expected &&
    readings.every(r => [r.baseline?.boardings, r.evaluated?.boardings].every(v => typeof v === 'number' && Number.isFinite(v) && v >= 0));
  if (!validReadings(body.totals)) return false;
  let cursor = Date.parse(s.window.from);
  const step = s.resolution === 'hour' ? HOUR : DAY;
  for (const frame of body.frames) {
    if (Date.parse(frame.window?.from) !== cursor || Date.parse(frame.window?.to) !== cursor + step || !validReadings(frame.routes)) return false;
    cursor += step;
  }
  return cursor === Date.parse(s.window.to);
}
export function mixedEndpoint(index) {
  const i = (index * 37) % 100;
  if (i < 60) return 'day';
  if (i < 65) return 'bootstrap';
  if (i < 70) return 'routes';
  if (i < 75) return 'geometry';
  if (i < 90) return 'scenario';
  if (i < 95) return i % 2 ? 'week' : 'month';
  return i % 2 ? 'export' : 'summary';
}
