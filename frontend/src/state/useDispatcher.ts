import { useCallback, useEffect, useRef, useState } from 'react';
import { api, errorText } from '../api/client';
import type { Bootstrap, Calculation, Geometry, Overrides, Schema, Selection } from '../api/types';
import { LatestRequest, assertCalculation } from './calculation';
import { nextDaySelection, overridesForDay } from '../features/time/serviceDay';
import { validateSelection } from '../features/time/dates';
type Confirmed={calculation:Calculation;nextDay?:Calculation;geometry:Geometry;route:Schema['RouteDetail']};
export function useDispatcher() {
 const [bootstrap,setBootstrap]=useState<Bootstrap>(),[confirmed,setConfirmed]=useState<Confirmed>(),[pending,setPending]=useState(false),[dirty,setDirty]=useState(false),[error,setError]=useState('');
 const requests=useRef(new LatestRequest()),bootstrapRequest=useRef(new LatestRequest()),attempt=useRef<{selection:Selection;overrides?:Overrides}|undefined>(undefined);
 const confirmedRef=useRef<Confirmed|undefined>(undefined);confirmedRef.current=confirmed;
 const calculate=useCallback(async(selection:Selection,overrides?:Overrides,b=bootstrap)=>{
  if(!b)return;
  const request=requests.current.next();attempt.current={selection,overrides};setPending(true);setDirty(true);setError('');
  try{
   validateSelection(selection,b);
   const routeId=selection.route_ids[0],version=b.active_snapshot.provenance.network_version,prev=confirmedRef.current;
   const sameRoute=prev?.geometry.route_id===routeId&&prev.geometry.network_version===version;
   const [route,geometry]=sameRoute?[prev.route,prev.geometry]:await Promise.all([api.route(routeId,version,request.signal),api.geometry(routeId,version,request.signal)]);
   validateSelection(selection,b,geometry.features.filter(f=>f.properties.kind==='stop').length);
   if(route.network_version!==version||geometry.network_version!==version||geometry.route_id!==routeId)throw new Error('Получена геометрия другой версии или маршрута.');
   const nextSelection=nextDaySelection(selection);
   if(nextSelection)validateSelection(nextSelection,b,geometry.features.filter(f=>f.properties.kind==='stop').length);
   if(overrides&&!overridesForDay(selection,overrides)&&!(nextSelection&&overridesForDay(nextSelection,overrides)))throw new Error('Изменения выпуска вне выбранного периода.');
   const query=async(s:Selection)=>{
    const o=overridesForDay(s,overrides);
    const result=await api.calculate(s,o,request.signal);
    assertCalculation(result,o?{kind:'scenario',selection:s,overrides:o}:{kind:'forecast',selection:s},geometry);
    return result;
   };
   const [calculation,nextDay]=await Promise.all([query(selection),nextSelection?query(nextSelection):Promise.resolve(undefined)]);
   if(nextDay&&nextDay.provenance.network_version!==calculation.provenance.network_version)throw new Error('Дни получены из разных версий геометрии.');
   if(request.isCurrent()){setConfirmed({calculation,nextDay,geometry,route});setDirty(false);}
  }catch(e){if(request.isCurrent())setError(errorText(e));}finally{if(request.isCurrent())setPending(false);}
 },[bootstrap]);
 const initialize=useCallback(async()=>{requests.current.cancel();const r=bootstrapRequest.current.next();setPending(true);setDirty(true);setError('');try{const b=await api.bootstrap(r.signal);if(!r.isCurrent())return;setBootstrap(b);const selection={...b.default_selection,route_ids:[b.default_selection.route_ids[0]],spatial_detail:b.capabilities.stop_forecasts&&b.capabilities.forecast_scopes.includes('route_stop')?'route_stop':'route'} as Selection;await calculate(selection,undefined,b);}catch(e){if(r.isCurrent()){setError(errorText(e));setPending(false);}}},[calculate]);
 useEffect(()=>{void initialize();return()=>{requests.current.cancel();bootstrapRequest.current.cancel();};},[]); // initialization only; all later transitions explicit
 return {bootstrap,confirmed,pending,pendingSelection:pending?attempt.current?.selection:undefined,dirty,error,calculate,initialize,markDirty:()=>{requests.current.cancel();setPending(false);setDirty(true);},retry:()=>attempt.current?calculate(attempt.current.selection,attempt.current.overrides):initialize(),discard:()=>{requests.current.cancel();setPending(false);setDirty(false);setError('');}};
}
