import http from 'k6/http';
import { check, fail } from 'k6';
import { selection, validForecast } from './lib/selection.mjs';
import { BASE, login, bootstrap, params, decode, header, teardown as cleanup } from './lib/runner.js';
export const options = {vus:1, iterations:1, thresholds:{checks:['rate==1'],http_req_failed:['rate==0']},setupTimeout:'90s',systemTags:['method','name','status']};
export function setup() {const token=login();return {token,bootstrap:bootstrap(token)};}
function expect(res,status,label,valid=()=>true) {
  const ok=check(res,{[label]:r=>r.status===status&&valid(r)});
  if(!ok) fail(`${label}: status ${res.status}`);
  return decode(res);
}
export default function(data) {
  const token=data.token,b=data.bootstrap;
  for(const path of ['/health/live','/health/ready']) expect(http.get(BASE+path,params(null,path)),200,path);
  expect(http.get(`${BASE}/api/v1/auth/session`,params(null,'session-unauthorized','smoke',401)),401,'unauthorized session');
  expect(http.get(`${BASE}/api/v1/auth/session`,params(token,'session')),200,'authenticated session',r=>decode(r)?.authenticated===true);
  const s=selection(b);
  const network=encodeURIComponent(b.active_snapshot.provenance.network_version),route=encodeURIComponent(s.route_ids[0]);
  expect(http.get(`${BASE}/api/v1/routes?network_version=${network}`,params(token,'routes')),200,'routes',r=>decode(r)?.items?.length>0);
  expect(http.get(`${BASE}/api/v1/routes/${route}/geometry?network_version=${network}`,params(token,'geometry')),200,'geometry',r=>decode(r)?.features?.length>0);
  const forecast=expect(http.post(`${BASE}/api/v1/forecasts/query`,JSON.stringify({selection:s}),params(token,'day')),200,'24 hourly frames',r=>validForecast(decode(r),s));
  const q={selection:s,overrides:{effective_window:s.window,factors:{event:1.1}}};
  const scenario=expect(http.post(`${BASE}/api/v1/scenarios/evaluate`,JSON.stringify(q),params(token,'scenario')),200,'scenario',r=>validForecast(decode(r),s));
  const base=forecast.totals[0].baseline.boardings,evaluated=scenario.totals[0].evaluated.boardings;
  if(!check(scenario,{'scenario event multiplier 1.1':()=>Math.abs(evaluated-base*1.1)<=1e-6*Math.max(1,base)}))fail('Scenario arithmetic mismatch');
  const replay={calculation:scenario.descriptor,expected_calculation_id:scenario.calculation_id};
  const csv=http.post(`${BASE}/api/v1/exports`,JSON.stringify({...replay,format:'csv'}),params(token,'export'));
  expect(csv,200,'CSV identity and rows',r=>header(r,'content-type').includes('text/csv')&&header(r,'x-calculation-id')===scenario.calculation_id&&r.body.trim().split('\n').length===25);
  expect(http.post(`${BASE}/api/v1/summaries/query`,JSON.stringify({...replay,focus:{kind:'overall'}}),params(token,'summary')),200,'summary',r=>decode(r)?.calculation_id===scenario.calculation_id&&decode(r)?.status==='ready');
  const mismatch={...replay,expected_calculation_id:'calc_'+ '0'.repeat(64),format:'csv'};
  expect(http.post(`${BASE}/api/v1/exports`,JSON.stringify(mismatch),params(token,'mismatch','smoke',409)),409,'export mismatched calculation');
  expect(http.post(`${BASE}/api/v1/forecasts/query`,'{"selection":',params(token,'malformed','smoke',400)),400,'malformed JSON');
  expect(http.post(`${BASE}/api/v1/forecasts/query`,JSON.stringify({selection:{...s,window:{from:s.window.to,to:s.window.from}}}),params(token,'invalid-window','smoke',422)),422,'invalid window');
}
export function teardown(data){cleanup(data)}
export function handleSummary(data){return {'/out/k6-summary.json':JSON.stringify({format:'tramflow-smoke-v1',metrics:data.metrics},null,2)}}
