import type { FeatureCollection, Polygon } from 'geojson';
import type { Calculation, Geometry, StopFeature } from '../../api/types';
export const loadColors = {normal:'#00ff00',elevated:'#f6ff00',high:'#ffb44a',very_high:'#ff0000',unavailable:'#bbc5d2'};
export function loadColor(value:number|undefined, thresholds: readonly number[] = [0.75, 1, 1.25]):string {
 if(value===undefined||!Number.isFinite(value))return loadColors.unavailable;
 const [low, medium, high] = thresholds;
 return value<low?loadColors.normal:value<medium?loadColors.elevated:value<high?loadColors.high:loadColors.very_high;
}
type Point = number[];
const meters = (a:Point,b:Point) => Math.hypot((a[0]-b[0])*111320*Math.cos(a[1]*Math.PI/180),(a[1]-b[1])*111320);
const lerp=(a:Point,b:Point,t:number)=>[a[0]+(b[0]-a[0])*t,a[1]+(b[1]-a[1])*t];
/** Ambiguous self-intersections and disconnected lines deliberately fall back to a single anchor. */
export function approach(line:Point[], stop:Point, length:number): Point[]|null {
 if(line.length<2||length<=0)return null;
 const candidates:{point:Point,distance:number,along:number}[]=[];let cumulative=0;
 for(let i=1;i<line.length;i++){
  const a=line[i-1],b=line[i],cos=Math.cos(stop[1]*Math.PI/180),dx=(b[0]-a[0])*cos,dy=b[1]-a[1];
  const t=Math.max(0,Math.min(1,((stop[0]-a[0])*cos*dx+(stop[1]-a[1])*dy)/(dx*dx+dy*dy||1)));
  const point=lerp(a,b,t),distance=meters(point,stop),segment=meters(a,b);
  if(distance<12)candidates.push({point,distance,along:cumulative+t*segment});cumulative+=segment;
 }
 candidates.sort((a,b)=>a.distance-b.distance);const hit=candidates[0];
 if(!hit||hit.along<length||candidates.some(c=>c.distance<hit.distance+2&&Math.abs(c.along-hit.along)>15))return null;
 return Array.from({length:7},(_,index)=>{
  const distance=hit.along-length+length*index/6;let progress=0;
  for(let i=1;i<line.length;i++){const segment=meters(line[i-1],line[i]);if(progress+segment>=distance)return lerp(line[i-1],line[i],(distance-progress)/(segment||1));progress+=segment;}
  return hit.point;
 });
}
// Successive delays of the original zoom-out interval: 20%, 2%, then 5%.
export const COLUMN_MIN_ZOOM = 16 - (16 - 14.5) * 1.2 * 1.02 * 1.05;
export const MIN_ROOF_HEIGHT = 120;
export const isTopDown = (pitch:number) => Math.abs(pitch) <= 5;
export type ColumnPin = { coordinates: number[]; height: number; value: number; color: string };

/** Symmetric samples on the supplied route only; never extend it beyond its endpoints. */
export function surroundingAnchors(line: Point[], stop: Point, length: number): Point[] | null {
 if(line.length<2)return null;
 let total=0;const segments=line.slice(1).map((p,i)=>{const start=total;total+=meters(line[i],p);return {a:line[i],b:p,start,length:total-start};});
 const hits=segments.map(s=>{
  const cos=Math.cos(stop[1]*Math.PI/180),dx=(s.b[0]-s.a[0])*cos,dy=s.b[1]-s.a[1];
  const t=Math.max(0,Math.min(1,((stop[0]-s.a[0])*cos*dx+(stop[1]-s.a[1])*dy)/(dx*dx+dy*dy||1)));
  const point=lerp(s.a,s.b,t);return {distance:meters(point,stop),along:s.start+s.length*t};
 }).sort((a,b)=>a.distance-b.distance);
 const hit=hits[0];
 if(hit.distance>20||hits.some(h=>h.distance<hit.distance+2&&Math.abs(h.along-hit.along)>15))return null;
 const sample=(distance:number)=>{const s=segments.find(s=>s.start+s.length>=distance)??segments[segments.length-1];return lerp(s.a,s.b,Math.max(0,Math.min(1,(distance-s.start)/(s.length||1))));};
 const before=Math.min(length/2,hit.along),after=Math.min(length/2,total-hit.along);
 return [-4,-3,-2,-1,0,1,2,3,4].filter(i=>i===0||(i<0?before:after)>0.5).map(i=>sample(hit.along+i*(i<0?before:after)/4));
}
function square(p:Point,halfWidth:number):Point[]{
 const dx=halfWidth/(111320*Math.cos(p[1]*Math.PI/180)),dy=halfWidth/111320;
 return [[p[0]-dx,p[1]-dy],[p[0]+dx,p[1]-dy],[p[0]+dx,p[1]+dy],[p[0]-dx,p[1]+dy],[p[0]-dx,p[1]-dy]];
}
export function buildRouteRibbon(geometry: Geometry, patternId?: string): FeatureCollection<Polygon> {
 const features:FeatureCollection<Polygon>['features']=[];
 for(const f of geometry.features){
  if(f.geometry.type!=='LineString'||patternId&&f.properties.route_pattern_id!==patternId)continue;
  const coordinates=f.geometry.coordinates;
  coordinates.slice(1).forEach((b,i)=>{
   const a=coordinates[i],cos=Math.cos(a[1]*Math.PI/180),x=(b[0]-a[0])*cos,y=b[1]-a[1],len=Math.hypot(x,y);
   if(!len)return;
   const dx=-y/len*1.5/(111320*cos),dy=x/len*1.5/111320;
   features.push({type:'Feature',properties:f.properties,geometry:{type:'Polygon',coordinates:[[[a[0]+dx,a[1]+dy],[b[0]+dx,b[1]+dy],[b[0]-dx,b[1]-dy],[a[0]-dx,a[1]-dy],[a[0]+dx,a[1]+dy]]]}});
  });
 }
 return {type:'FeatureCollection',features};
}
export function buildColumns(geometry:Geometry, calculation:Calculation|undefined,index:number,patternId?:string,roofHeight:(point:Point)=>number=()=>MIN_ROOF_HEIGHT) {
 const features:FeatureCollection<Polygon>['features']=[],pins:ColumnPin[]=[];let simplified=0;
 if(!calculation)return {data:{type:'FeatureCollection',features} as FeatureCollection<Polygon>,pins,simplified};
 const route=calculation.frames[index]?.routes.find(r=>r.route_id===geometry.route_id),policy=calculation.visualization;
 for(const stop of geometry.features.filter((f):f is StopFeature=>f.geometry.type==='Point')){
  if(patternId&&stop.properties.route_pattern_id!==patternId)continue;
  const reading=route?.stop_readings.find(r=>r.route_stop_id===stop.properties.route_stop_id&&r.route_pattern_id===stop.properties.route_pattern_id&&r.route_id===stop.properties.route_id);
  if(!reading||reading.evaluated.load_index.status==='unavailable')continue;
  const line=geometry.features.find(f=>f.properties.kind==='route_line'&&f.properties.route_pattern_id===stop.properties.route_pattern_id);
  const points=line?.geometry.type==='LineString'?surroundingAnchors(line.geometry.coordinates,stop.geometry.coordinates,Math.max(120,policy.stop_columns.approach_length_m*4/3)):null;
  if(!points)simplified++;
  const anchors=points??[stop.geometry.coordinates],value=reading.evaluated.load_index.value;
  const baseline=Math.max(MIN_ROOF_HEIGHT,roofHeight(stop.geometry.coordinates))+5;
  const height=0.4*(baseline+Math.max(3,Math.min(1,(value-policy.scale_min)/(policy.scale_max-policy.scale_min||1))*300));
  const center=anchors.reduce((best,p,i)=>meters(p,stop.geometry.coordinates)<meters(anchors[best],stop.geometry.coordinates)?i:best,0);
   const color=loadColor(value,policy.thresholds);
  anchors.forEach((p,i)=>{
   const h=i===center?height:height*Math.pow(0.65,Math.abs(i-center));
   features.push({type:'Feature',geometry:{type:'Polygon',coordinates:[square(p,i===center?2.4:2)]},properties:{...stop.properties,height:h,value,central:i===center,overflow:value>policy.scale_max,color}});
   if(i===center)pins.push({coordinates:p,height:h,value,color});
  });
 }
 return {data:{type:'FeatureCollection',features} as FeatureCollection<Polygon>,pins,simplified};
}
