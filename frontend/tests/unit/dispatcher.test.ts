import { describe, it, expect } from 'vitest';
import { bootstrap, calculate, geometryFor } from '../../scripts/mock';
import { makeWindow, frameWindows, validateSelection, moscow } from '../../src/features/time/dates';
import { LatestRequest, assertCalculation, selectReading } from '../../src/state/calculation';
import { nextDaySelection, overridesForDay, serviceFrames } from '../../src/features/time/serviceDay';
import { approach, buildColumns, surroundingAnchors, buildRouteRibbon, MIN_ROOF_HEIGHT, isTopDown } from '../../src/features/map/columns';
import type { Selection } from '../../src/api/types';
const selection=(mode:Selection['view_mode'],start:string,end?:string):Selection=>({...bootstrap.default_selection,view_mode:mode,resolution:mode==='day'?'hour':'day',window:makeWindow(mode,start,end)}) as Selection;
describe('Moscow calendar and coverage',()=>{
 it.each([['2024-02-14',29],['2025-02-14',28],['2026-04-14',30],['2026-01-14',31]])('month %s has %i daily frames',(date,count)=>{const s=selection('month',date);expect(frameWindows(s)).toHaveLength(count);expect(moscow(s.window.from).day).toBe(1);validateSelection(s,bootstrap,3);});
 it('day is exactly 24 Moscow hours regardless of host timezone',()=>{const s=selection('day','2026-09-27');expect(s.window.from).toBe('2026-09-27T00:00:00+03:00');expect(frameWindows(s)).toHaveLength(24);});
 it('week crosses year boundary',()=>{const s=selection('week','2026-12-28');expect(frameWindows(s)).toHaveLength(7);expect(s.window.to).toBe('2027-01-04T00:00:00+03:00');});
 it('custom spans three months including leap day and inclusive last date',()=>{const s=selection('custom','2024-01-31','2024-03-01');expect(frameWindows(s)).toHaveLength(31);expect(s.window.to).toBe('2024-03-02T00:00:00+03:00');});
 it('custom rejects unsupported three-month selection instead of clipping',()=>{const s=selection('custom','2026-01-30','2026-05-01');expect(()=>validateSelection(s,bootstrap,3)).toThrow();expect(frameWindows(s)).toHaveLength(92);});
 it('rejects reversed dates',()=>expect(()=>makeWindow('custom','2026-03-02','2026-03-01')).toThrow());
 it('rejects custom hour',()=>{const s={...selection('custom','2026-01-01'),resolution:'hour'} as Selection;expect(()=>validateSelection(s,bootstrap)).toThrow();});
 it('honors deployment-specific custom limit',()=>{const b=structuredClone(bootstrap);b.limits.max_custom_days=12;expect(()=>validateSelection(selection('custom','2026-01-01','2026-01-13'),b)).toThrow('12');});
 it('rejects stop cells overflow',()=>{const b=structuredClone(bootstrap);b.limits.max_stop_cells=60;expect(()=>validateSelection(selection('day','2026-01-01'),b,3)).toThrow('60');});
 it('rejects response cells overflow',()=>{const b=structuredClone(bootstrap);b.limits.max_response_cells=4;expect(()=>validateSelection(selection('week','2026-01-01'),b,3)).toThrow();});
 it('rejects unsupported stop capability',()=>{const b=structuredClone(bootstrap);b.capabilities.stop_forecasts=false;expect(()=>validateSelection(bootstrap.default_selection,b,3)).toThrow();});
 it('rejects missing route coverage without clipping',()=>{const s=selection('custom','2023-12-31','2024-01-01');expect(()=>validateSelection(s,bootstrap)).toThrow('покрытия');});
});
describe('request identity and atomic data',()=>{
 it('aborts an older request and invalidates late completion',()=>{const gate=new LatestRequest();const a=gate.next(),b=gate.next();expect(a.signal.aborted).toBe(true);expect(a.isCurrent()).toBe(false);expect(b.isCurrent()).toBe(true);gate.cancel();expect(b.isCurrent()).toBe(false);});
 it('rejects partial frames',async()=>{const s=selection('day','2026-09-27');const c=await calculate(s);c.frames.pop();expect(()=>assertCalculation(c,{kind:'forecast',selection:s})).toThrow('неполный');});
 it('rejects mixed network versions',async()=>{const s=bootstrap.default_selection,c=await calculate(s),g=geometryFor(s.route_ids[0]);g.network_version='different';expect(()=>assertCalculation(c,{kind:'forecast',selection:s},g)).toThrow('геометрии');});
 it('joins repeated physical stops using full route-pattern-stop identity',async()=>{const c=await calculate(bootstrap.default_selection);const r=c.frames[0].routes[0];r.stop_readings[1].stop_id=r.stop_readings[0].stop_id;const s=r.stop_readings[1];const focus={kind:'route_stop' as const,route_id:s.route_id,route_pattern_id:s.route_pattern_id,route_stop_id:s.route_stop_id};expect(selectReading(c,0,r.route_id,focus)).toBe(s);expect(selectReading(c,0,r.route_id,{...focus,route_pattern_id:'other'})).toBeUndefined();expect(s.evaluated.load_index).not.toEqual(r.evaluated.load_index);});
});
describe('scenario fixtures',()=>{
 it('fleet-only keeps all route and stop boardings unchanged, and zero is unavailable',async()=>{const s=bootstrap.default_selection;const c=await calculate(s,{effective_window:s.window,fleet:{kind:'absolute',vehicle_count:0}});for(const f of c.frames)for(const r of f.routes)for(const v of [r,...r.stop_readings]){expect(v.evaluated.boardings).toBe(v.baseline.boardings);expect(v.evaluated.load_index).toEqual({status:'unavailable',value:null,reason:'NO_SUPPLY'});}expect(c.totals[0].evaluated.load_index.status).toBe('unavailable');});
 it('applies an override only within explicit effective window',async()=>{const s=bootstrap.default_selection,w=frameWindows(s)[4];const c=await calculate(s,{effective_window:w,fleet:{kind:'absolute',vehicle_count:15}});expect(c.frames[4].routes[0].evaluated.mean_vehicle_count.value).toBe(15);expect(c.frames[3].routes[0].evaluated.mean_vehicle_count.value).toBe(10);});
 it('rejects negative or unknown baseline delta',async()=>{const s=bootstrap.default_selection;await expect(calculate(s,{effective_window:s.window,fleet:{kind:'delta',vehicle_count_delta:-11}})).rejects.toThrow();await expect(calculate(s,{effective_window:s.window,fleet:{kind:'delta',vehicle_count_delta:1}},'unknown-fleet')).rejects.toThrow();});
 it('produces deterministic calculation IDs and data',async()=>{expect(await calculate(bootstrap.default_selection)).toEqual(await calculate(bootstrap.default_selection));});
 it('route-only never supplies fake stop values',async()=>{const c=await calculate({...bootstrap.default_selection,spatial_detail:'route'},undefined,'no-stops');expect(c.frames.every(f=>f.routes.every(r=>!r.stop_readings.length))).toBe(true);});
});
describe('decorative approach geometry',()=>{
 it('places the ramp before the stop on direction geometry',()=>{const line=[[37.6,55.75],[37.601,55.75],[37.602,55.75]];const points=approach(line,line[2],40)!;expect(points).toHaveLength(7);expect(points[0][0]).toBeLessThan(points[6][0]);expect(points[6]).toEqual(line[2]);});
 it('does not invent an approach before the first stop',()=>{expect(approach([[37.6,55.75],[37.61,55.75]],[37.6,55.75],40)).toBeNull();});
 it('does not guess on self-intersection or repeated stop occurrence',()=>{const line=[[37.6,55.75],[37.61,55.75],[37.6,55.75],[37.61,55.75]];expect(approach(line,[37.605,55.75],20)).toBeNull();});
 it('clips only height, preserving source value and overflow',async()=>{const c=await calculate(bootstrap.default_selection);const stop=c.frames[0].routes[0].stop_readings[1];stop.evaluated.load_index={status:'available',value:7};const {data}=buildColumns(geometryFor('demo-1'),c,0);const bars=data.features.filter(f=>f.properties?.route_stop_id===stop.route_stop_id);expect(bars).toHaveLength(9);expect(bars.every(f=>f.properties?.value===7&&f.properties?.overflow&&f.properties?.height<=0.4*(MIN_ROOF_HEIGHT+305))).toBe(true);expect(stop.evaluated.load_index.value).toBe(7);});
 it('does not draw unavailable as zero',async()=>{const s=bootstrap.default_selection,c=await calculate(s,{effective_window:s.window,fleet:{kind:'absolute',vehicle_count:0}});expect(buildColumns(geometryFor('demo-1'),c,0).data.features).toHaveLength(0);});
});

describe('service day from 06 to 01',()=>{
 it('uses all 24 actual hours when next-day coverage is unavailable',async()=>{const c=await calculate(selection('day','2026-12-31'));expect(serviceFrames(c)).toHaveLength(24);expect(serviceFrames(c)[0].index).toBe(0);});
 it('uses real next-day frames across month and year boundaries',async()=>{
  const s=selection('day','2026-12-31'),next=nextDaySelection(s)!;
  const [c,n]=await Promise.all([calculate(s),calculate(next)]);
  const frames=serviceFrames(c,n);
  expect(frames).toHaveLength(20);
  expect(frames[0].calculation.frames[frames[0].index].window.from).toBe('2026-12-31T06:00:00+03:00');
  expect(frames[18].calculation).toBe(n);
  expect(frames[19].calculation.frames[frames[19].index].window.from).toBe('2027-01-01T01:00:00+03:00');
  expect(frames[18].calculation.frames[0].routes[0].evaluated.boardings).not.toBe(c.frames[0].routes[0].evaluated.boardings);
 });
 it('sends an after-midnight scenario only to its actual calendar day',()=>{
  const s=selection('day','2026-09-27'),next=nextDaySelection(s)!;
  const o={effective_window:frameWindows(next)[1],fleet:{kind:'absolute' as const,vehicle_count:99}};
  expect(overridesForDay(s,o)).toBeUndefined();expect(overridesForDay(next,o)).toBe(o);
 });
 it('keeps daily frames for week mode',async()=>{
  const s=selection('week','2026-09-27');expect(nextDaySelection(s)).toBeUndefined();
  expect(serviceFrames(await calculate(s))).toHaveLength(7);
 });
});
describe('raised map geometry',()=>{
 it('places four samples on each side along a bent road',()=>{
  const line=[[37.6,55.75],[37.601,55.75],[37.601,55.752]];
  const anchors=surroundingAnchors(line,line[1],72)!;
  expect(anchors).toHaveLength(9);expect(anchors[4]).toEqual(line[1]);
  expect(anchors[0][1]).toBe(55.75);expect(anchors[8][0]).toBe(37.601);
 });
 it('reduces central column height by 60% without changing the source value',async()=>{
  const c=await calculate(bootstrap.default_selection);for(const r of c.frames[0].routes[0].stop_readings)r.evaluated.load_index={status:'available',value:.2};
  const result=buildColumns(geometryFor('demo-1'),c,0,undefined,()=>180);
  expect(result.pins.length).toBe(4);expect(result.pins.every(p=>Math.abs(p.height-0.4*(185+(.2/c.visualization.scale_max)*300))<.001&&p.value===.2)).toBe(true);
  expect(result.data.features.filter(f=>f.properties?.central).every(f=>f.geometry.coordinates[0].length===5)).toBe(true);
 });
 it('creates a closed ground ribbon for every supplied route segment',()=>{
  const g=geometryFor('demo-1'),ribbon=buildRouteRibbon(g);
  expect(ribbon.features.length).toBeGreaterThan(0);
  for(const f of ribbon.features)expect(f.geometry.coordinates[0][0]).toEqual(f.geometry.coordinates[0].at(-1));
 });
});

describe('overhead map mode',()=>{
 it('includes the five-degree boundary and leaves overhead mode beyond it',()=>{
  for(const pitch of [0,4.99,5,-5])expect(isTopDown(pitch)).toBe(true);
  for(const pitch of [5.01,15,55])expect(isTopDown(pitch)).toBe(false);
 });
});
