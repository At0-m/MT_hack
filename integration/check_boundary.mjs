import assert from 'node:assert/strict';
import fs from 'node:fs';
import {fileURLToPath} from 'node:url';
import {assertCalculation, sameFactors, sameFleet, LatestRequest, selectReading} from './.tmp/ts-boundary/calculation.js';
const dir=fileURLToPath(new URL('./fixtures/',import.meta.url));
const read=name=>JSON.parse(fs.readFileSync(`${dir}/${name}.json`,'utf8'));
const geometry=read('geometry');let tests=0;
function test(name,fn){fn();tests++;console.log(`PASS ${name}`);}
for(const name of ['day','fleet','factors','zero','month','last-day']){
 test(`Go ${name} accepted by actual frontend assertion`,()=>assertCalculation(read(name),read(name+'-request'),geometry));
}
test('all periods preserve expected frame counts',()=>{assert.equal(read('day').frames.length,24);assert.equal(read('month').frames.length,30);});
test('neutral factor normalization not a false mismatch',()=>assert(sameFactors(undefined,{weather:1,event:1,season:1})));
test('changed demand multiplier is rejected',()=>{const c=read('factors');c.descriptor.overrides.factors.weather=1.3;assert.throws(()=>assertCalculation(c,read('factors-request'),geometry));});
test('fleet object order is immaterial',()=>assert(sameFleet({kind:'absolute',vehicle_count:12},{vehicle_count:12,kind:'absolute'})));
test('direction mismatch is rejected',()=>{const c=read('day');c.frames[0].routes[0].stop_readings[0].route_pattern_id='wrong';assert.throws(()=>assertCalculation(c,read('day-request'),geometry));});
test('incomplete stop array is rejected',()=>{const c=read('day');c.frames[1].routes[0].stop_readings.pop();assert.throws(()=>assertCalculation(c,read('day-request'),geometry));});
test('incomplete route frames are rejected',()=>{const c=read('day');c.frames.pop();assert.throws(()=>assertCalculation(c,read('day-request'),geometry));});
test('stale network version is rejected',()=>{const g={...geometry,network_version:'stale'};assert.throws(()=>assertCalculation(read('day'),read('day-request'),g));});
test('zero fleet never turns into zero load index',()=>{const c=read('zero');for(const s of c.frames[0].routes[0].stop_readings){assert.equal(s.evaluated.load_index.status,'unavailable');assert.equal(s.evaluated.load_index.value,null);assert.equal(s.evaluated.load_index.reason,'NO_SUPPLY');}});
test('fleet scenario never changes boardings',()=>{const c=read('fleet');for(const f of c.frames)for(const r of f.routes)for(const row of [r,...r.stop_readings])assert.equal(row.evaluated.boardings,row.baseline.boardings);});
test('demand factors apply to stops only inside the window',()=>{const c=read('factors');for(const [i,f] of c.frames.entries())for(const s of f.routes[0].stop_readings)assert(Math.abs(s.evaluated.boardings-s.baseline.boardings*(i===0?1.2:1))<1e-8);});
test('summary is bound to exact calculation',()=>{for(const n of ['day','fleet','factors','month']){const s=read(n+'-summary'),c=read(n);assert.equal(s.status,'ready');assert.equal(s.calculation_id,c.calculation_id);assert.deepEqual(s.window,c.descriptor.selection.window);}});
test('stale requests cannot overwrite confirmed state',()=>{const gate=new LatestRequest(),a=gate.next(),b=gate.next();assert(a.signal.aborted);assert(!a.isCurrent());assert(b.isCurrent());gate.cancel();assert(!b.isCurrent());});
test('full stop identity selects independent values',()=>{const c=read('day'),s=c.frames[0].routes[0].stop_readings[1];assert.equal(selectReading(c,0,s.route_id,{kind:'route_stop',route_id:s.route_id,route_pattern_id:s.route_pattern_id,route_stop_id:s.route_stop_id}),s);});
console.log(`TOTAL ${tests} passed; real HTTP/React/ONNX not exercised by this boundary test.`);
