import test from 'node:test';
import assert from 'node:assert/strict';
import {selection,framesExpected,mixedEndpoint,durationSeconds,validForecast} from '../lib/selection.mjs';
function fixture(from='2026-09-30T21:00:00Z',to='2026-10-31T21:00:00Z',n=10){
 return {active_snapshot:{provenance:{forecast_snapshot_id:'test-v1',forecast_origin_at:from},view_modes:['day','week','month'],coverage:Array.from({length:n},(_,i)=>({route_id:`r${i}`,window:{from,to},resolutions:['hour','day'],prediction_scopes:['route'],max_lead_hours:744}))}};
}
test('one calendar day has 24 hourly frames',()=>{const s=selection(fixture());assert.equal(framesExpected(s),24);assert.equal(s.window.from,'2026-09-30T21:00:00Z');assert.equal(s.window.to,'2026-10-01T21:00:00Z')});
test('October is 31 calendar days, not 30',()=>{const s=selection(fixture(),{view:'month',routeCount:10});assert.equal(framesExpected(s),31);assert.equal(s.route_ids.length,10)});
test('February leap year has 29 days',()=>{const b=fixture('2024-01-31T21:00:00Z','2024-02-29T21:00:00Z');assert.equal(framesExpected(selection(b,{view:'month'})),29)});
test('week is 7 days at daily resolution',()=>assert.equal(framesExpected(selection(fixture(),{view:'week'})),7));
test('partial month is rejected',()=>assert.throws(()=>selection(fixture('2026-10-02T21:00:00Z'),{view:'month'}),/No complete/));
test('route count is never silently reduced',()=>assert.throws(()=>selection(fixture(undefined,undefined,2),{routeCount:10}),/Insufficient/));
test('common coverage intersection is enforced',()=>{const b=fixture();b.active_snapshot.coverage[1].window.to='2026-10-10T21:00:00Z';assert.throws(()=>selection(b,{view:'month',routeCount:2}),/No complete/)});
test('max lead is enforced',()=>{const b=fixture();b.active_snapshot.coverage[0].max_lead_hours=12;assert.throws(()=>selection(b),/No complete/)});
test('unpublished view rejected',()=>{const b=fixture();b.active_snapshot.view_modes=['day'];assert.throws(()=>selection(b,{view:'month'}),/not published/)});
test('explicit routes preserved, duplicate rejected',()=>{assert.deepEqual(selection(fixture(),{routeIds:['r7','r1']}).route_ids,['r1','r7']);assert.throws(()=>selection(fixture(),{routeIds:['r1','r1']}),/Duplicate/)});
test('mixed shares sum to 100 business requests',()=>{const c={};for(let i=0;i<100;i++){const k=mixedEndpoint(i);c[k]=(c[k]||0)+1}assert.equal(c.day,60);assert.equal(c.scenario,15);assert.equal(c.bootstrap+c.routes+c.geometry,15);assert.equal(c.week+c.month,5);assert.equal(c.export+c.summary,5)});
test('duration parser rejects ambiguous strings',()=>{assert.equal(durationSeconds('2m'),120);assert.equal(durationSeconds('0s'),0);assert.throws(()=>durationSeconds('2 minutes'))});
test('empty or fake forecasts fail semantic checks',()=>{const s=selection(fixture());assert.equal(validForecast({},s),false);assert.equal(validForecast({calculation_id:'calc_0',frames:[],totals:[]},s),false)});
function responseFor(s) {
 const readings=()=>s.route_ids.map(route_id=>({route_id,baseline:{boardings:100},evaluated:{boardings:110}}));
 const step=s.resolution==='hour'?3600000:86400000;
 const from=Date.parse(s.window.from);
 return {calculation_id:'calc_'+ 'a'.repeat(64),provenance:{forecast_snapshot_id:s.forecast_snapshot_id},totals:readings(),
   frames:Array.from({length:framesExpected(s)},(_,i)=>({window:{from:new Date(from+i*step).toISOString(),to:new Date(from+(i+1)*step).toISOString()},routes:readings()}))};
}
test('complete numeric forecast passes semantic validation',()=>{const s=selection(fixture());assert.equal(validForecast(responseFor(s),s),true)});
test('wrong routes, negative values and broken time frames fail',()=>{
 const s=selection(fixture());
 for(const mutate of [b=>b.totals[0].route_id='wrong',b=>b.frames[0].routes[0].evaluated.boardings=-1,b=>b.frames[1].window.from=b.frames[0].window.from,b=>b.frames.pop()]) {
  const b=responseFor(s);mutate(b);assert.equal(validForecast(b,s),false);
 }
});
