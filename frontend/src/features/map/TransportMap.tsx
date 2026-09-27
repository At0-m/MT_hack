import { useEffect, useRef, useState } from 'react';
import * as maplibregl from 'maplibre-gl';
import workerUrl from 'maplibre-gl/dist/maplibre-gl-worker.mjs?worker&url';
maplibregl.setWorkerUrl(workerUrl);
import type { GeoJSONSource, StyleSpecification } from 'maplibre-gl';
import type { FeatureCollection } from 'geojson';
import type { Bootstrap, Calculation, Geometry, StopFeature, StopFocus } from '../../api/types';
import { buildColumns, buildRouteRibbon, COLUMN_MIN_ZOOM, MIN_ROOF_HEIGHT, loadColor, isTopDown } from './columns';
import { columnPins } from './columnPins';
import 'maplibre-gl/dist/maplibre-gl.css';
const BUILDING_MIN_ZOOM=13;
const BUILDING_DETAIL_DISTANCE=2.5;
const APPROACH_NOTICE='Для части остановок подход неоднозначен: показан один столбик у остановки.';
const localFallback:StyleSpecification={version:8,sources:{},layers:[{id:'background',type:'background',paint:{'background-color':'#041a38'}}]};
export function TransportMap(props:{config:Bootstrap['map'];geometry?:Geometry;calculation?:Calculation;index:number;patternId?:string;selected?:StopFocus;onSelect:(s:StopFocus)=>void}) {
 const container=useRef<HTMLDivElement>(null),mapRef=useRef<maplibregl.Map|null>(null),latest=useRef(props);
 const [notice,setNotice]=useState(''),[noticeVisible,setNoticeVisible]=useState(true);
 useEffect(()=>{setNoticeVisible(true);if(notice!==APPROACH_NOTICE)return;const timeout=setTimeout(()=>setNoticeVisible(false),6000);return()=>clearTimeout(timeout);},[notice,props.geometry]);
 latest.current=props;
 const update=useRef<()=>void>(()=>{});
 useEffect(()=>{
  const ctrl=new AbortController();let alive=true,fallback=false;
  const markers:maplibregl.Marker[]=[];let markerGeometry:Geometry|undefined;let markerPattern:string|undefined;
  let map:maplibregl.Map;
  try {map=new maplibregl.Map({container:container.current!,style:props.config.style_url,center:[props.config.center.longitude,props.config.center.latitude],zoom:props.config.zoom,pitch:55,bearing:-22,attributionControl:{compact:true,customAttribution:props.config.attribution},maxPitch:75});mapRef.current=map;}
  catch {setNotice('WebGL недоступен. Карта не может быть показана.');return;}
  const fallbackStyle=async()=>{if(fallback||!alive)return;fallback=true;setNotice('Подложка недоступна. Отображается загруженная геометрия.');try{const r=await fetch(props.config.fallback_style_path,{signal:ctrl.signal});if(!r.ok)throw new Error();const style=await r.json();if(alive)map.setStyle(style);}catch{if(alive)map.setStyle(localFallback);}};
  const pins=columnPins(map,container.current!);const roofs=new Map<string,number>();
  const zoomMarkers=()=>{
   const zoom=map.getZoom(),topDown=isTopDown(map.getPitch()),size=0.85*1.2*Math.max(32,Math.min(92,46+(zoom-14)*12));
   container.current?.classList.toggle('map-top-down',topDown);
   if(map.getLayer('stop-columns')){
    const visibility=topDown?'none':'visible';
    if(map.getLayoutProperty('stop-columns','visibility')!==visibility)map.setLayoutProperty('stop-columns','visibility',visibility);
   }
   for(const marker of markers){
    const el=marker.getElement();el.style.setProperty('--marker-size',`${size}px`);
     el.classList.toggle('top-down',topDown&&zoom>=COLUMN_MIN_ZOOM);
     el.classList.toggle('distant',zoom<BUILDING_MIN_ZOOM);
      el.classList.toggle('far-zoom',topDown||zoom<COLUMN_MIN_ZOOM);
     el.classList.toggle('load-colored',zoom>=BUILDING_MIN_ZOOM&&zoom<COLUMN_MIN_ZOOM);
   }
  };
  const selectMarkers=()=>{for(const marker of markers){const el=marker.getElement(), selected=latest.current.selected;const active=el.dataset.stopId===selected?.route_stop_id&&el.dataset.patternId===selected?.route_pattern_id;el.classList.toggle('selected',active);el.setAttribute('aria-pressed',String(active));}};
  const sync=()=>{
   selectMarkers();
   if(!map.isStyleLoaded()){map.once('idle',sync);return;}
   const {geometry,calculation,index,patternId}=latest.current;
   const lines:FeatureCollection={type:'FeatureCollection',features:(geometry?.features??[]).filter(f=>f.properties.kind==='route_line'&&(!patternId||f.properties.route_pattern_id===patternId)) as FeatureCollection['features']};
   if(!map.getSource('routes'))map.addSource('routes',{type:'geojson',data:lines});else(map.getSource('routes') as GeoJSONSource).setData(lines);
   const ribbon=geometry?buildRouteRibbon(geometry,patternId):{type:'FeatureCollection' as const,features:[]};
   if(!map.getSource('route-ribbon'))map.addSource('route-ribbon',{type:'geojson',data:ribbon});else(map.getSource('route-ribbon') as GeoJSONSource).setData(ribbon);
   if(!map.getLayer('route-raised'))map.addLayer({id:'route-raised',type:'fill-extrusion',source:'route-ribbon',paint:{'fill-extrusion-height':1.4,'fill-extrusion-base':0,'fill-extrusion-color':'#7ce0ff','fill-extrusion-opacity':0.95}});
   const columns=geometry?buildColumns(geometry,calculation,index,patternId,point=>roofs.get(point.join(','))??MIN_ROOF_HEIGHT):{data:{type:'FeatureCollection' as const,features:[]},pins:[],simplified:0};
   if(!map.getSource('columns'))map.addSource('columns',{type:'geojson',data:columns.data});else(map.getSource('columns') as GeoJSONSource).setData(columns.data);
   if(!map.getLayer('stop-columns'))map.addLayer({id:'stop-columns',type:'fill-extrusion',source:'columns',minzoom:COLUMN_MIN_ZOOM,paint:{'fill-extrusion-height':['get','height'],'fill-extrusion-color':['get','color'],'fill-extrusion-opacity':0.98,'fill-extrusion-vertical-gradient':false}});
   pins.update(columns.pins);
   if(!map.getLayer(pins.layer.id))map.addLayer(pins.layer);
   if(!fallback)setNotice(columns.simplified?APPROACH_NOTICE:'');
   if(markerGeometry!==geometry||markerPattern!==patternId){markers.splice(0).forEach(m=>m.remove());markerGeometry=geometry;markerPattern=patternId;
    const stops=(geometry?.features??[]).filter((f):f is StopFeature=>f.geometry.type==='Point'&&(!patternId||f.properties.route_pattern_id===patternId));
    for(const stop of stops){const button=document.createElement('button');button.className='stop-marker';button.type='button';button.setAttribute('aria-label',`Остановка ${stop.properties.name}`);button.dataset.stopId=stop.properties.route_stop_id;button.dataset.patternId=stop.properties.route_pattern_id;
     const img=document.createElement('img');img.className='stop-tram-icon';img.src='/assets/icons/incoming/train_white.png';img.alt='';const label=document.createElement('span');label.className='stop-label';label.textContent=stop.properties.name;const dot=document.createElement('img');dot.className='stop-orb';dot.src='/assets/icons/8584d.svg';dot.alt='';const colorDot=document.createElement('span');colorDot.className='stop-load-orb';const indexLabel=document.createElement('span');indexLabel.className='stop-top-index';button.append(img,label,dot,colorDot,indexLabel);
     button.addEventListener('click',()=>latest.current.onSelect({kind:'route_stop',route_id:stop.properties.route_id,route_pattern_id:stop.properties.route_pattern_id,route_stop_id:stop.properties.route_stop_id}));
     markers.push(new maplibregl.Marker({element:button,pitchAlignment:'viewport',rotationAlignment:'viewport'}).setLngLat(stop.geometry.coordinates as [number,number]).addTo(map));
    }
    if(stops.length){const bounds=new maplibregl.LngLatBounds();stops.forEach(s=>bounds.extend(s.geometry.coordinates as [number,number]));map.fitBounds(bounds,{padding:{top:180,bottom:120,left:180,right:180},maxZoom:16,duration:0});}
   }
    for(const marker of markers){const el=marker.getElement();const reading=calculation?.frames[index]?.routes.find(r=>r.route_id===geometry?.route_id)?.stop_readings.find(r=>r.route_stop_id===el.dataset.stopId&&r.route_pattern_id===el.dataset.patternId);el.style.setProperty('--stop-color',reading?.evaluated.load_index.status==='available'?loadColor(reading.evaluated.load_index.value,calculation?.visualization.thresholds):loadColor(undefined));const label=el.querySelector('.stop-top-index');if(label)label.textContent=reading?.evaluated.load_index.status==='available'?`${Math.round(reading.evaluated.load_index.value*100)}%`:'Н/Д';}
   selectMarkers();
   zoomMarkers();
  };update.current=sync;
  map.on('style.load',()=>{clearTimeout(timer);
   // Recolor existing vector layers; do not assume buildings exist in a raster basemap.
   const style=map.getStyle();for(const layer of style.layers){try{if(layer.type==='background')map.setPaintProperty(layer.id,'background-color','#041a38');else if(layer.type==='fill')map.setPaintProperty(layer.id,'fill-color',/water/.test(layer.id)?'#041a38':/building/.test(layer.id)?'#39495a':'#0b2544');else if(layer.type==='line')map.setPaintProperty(layer.id,'line-color',/water/.test(layer.id)?'#193b61':'#294566');else if(layer.type==='symbol')map.setLayoutProperty(layer.id,'visibility','none');}catch{/* A provider may expose unsupported expressions. */}}
   const building=style.layers.find(l=>'source-layer' in l&&l['source-layer']==='building');
   // Keep higher-detail building tiles farther toward the horizon. Tile budgets
   // scale with area; the distance target is approximate and depends on provider data.
   if(building&&'source' in building&&typeof map.setSourceTileLodParams==='function')map.setSourceTileLodParams(
    1+(9.314-1)/BUILDING_DETAIL_DISTANCE,
    3*BUILDING_DETAIL_DISTANCE**2,
    building.source as string
   );
   if(building&&'source' in building&&!map.getLayer('buildings-3d'))map.addLayer({id:'buildings-3d',type:'fill-extrusion',source:building.source as string,'source-layer':'building',minzoom:BUILDING_MIN_ZOOM,paint:{'fill-extrusion-color':'#39495a','fill-extrusion-height':['coalesce',['get','render_height'],['get','height'],0],'fill-extrusion-base':['coalesce',['get','render_min_height'],0],'fill-extrusion-opacity':0.65}});
   // Paint updates above can leave the style loading until its next idle event.
   map.once('idle',sync);});
  map.on('idle',()=>{
   if(!map.getLayer('buildings-3d')||map.getZoom()<COLUMN_MIN_ZOOM)return;
   let changed=false;
   for(const stop of latest.current.geometry?.features??[]){
    if(stop.geometry.type!=='Point')continue;
    const [lng,lat]=stop.geometry.coordinates,key=stop.geometry.coordinates.join(',');
    const dx=220/(111320*Math.cos(lat*Math.PI/180)),dy=220/111320;
    const corners=[[lng-dx,lat-dy],[lng-dx,lat+dy],[lng+dx,lat-dy],[lng+dx,lat+dy]].map(p=>map.project(p as [number,number]));
    const bounds:[maplibregl.PointLike,maplibregl.PointLike]=[[Math.min(...corners.map(p=>p.x)),Math.min(...corners.map(p=>p.y))],[Math.max(...corners.map(p=>p.x)),Math.max(...corners.map(p=>p.y))]];
    const previous=roofs.get(key)??MIN_ROOF_HEIGHT;
    const height=map.queryRenderedFeatures(bounds,{layers:['buildings-3d']}).reduce((max,f)=>Math.max(max,Number(f.properties.render_height??f.properties.height)||0),previous);
    if(height>previous){roofs.set(key,height);changed=true;}
   }
   if(changed)sync();
  });
  map.on('error',()=>{void fallbackStyle();});map.on('zoom',zoomMarkers);map.on('pitch',zoomMarkers);
  map.on('click','stop-columns',event=>{const p=event.features?.[0]?.properties;if(p)latest.current.onSelect({kind:'route_stop',route_id:p.route_id,route_pattern_id:p.route_pattern_id,route_stop_id:p.route_stop_id});});
  const timer=setTimeout(()=>{if(!map.isStyleLoaded())void fallbackStyle();},8000);
  const resize=new ResizeObserver(()=>map.resize());resize.observe(container.current!);
  return()=>{alive=false;ctrl.abort();clearTimeout(timer);resize.disconnect();markers.forEach(m=>m.remove());pins.remove();map.remove();mapRef.current=null;};
 },[props.config]);
 useEffect(()=>update.current(),[props.geometry,props.calculation,props.index,props.patternId,props.selected]);
 return <><div className="map" ref={container} aria-label="Интерактивная карта маршрута"/>{notice&&noticeVisible&&<div className="map-notice" role="status">{notice}<button aria-label="Закрыть сообщение карты" onClick={()=>setNoticeVisible(false)}>Закрыть</button></div>}</>;
}
