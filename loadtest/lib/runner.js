import http from 'k6/http';
import exec from 'k6/execution';
import { check, fail } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';
import { selection, validForecast, mixedEndpoint, durationSeconds, integer } from './selection.mjs';

const latency = new Trend('business_latency', true);
const wall = new Trend('business_client_wall', true);
const requests = new Counter('business_requests');
const successful = new Counter('business_successes');
const failures = new Rate('business_failures');
const started = new Trend('measurement_request_start_unix_ms');
const ended = new Trend('measurement_request_end_unix_ms');
const BASE = (__ENV.BASE_URL || 'http://gateway:8080').replace(/\/$/, '');
const tags = ['method','name','status','scenario','expected_response'];
export function header(res, name) {
  const key = Object.keys(res.headers || {}).find(k => k.toLowerCase() === name.toLowerCase());
  return key ? res.headers[key] : '';
}
export function params(token, name, phase = 'setup', expected = 200) {
  return {headers: {'Content-Type': 'application/json', ...(token ? {Cookie: `tramflow_session=${token}`} : {})},
    tags: {name, endpoint: name, phase}, redirects: 0, timeout: __ENV.REQUEST_TIMEOUT || '12s',
    responseCallback: http.expectedStatuses(expected)};
}
export function decode(res) { try { return res.json(); } catch (_) { return null; } }
export function login() {
  if (!__ENV.API_USER || !__ENV.API_PASSWORD) fail('API_USER and API_PASSWORD are required');
  const res = http.post(`${BASE}/api/v1/auth/login`, JSON.stringify({username: __ENV.API_USER, password: __ENV.API_PASSWORD}), params(null,'login'));
  const token = res.cookies?.tramflow_session?.[0]?.value;
  if (res.status !== 200 || !token || decode(res)?.authenticated !== true) fail(`Login failed with status ${res.status}; do not bypass rate limits`);
  return token;
}
export function bootstrap(token) {
  const res = http.get(`${BASE}/api/v1/bootstrap`, params(token,'bootstrap'));
  const b = decode(res);
  if (res.status !== 200 || !b?.active_snapshot?.provenance) fail(`Bootstrap failed: ${res.status}`);
  if (__ENV.EXPECTED_RUNTIME_MODE && b.active_snapshot.provenance.runtime_mode !== __ENV.EXPECTED_RUNTIME_MODE) fail('Unexpected runtime_mode');
  if (__ENV.EXPECTED_MODEL_VERSION && b.active_snapshot.provenance.model_version !== __ENV.EXPECTED_MODEL_VERSION) fail('Unexpected model version');
  return b;
}
function config(kind) {
  const warmup = __ENV.WARMUP || '2m', duration = __ENV.DURATION || '5m';
  const rate = integer(__ENV.RATE || 50,'RATE',1,10000);
  const pre = integer(__ENV.PRE_VUS || Math.max(20, Math.ceil(rate * .5)), 'PRE_VUS', 1, 10000);
  const max = integer(__ENV.MAX_VUS || Math.max(100, rate * 2), 'MAX_VUS', pre, 20000);
  if (durationSeconds(duration) <= 0) throw new Error('DURATION must be positive');
  return {kind, warmup, duration, rate, pre, max, warmSeconds: durationSeconds(warmup)};
}
export function optionsFor(kind) {
  const c = config(kind);
  const scenario = {executor: 'constant-arrival-rate', rate: c.rate, timeUnit: '1s', preAllocatedVUs: c.pre, maxVUs: c.max, gracefulStop: '15s'};
  const scenarios = {};
  if (c.warmSeconds > 0) scenarios.warmup = {...scenario, duration: c.warmup};
  scenarios.measure = {...scenario, duration: c.duration, startTime: `${c.warmSeconds > 0 ? c.warmSeconds + 15 : 0}s`};
  const thresholds = {
    'business_requests{phase:measurement}': ['count>0'],
    'business_successes{phase:measurement}': ['count>=0'],
    'business_failures{phase:measurement}': ['rate<0.001'],
    'business_latency{phase:measurement}': [`p(95)<${__ENV.P95_MS || 250}`],
    'business_client_wall{phase:measurement}': [`p(95)<${__ENV.WALL_P95_MS || 300}`],
    'checks{phase:measurement}': ['rate>0.999'],
    'dropped_iterations{scenario:measure}': ['count==0'],
    'iterations{scenario:measure}': [`count>=${Math.max(1, Math.floor(c.rate * durationSeconds(c.duration)) - 1)}`],
  };
  const endpoints = kind === 'mixed' ? ['day','bootstrap','routes','geometry','scenario','week','month','summary','export'] : [kind];
  for (const endpoint of endpoints) {
    thresholds[`business_latency{phase:measurement,endpoint:${endpoint}}`] = [`p(95)<${__ENV.P95_MS || 250}`];
    thresholds[`business_requests{phase:measurement,endpoint:${endpoint}}`] = ['count>0'];
    thresholds[`business_failures{phase:measurement,endpoint:${endpoint}}`] = ['rate<0.001'];
  }
  if (kind === 'day' || kind === 'mixed') thresholds['business_latency{phase:measurement,endpoint:day}'] = [`p(95)<${__ENV.FORECAST_P95_MS || 200}`];
  return {scenarios, thresholds, setupTimeout: '90s', teardownTimeout: '15s', systemTags: tags,
    summaryTrendStats: ['avg','min','med','max','p(90)','p(95)','p(99)'], userAgent: 'tramflow-authorized-loadtest'};
}
export function setupFor(kind) {
  const token = login(), b = bootstrap(token);
  const routeCount = integer(__ENV.ROUTE_COUNT || 1, 'ROUTE_COUNT', 1, 10);
  const routeIds = (__ENV.ROUTE_IDS || '').split(',').filter(Boolean);
  const dayOffset = integer(__ENV.DAY_OFFSET || 0, 'DAY_OFFSET', 0, 365);
  const data = {token, bootstrap: b, selections: {}, replay: null};
  const required = kind === 'mixed' ? ['day','week','month'] : [ ['scenario','export','summary','bootstrap','routes','geometry'].includes(kind) ? 'day' : kind ];
  for (const view of required) data.selections[view] = selection(b,{view,routeCount,routeIds,dayOffset:view === 'month' ? 0 : dayOffset});
  if (kind === 'mixed' || kind === 'export' || kind === 'summary') {
    const s = data.selections.day;
    const res = http.post(`${BASE}/api/v1/forecasts/query`,JSON.stringify({selection:s}),params(token,'prepare-replay'));
    const body = decode(res);
    if (res.status !== 200 || !validForecast(body,s)) fail('Cannot prepare exact descriptor for summary/export');
    data.replay = {calculation:body.descriptor, expected_calculation_id:body.calculation_id};
  }
  console.log(`PROVENANCE ${JSON.stringify(b.active_snapshot.provenance)}`);
  return data;
}
export function teardown(data) {
  if (data?.token) http.post(`${BASE}/api/v1/auth/logout`,null,params(data.token,'logout','teardown',204));
}
function operation(endpoint, data) {
  const token = data.token, phase = exec.scenario.name === 'measure' ? 'measurement' : 'warmup';
  const p = params(token,endpoint,phase);
  const b = data.bootstrap, network = encodeURIComponent(b.active_snapshot.provenance.network_version);
  const route = encodeURIComponent((data.selections.day || Object.values(data.selections)[0]).route_ids[0]);
  const s = data.selections[endpoint] || data.selections.day;
  let path, body = null, validate;
  switch (endpoint) {
    case 'bootstrap': path='/api/v1/bootstrap'; validate=x=>!!x?.active_snapshot?.provenance; break;
    case 'routes': path=`/api/v1/routes?network_version=${network}`; validate=x=>Array.isArray(x?.items)&&x.items.length>0; break;
    case 'geometry': path=`/api/v1/routes/${route}/geometry?network_version=${network}`; validate=x=>x?.type==='FeatureCollection'&&x.features?.length>0; break;
    case 'export': path='/api/v1/exports'; body={...data.replay,format:'csv'}; break;
    case 'summary': path='/api/v1/summaries/query'; body={...data.replay,focus:{kind:'overall'}}; validate=x=>x?.status==='ready'&&x.calculation_id===data.replay.expected_calculation_id; break;
    case 'scenario': path='/api/v1/scenarios/evaluate'; body={selection:s,overrides:{effective_window:s.window,factors:{event:1.1}}}; validate=x=>validForecast(x,s)&&x.descriptor?.kind==='scenario'; break;
    default: path='/api/v1/forecasts/query'; body={selection:s}; validate=x=>validForecast(x,s);
  }
  const metricTags = {phase,endpoint};
  const encoded = body === null ? null : JSON.stringify(body);
  const t0 = Date.now();
  if (phase === 'measurement') started.add(t0);
  const res = body === null ? http.get(BASE+path,p) : http.post(BASE+path,encoded,p);
  const t1 = Date.now();
  requests.add(1,metricTags); 
  if (phase === 'measurement') ended.add(t1);
  let ok = res.status === 200;
  if (ok && endpoint === 'export') ok = String(header(res,'content-type')).includes('text/csv') &&
      header(res,'x-calculation-id') === data.replay.expected_calculation_id && res.body.includes('baseline_boardings');
  else if (ok) ok = validate(decode(res));
  check(res,{ 'business status and response contract':()=>ok },metricTags);
  failures.add(!ok,metricTags); if (ok) successful.add(1,metricTags);
  latency.add(res.timings.duration,metricTags); wall.add(t1-t0,metricTags);
}
export function run(kind, data) {
  operation(kind === 'mixed' ? mixedEndpoint(exec.scenario.iterationInTest) : kind, data);
}
export function summaryFor(kind, data) {
  const result = {format:'tramflow-k6-v1', configuration:config(kind), metrics:data.metrics, state:data.state,
    thresholds_passed: Object.values(data.metrics).every(m=>Object.values(m.thresholds || {}).every(t=>t.ok))};
  return {'/out/k6-summary.json': JSON.stringify(result,null,2), stdout: `\n${kind}: thresholds_passed=${result.thresholds_passed}; detailed summary saved\n`};
}
export { BASE };
