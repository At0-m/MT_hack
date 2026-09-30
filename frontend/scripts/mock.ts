// Isolated development HTTP fixture service. Never imported by the application bundle.
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { parse } from 'yaml';
import { DateTime } from 'luxon';
import type { FeatureCollection, Point, LineString } from 'geojson';
import type { IncomingMessage, ServerResponse } from 'node:http';
import type { Bootstrap, Calculation, Descriptor, Geometry, Reading, Schema, Selection, Overrides } from '../src/api/types';
import { frameWindows, moscow, validateSelection } from '../src/features/time/dates';
const spec = parse(readFileSync(new URL('../openapi/openapi.yaml',import.meta.url),'utf8'));
const seed = spec.components.examples.ForecastResponse.value as Calculation;
const originalGeometry = spec.components.examples.Geometry.value as Geometry;
// Replace this fixture with Point stops and a LineString following the actual road.
const routeFixture = JSON.parse(readFileSync(new URL('../public/mock/route-13.geojson',import.meta.url),'utf8')) as FeatureCollection<Point|LineString>;
const stopTemplate=originalGeometry.features.find((f):f is Schema['StopPointFeature']=>f.geometry.type==='Point')!;
const lineTemplate=originalGeometry.features.find((f):f is Schema['RouteLineFeature']=>f.geometry.type==='LineString')!;
const geometrySeed:Geometry={...originalGeometry,features:routeFixture.features.map((f,i)=>f.geometry.type==='Point'?{
 ...stopTemplate,id:String(f.id??f.properties?.stop_id??i),geometry:f.geometry,
 properties:{...stopTemplate.properties,stop_id:String(f.properties?.stop_id??f.id??i),name:String(f.properties?.name??'Остановка'),sequence:Number(f.properties?.sequence_sample??i+1)}
}:{...lineTemplate,id:String(f.id??'route-13'),geometry:f.geometry})};
const available = (value: number): Schema['NumericReading'] => ({status:'available',value});
const routes = ['13','1','3','25'].map((n,i)=>({route_id:`demo-${i+1}`,route_number:n,name:`Демонстрационный маршрут ${n}`,geometry_status:'approximate' as const}));
const coverageWindow = {from:'2024-01-01T00:00:00+03:00',to:'2028-01-01T00:00:00+03:00'};
export const bootstrap: Bootstrap = {
 api_version:'1.2.0',display_timezone:'Europe/Moscow',locale:'ru-RU',routes,
 active_snapshot:{provenance:{...seed.provenance,forecast_snapshot_id:'synthetic-ui-v1'},created_at:'2026-09-01T00:00:00+03:00',coverage:routes.map(r=>({route_id:r.route_id,window:coverageWindow,resolutions:['hour','day'],max_lead_hours:744,status:'experimental',prediction_scopes:['route','route_stop'],stop_forecast_status:'experimental'})),view_modes:['day','week','month'],reference_quantile:0.75,target_load_index:0.75,retention_hours_after_deactivation:24,limitations:[{code:'SYNTHETIC_MOCK',severity:'warning',message:'Демонстрационные данные и геометрия. Не результат прогнозной модели.'}]},
 default_selection:{forecast_snapshot_id:'synthetic-ui-v1',route_ids:['demo-1'],view_mode:'day',resolution:'hour',spatial_detail:'route_stop',window:{from:'2026-09-27T00:00:00+03:00',to:'2026-09-28T00:00:00+03:00'}},
 map:{renderer:'maplibre',style_url:'https://tiles.openfreemap.org/styles/bright',fallback_style_path:'/map/fallback.json',attribution:'© OpenStreetMap contributors · OpenFreeMap',center:{longitude:37.7515,latitude:55.8212},zoom:15.4},
 limits:{max_routes:10,max_window_hours:8784,max_hourly_cells:7440,max_body_bytes:65536,max_export_rows:50000,max_export_bytes:10000000,forecast_timeout_ms:15000,max_concurrent_inference:2,max_response_cells:960,max_custom_days:180,max_stop_cells:10000},
 capabilities:{forecast_scopes:['route','route_stop'],scenario_fleet:true,scenario_factors:['weather','event','season'],physical_weather_overrides:false,csv_export:true,live_ingestion:false,summary:true,custom_period:true,session_auth:true,stop_forecasts:true,segment_forecasts:false,decorative_approach_effect:true}
};
export function geometryFor(routeId: string): Geometry {
 const routeIndex=routes.findIndex(r=>r.route_id===routeId);
 return {...geometrySeed,route_id:routeId,features:geometrySeed.features.map(f=>{
  const properties = {...f.properties,route_id:routeId,route_pattern_id:routeId+'-out'};
  if (f.geometry.type === 'Point' && 'route_stop_id' in properties) return {...f,id:routeId+'-stop-'+properties.sequence,properties:{...properties,route_stop_id:routeId+'-stop-'+properties.sequence,name:properties.name},geometry:{...f.geometry,coordinates:[f.geometry.coordinates[0],f.geometry.coordinates[1]+routeIndex*0.003]}} as Schema['StopPointFeature'];
  return {...f,id:routeId+'-out',properties,geometry:{type:'LineString',coordinates:(f.geometry as Schema['LineString']).coordinates.map(p=>[p[0],p[1]+routeIndex*0.003])}} as Schema['RouteLineFeature'];
 })};
}
const metric = (boardings: number, fleet: number|null, hours: number, reference = 65, source: Schema['MetricSet']['fleet_source']='observed_vehicle_profile'): Schema['MetricSet'] => {
 const index=fleet===null||fleet===0?null:boardings/(fleet*hours*reference);
 const unavailable: Schema['NumericReading']={status:'unavailable',value:null,reason:fleet===null?'SUPPLY_UNKNOWN':'NO_SUPPLY'};
 return {boardings,vehicle_hours:fleet===null?unavailable:available(fleet*hours),mean_vehicle_count:fleet===null?unavailable:available(fleet),boardings_per_vehicle_hour:index===null?unavailable:available(boardings/(fleet!*hours)),reference_boardings:available(reference),load_index:index===null?unavailable:available(index),load_level:index===null?'unavailable':index<0.75?'normal':index<1?'elevated':index<1.25?'high':'very_high',required_vehicle_count:{status:'available',value:Math.ceil(boardings/(hours*reference*0.75))},fleet_source:fleet===null?'unavailable':source,fleet_is_proxy:true};
};
function delta(a: Schema['MetricSet'], b: Schema['MetricSet']): Schema['MetricDelta'] {return {boardings:b.boardings-a.boardings,load_index:a.load_index.value===null||b.load_index.value===null?null:b.load_index.value-a.load_index.value,vehicle_hours:a.vehicle_hours.value===null||b.vehicle_hours.value===null?null:b.vehicle_hours.value-a.vehicle_hours.value};}
function aggregateMetrics(items: Schema['MetricSet'][], hours:number): Schema['MetricSet'] {
 const fleet=items.every(m=>m.vehicle_hours.status==='available')?items.reduce((s,m)=>s+m.vehicle_hours.value!,0)/hours:null;
 return metric(items.reduce((s,m)=>s+m.boardings,0),fleet,hours,items[0].reference_boardings.value??65,items.every(m=>m.fleet_source===items[0].fleet_source)?items[0].fleet_source:'mixed');
}
function aggregate(readings: Reading[], hours:number): Reading {
 const baseline=aggregateMetrics(readings.map(r=>r.baseline),hours),evaluated=aggregateMetrics(readings.map(r=>r.evaluated),hours);
 const typical=readings.reduce((s,r)=>s+(r.typical_boardings.value??0),0);
 const common={...readings[0],baseline,evaluated,delta:delta(baseline,evaluated),typical_boardings:available(typical),relative_to_typical:typical?evaluated.boardings/typical-1:null};
 if ('stop_readings' in common) { const rr=readings as Schema['RouteReading'][]; return {...common,stop_readings:rr[0].stop_readings.map(s=>aggregate(rr.map(r=>r.stop_readings.find(x=>x.route_stop_id===s.route_stop_id)!),hours) as Schema['StopReading'])}; }
 return common;
}
export async function calculate(selection:Selection, overrides?: Overrides, fault=''): Promise<Calculation> {
 const boot=structuredClone(bootstrap);
 if(fault==='no-stops') {boot.capabilities.stop_forecasts=false;}
 validateSelection(selection,boot,Math.max(...selection.route_ids.map(id=>geometryFor(id).features.filter(f=>f.geometry.type==='Point').length)));
 if(selection.forecast_snapshot_id!==boot.default_selection.forecast_snapshot_id) throw new Error('Snapshot недоступен.');
 if(overrides){const w=overrides.effective_window;if(moscow(w.from)<moscow(selection.window.from)||moscow(w.to)>moscow(selection.window.to)||moscow(w.to)<=moscow(w.from))throw new Error('Область сценария вне периода.');}
 const descriptor:Descriptor=overrides?{kind:'scenario',selection,overrides}:{kind:'forecast',selection};
 const frames=frameWindows(selection).map(w=>{
  const start=moscow(w.from),hours=moscow(w.to).diff(start,'hours').hours;
  const routeReadings=selection.route_ids.map(routeId=>{
   const ri=routes.findIndex(r=>r.route_id===routeId);
   const hourly:Schema['RouteReading'][]=[];
   for(let h=0;h<hours;h++){
    const time=start.plus({hours:h});const hour=time.hour;
    const baseFleet=fault==='unknown-fleet'?null:10+ri;
    const demand=(210+((hour*37+time.ordinal*11+ri*53)%660));
    const affected=overrides&&time>=moscow(overrides.effective_window.from)&&time<moscow(overrides.effective_window.to);
    const fleetRule=affected?overrides.fleet:undefined;
    let fleet=baseFleet;
    if(fleetRule){if(fleetRule.kind==='delta'&&baseFleet===null)throw new Error('Delta недоступна при неизвестном baseline.');fleet=fleetRule.kind==='absolute'?fleetRule.vehicle_count:baseFleet!+fleetRule.vehicle_count_delta;if(!Number.isInteger(fleet)||fleet<0||fleet>200)throw new Error('Выпуск должен быть целым числом от 0 до 200.');}
    const factor=affected?Object.values(overrides.factors??{}).reduce((a,b)=>a*b,1):1;
    const baseline=metric(demand,baseFleet,1),evaluated=metric(demand*factor,fleet,1,65,fleetRule?fleetRule.kind==='absolute'?'scenario_absolute':'scenario_delta':undefined);
    const stop_readings=selection.spatial_detail==='route_stop'?geometryFor(routeId).features.filter((f):f is Schema['StopPointFeature']=>f.geometry.type==='Point').map((f,i)=>{
     const stopDemand=35+((hour*19+i*83+time.ordinal*7)%160);const reference=18+i*9;
     const a=metric(stopDemand,baseFleet,1,reference),e=metric(stopDemand*factor,fleet,1,reference,fleetRule?fleetRule.kind==='absolute'?'scenario_absolute':'scenario_delta':undefined);
     return {...f.properties,prediction_method:'synthetic_mock' as const,typical_boardings:available(110+i*15),relative_to_typical:stopDemand*factor/(110+i*15)-1,baseline:a,evaluated:e,delta:delta(a,e)};
    }).map(({kind:_kind,name:_name,...r})=>r):[];
    hourly.push({route_id:routeId,baseline,evaluated,delta:delta(baseline,evaluated),typical_boardings:available(550),relative_to_typical:demand*factor/550-1,weather:{mode:'forecast',temperature_mean_c:9.8,precipitation_sum_mm:0.4,forecast_hours:1,climatology_hours:0,missing_hours:0},stop_readings,indicators:[{key:'weather',variant:'rain',tone:'positive',icon:'weather',title:'Дождь',text:'Демонстрационный погодный индикатор.'},{key:'trend',variant:'down',tone:'warning',icon:'trend',title:'Снижение',text:'Демонстрационный индикатор динамики.'},{key:'fleet',variant:'deficit',tone:'critical',icon:'fleet',title:'Выпуск',text:'Демонстрационный статус выпуска.'}]});
   }
   return aggregate(hourly,hours) as Schema['RouteReading'];
  });return {window:w,routes:routeReadings};
 });
 const hours=moscow(selection.window.to).diff(moscow(selection.window.from),'hours').hours;
 return {...structuredClone(seed),descriptor,calculation_id:'calc_'+createHash('sha256').update(JSON.stringify(descriptor)).digest('hex'),provenance:boot.active_snapshot.provenance,frames,totals:selection.route_ids.map(id=>aggregate(frames.map(f=>f.routes.find(r=>r.route_id===id)!),hours) as Schema['RouteReading']),notices:boot.active_snapshot.limitations};
}
export function mockMiddleware() {
 const sessions=new Map<string,number>();
 return async (req:IncomingMessage,res:ServerResponse,next:()=>void)=>{
  if(!req.url?.startsWith('/api/v1/'))return next();
  const url=new URL(req.url,'http://localhost');const path=url.pathname.replace('/api/v1','');
  const fault=(req.headers.cookie??'').match(/(?:^|; )mt_mock_fault=([^;]+)/)?.[1]??'';
  const sid=(req.headers.cookie??'').match(/(?:^|; )mt_mock_session=([^;]+)/)?.[1];
  const send=(status:number,data?:unknown)=>{res.statusCode=status;res.setHeader('Cache-Control','no-store');res.setHeader('Content-Type','application/json');res.end(data===undefined?undefined:JSON.stringify(data));};
  const problem=(status:number,code:string,detail:string)=>send(status,{type:'about:blank',title:detail,status,code,detail,request_id:'mock-request'});
  const session=()=>({authenticated:true,user:{user_id:'demo-user',username:'demo',display_name:'Демонстрационный диспетчер'},expires_at:DateTime.now().plus({hours:1}).toISO()});
  try {
   let body: Record<string,unknown>={};if(req.method==='POST'){let raw='';for await(const chunk of req){raw+=chunk;if(raw.length>65536)return problem(413,'BODY_TOO_LARGE','Слишком большой запрос.');}body=raw?JSON.parse(raw):{};}
   if(path==='/auth/login'){if(body.username!=='demo'||body.password!=='demo')return problem(401,'INVALID_CREDENTIALS','Неверный демонстрационный логин или пароль.');const id=createHash('sha256').update(String(Date.now())).digest('hex');sessions.set(id,Date.now()+3600000);res.setHeader('Set-Cookie',`mt_mock_session=${id}; Path=/; HttpOnly; SameSite=Strict`);return send(200,session());}
   if(!sid||!sessions.has(sid)||sessions.get(sid)!<Date.now()||fault==='expired')return problem(401,'SESSION_EXPIRED','Сессия истекла.');
   if(path==='/auth/session')return send(200,session());
   if(path==='/auth/logout'){sessions.delete(sid);res.setHeader('Set-Cookie','mt_mock_session=; Path=/; Max-Age=0; HttpOnly; SameSite=Strict');return send(204);}
   await new Promise(resolve=>setTimeout(resolve,fault==='slow'?1600:160));
   if(fault==='error')return problem(503,'SERVICE_UNAVAILABLE','Демонстрация ошибки сервиса.');
   if(path==='/bootstrap'){const b=structuredClone(bootstrap);if(fault==='no-stops'){b.capabilities.stop_forecasts=false;b.capabilities.forecast_scopes=['route'];b.default_selection.spatial_detail='route';}return send(200,b);}
   const route=path.match(/^\/routes\/([^/]+)(\/geometry)?$/);
   if(route){const g=geometryFor(route[1]);if(route[2])return send(200,g);const stops=g.features.filter((f):f is Schema['StopPointFeature']=>f.geometry.type==='Point');return send(200,{route:routes.find(r=>r.route_id===route[1]),network_version:g.network_version,patterns:[{route_pattern_id:route[1]+'-out',name:'Прямое направление',stops:stops.map(f=>({route_stop_id:f.properties.route_stop_id,stop_id:f.properties.stop_id,sequence:f.properties.sequence,name:f.properties.name,position:{longitude:f.geometry.coordinates[0],latitude:f.geometry.coordinates[1]}}))}],source_id:g.source_id,geometry_observed_at:'2026-09-01T00:00:00+03:00',historical_match_status:'unverified',notices:[]});}
   if(path==='/forecasts/query'||path==='/scenarios/evaluate'){const c=await calculate(body.selection as Selection,body.overrides as Overrides|undefined,fault);if(fault==='partial')c.frames.pop();return send(200,c);}
   if(path==='/summaries/query'||path==='/exports'){
    if(fault==='mismatch')return problem(409,'CALCULATION_MISMATCH','Версия расчёта не совпадает.');
    if(fault==='gone')return problem(410,'SNAPSHOT_EXPIRED','Snapshot недоступен.');
    const d=body.calculation as Descriptor;const c=await calculate(d.selection,d.kind==='scenario'?d.overrides:undefined,fault);
    if(c.calculation_id!==body.expected_calculation_id)return problem(409,'CALCULATION_MISMATCH','Версия расчёта не совпадает.');
    if(path==='/summaries/query'){const common={calculation_id:c.calculation_id,focus:body.focus,window:d.selection.window};return send(200,fault==='summary-unavailable'?{...common,status:'unavailable',reason:'NO_TEXT_AVAILABLE'}:{...common,status:'ready',text:'Демонстрационная сводка за выбранный период. Данные синтетические, выводы модели отсутствуют.',facts:[{key:'boardings',text:`В расчёте ${c.frames.length} временных кадров.`}]});}
    const q=(x:unknown)=>`"${String(x??'').replaceAll('"','""')}"`;
    const rows=[['from','to','route_id','route_pattern_id','route_stop_id','boardings','load_index','load_status']];
    for(const f of c.frames)for(const r of f.routes)for(const value of [r,...r.stop_readings])rows.push([f.window.from,f.window.to,r.route_id,'route_pattern_id' in value?value.route_pattern_id:'','route_stop_id' in value?value.route_stop_id:'',String(value.evaluated.boardings),String(value.evaluated.load_index.value??''),value.evaluated.load_index.status]);
    res.setHeader('Content-Type','text/csv; charset=utf-8');res.setHeader('Content-Disposition','attachment; filename="transport.csv"');res.end('\uFEFF'+rows.map(r=>r.map(q).join(',')).join('\r\n'));return;
   }
   problem(404,'NOT_FOUND','Операция отсутствует.');
  }catch(e){problem(422,'INVALID_SELECTION',e instanceof Error?e.message:'Некорректный запрос.');}
 };
}
