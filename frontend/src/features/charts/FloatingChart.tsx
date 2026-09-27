import { IncomingIcon } from '../../assets/IncomingIcon';
import { useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import type { CSSProperties, PointerEvent, ReactNode } from 'react';
import type { Calculation, Geometry, StopFeature, StopFocus, Reading } from '../../api/types';
import type { DisplayFrame } from '../time/serviceDay';
import { dateLabel } from '../time/dates';
import { selectReading } from '../../state/calculation';
import { metric } from '../../ui/common';
import { loadColor } from '../map/columns';

type Props={geometry:Geometry;calculation:Calculation;frameIndex:number;frames:DisplayFrame[];displayIndex:number;onFrame:(index:number)=>void;onClose:()=>void;ordinal:number};
const keyOf=(s:StopFeature)=>`${s.properties.route_pattern_id}:${s.properties.route_stop_id}`;
const focusOf=(s:StopFeature):StopFocus=>({kind:'route_stop',route_id:s.properties.route_id,route_pattern_id:s.properties.route_pattern_id,route_stop_id:s.properties.route_stop_id});
const compactIndex=(r?:Reading)=>r?.evaluated.load_index.status==='available'?metric(r.evaluated.load_index,true):'Н/Д';
const percent=(r?:Reading)=>r?.evaluated.load_index.status==='available'?Math.max(0,r.evaluated.load_index.value):undefined;
function IconButton({label,onClick,disabled=false}:{label:string;onClick:()=>void;disabled?:boolean}){
 return <button type="button" aria-label={label} title={label} onClick={onClick} disabled={disabled}><IncomingIcon name={label.includes("Закрыть")?"cross.png":label.includes("Свернуть")?"up.png":"down.png"} className={label.includes("Предыдущ")?"arrow-left":label.includes("Следующ")?"arrow-right":""}/></button>;
}
function Bars({items,selected,onSelect,ticks,ceiling,thresholds}:{ceiling:number;thresholds:readonly number[];items:{key:string;label:string;reading?:Reading}[];selected:number;onSelect:(i:number)=>void;ticks:{index:number;label:string;position?:number}[]}){
 const positionLabel=(button:HTMLButtonElement)=>{
  const label=button.querySelector<HTMLElement>('.floating-tooltip'),row=button.parentElement;
  if(!label||!row)return;
  label.style.maxWidth=`${row.clientWidth}px`;
  const desired=button.offsetLeft+button.offsetWidth/2-label.offsetWidth/2;
  label.style.left=`${Math.max(0,Math.min(row.clientWidth-label.offsetWidth,desired))-button.offsetLeft}px`;
 };
 return <div className="floating-plot" role="group" aria-label="График загруженности">
  <div className="floating-grid" aria-hidden="true">{[25,50,75,100].map(v=><span key={v} style={{bottom:`${v}%`}}><b>{Math.round(v*ceiling)}</b></span>)}</div>
  <div className="floating-bars">{items.map((item,i)=>{const value=percent(item.reading);return <button key={item.key} aria-label={`${item.label}: ${metric(item.reading?.evaluated.load_index,true)}`} aria-pressed={i===selected} className={i===selected?'selected':''} onMouseEnter={e=>positionLabel(e.currentTarget)} onFocus={e=>positionLabel(e.currentTarget)} onClick={()=>onSelect(i)}>
   <span className={value===undefined?'no-data':'floating-bar'} style={{height:value===undefined?'6px':`${Math.min(100,value/ceiling*100)}%`,background:i===selected&&value!==undefined?`linear-gradient(to top,#00ff00,${loadColor(value,thresholds)})`:undefined}}/>
   {i!==selected&&<span className="floating-tooltip">{item.label}</span>}
  </button>;})}</div>
   <div className="floating-ticks">{ticks.map(t=><span key={t.index} style={{left:`${t.position??t.index/Math.max(1,items.length-1)*100}%`}}>{t.label}</span>)}</div>
 </div>;
}
export function FloatingChart({geometry,calculation,frameIndex,frames,displayIndex,onFrame,onClose,ordinal}:Props){
 const windowRef=useRef<HTMLElement>(null);
 const [nameTip,setNameTip]=useState<{text:string;x:number;y:number;overlay:Element}>();
 const showName=(element:HTMLElement,text:string)=>{
  const overlay=element.closest('.ui-overlay');if(!overlay)return;
  const area=overlay.getBoundingClientRect(),rect=element.getBoundingClientRect(),scale=area.width/1920;
  setNameTip({text,overlay,x:Math.min(1570,Math.max(8,(rect.left-area.left)/scale)),y:Math.min(960,(rect.bottom-area.top)/scale+6)});
 };
 const [mode,setMode]=useState<'choose'|'route'|'pick'|'station'>('choose');
 const [collapsed,setCollapsed]=useState(false),[search,setSearch]=useState(''),[selectedKey,setSelectedKey]=useState<string>();
 const [routePage,setRoutePage]=useState(0),[closing,setClosing]=useState(false);
 const [position,setPosition]=useState({x:100+(ordinal%5)*42,y:180+(ordinal%5)*42});
 const [raised,setRaised]=useState(0);
 const drag=useRef<{x:number;y:number;px:number;py:number;scale:number}|undefined>(undefined);
 const stops=geometry.features.filter((s):s is StopFeature=>s.geometry.type==='Point').sort((a,b)=>a.properties.route_pattern_id.localeCompare(b.properties.route_pattern_id)||a.properties.sequence-b.properties.sequence);
 const selected=stops.find(s=>keyOf(s)===selectedKey)??(mode==='route'?stops[0]:undefined);
 const selectedReading=selected?selectReading(calculation,frameIndex,geometry.route_id,focusOf(selected)):undefined;
 const pageCount=Math.ceil(stops.length/20),page=Math.min(routePage,Math.max(0,pageCount-1));
 const pageStops=stops.slice(page*20,(page+1)*20);
 const wide=mode==='route',width=collapsed?(wide?420:350):(wide?700:mode==='station'?350:310),height=collapsed?46:wide?360:400;
 const x=Math.max(0,Math.min(position.x,1920-width)),y=Math.max(0,Math.min(position.y,1080-height));
 const title=mode==='choose'?'Выбрать график':mode==='route'?'График маршрута':mode==='pick'?'График станции':selected?.properties.name??'График станции';
 const startDrag=(event:PointerEvent<HTMLElement>)=>{
  if(event.button!==0||(event.target as HTMLElement).closest('button'))return;
  const overlay=event.currentTarget.closest('.ui-overlay')!;
  drag.current={x:event.clientX,y:event.clientY,px:x,py:y,scale:overlay.getBoundingClientRect().width/1920};
  event.currentTarget.setPointerCapture(event.pointerId);
 };
 const moveDrag=(event:PointerEvent<HTMLElement>)=>{const d=drag.current;if(d)setPosition({x:Math.max(0,Math.min(1920-width,d.px+(event.clientX-d.x)/d.scale)),y:Math.max(0,Math.min(1080-height,d.py+(event.clientY-d.y)/d.scale))});};
 const switchPage=(next:number)=>{setRoutePage(next);const stop=stops[next*20];if(stop)setSelectedKey(keyOf(stop));};
 const hourly=calculation.descriptor.selection.view_mode==='day';
 const stationPage=hourly?0:Math.floor(displayIndex/10);
 const stationOffset=stationPage*10;
 const stationFrames=hourly?frames:frames.slice(stationOffset,stationOffset+10);
 const stationPages=hourly?1:Math.ceil(frames.length/10);
 const stationTickPositions=hourly||calculation.descriptor.selection.view_mode==='week'?undefined:[0,.25,.5,.75,1];
 const tickIndices=Array.from(new Set((hourly?[0,.25,.5,.75,1].map(v=>Math.round((stationFrames.length-1)*v)):calculation.descriptor.selection.view_mode==='week'?[0,3,6]:stationTickPositions!.map(v=>Math.round((stationFrames.length-1)*v))).filter(i=>i>=0&&i<stationFrames.length)));

 let content:ReactNode;
 if(mode==='choose')content=<div className="floating-choices"><button onClick={()=>{setMode('route');setSelectedKey(undefined);}}>График<br/>маршрута</button><button onClick={()=>setMode('pick')}>График<br/>станции</button></div>;
 else if(mode==='pick')content=<><label className="floating-search"><input style={{width:`${Math.max(5,search.length)}ch`}} placeholder="Поиск" aria-label="Поиск станции" value={search} onChange={e=>setSearch(e.target.value)}/><IncomingIcon name="search.png"/></label><div className="floating-stop-list" onScroll={()=>setNameTip(undefined)}>{stops.filter(s=>s.properties.name.toLocaleLowerCase('ru').includes(search.toLocaleLowerCase('ru'))).map(s=><button key={keyOf(s)} onMouseEnter={e=>showName(e.currentTarget,s.properties.name)} onMouseLeave={()=>setNameTip(undefined)} onFocus={e=>showName(e.currentTarget,s.properties.name)} onBlur={()=>setNameTip(undefined)} onClick={()=>{setNameTip(undefined);setSelectedKey(keyOf(s));setMode('station');}}>{s.properties.sequence}. {s.properties.name}</button>)}{!stops.some(s=>s.properties.name.toLocaleLowerCase('ru').includes(search.toLocaleLowerCase('ru')))&&<p className="floating-empty-search">Пусто</p>}</div></>;
 else if(mode==='route')content=stops.length?<><Bars ceiling={calculation.visualization.scale_max} thresholds={calculation.visualization.thresholds} items={pageStops.map(s=>({key:keyOf(s),label:s.properties.name,reading:selectReading(calculation,frameIndex,geometry.route_id,focusOf(s))}))} selected={pageStops.findIndex(s=>keyOf(s)=== (selected&&keyOf(selected)))} onSelect={i=>setSelectedKey(keyOf(pageStops[i]))} ticks={[]}/>{pageCount>1&&<div className="floating-route-pages"><IconButton label="Предыдущие остановки" disabled={page===0} onClick={()=>switchPage(page-1)}/><IconButton label="Следующие остановки" disabled={page===pageCount-1} onClick={()=>switchPage(page+1)}/></div>}<div className="floating-route-caption"><IncomingIcon name="train.png"/><span title={selected?.properties.name}>{selected?.properties.name}</span><strong style={{color:percent(selectedReading)!==undefined?loadColor(percent(selectedReading)!):undefined}}>{compactIndex(selectedReading)}</strong></div></>:<p className="floating-empty">Нет остановок для этого маршрута</p>;
 else content=selected?<><Bars ceiling={calculation.visualization.scale_max} thresholds={calculation.visualization.thresholds} items={stationFrames.map(f=>({key:f.calculation.frames[f.index].window.from,label:dateLabel(f.calculation.frames[f.index].window.from,hourly?'HH:mm':'dd.MM'),reading:selectReading(f.calculation,f.index,geometry.route_id,focusOf(selected))}))} selected={displayIndex-stationOffset} onSelect={i=>onFrame(stationOffset+i)} ticks={tickIndices.map((i,position)=>({index:i,label:dateLabel(stationFrames[i].calculation.frames[stationFrames[i].index].window.from,hourly?'HH:mm':'dd.MM'),position:stationTickPositions ? stationTickPositions[position]*100 : undefined}))}/><div className={`floating-assessment ${stationPages>1?'with-pages':''}`}><strong style={{color:percent(selectedReading)!==undefined?loadColor(percent(selectedReading)!):undefined}}>{compactIndex(selectedReading)}</strong>{stationPages>1&&<div className="floating-station-pages" aria-label={`Страница ${stationPage+1} из ${stationPages}`}><IconButton label="Предыдущие 10 кадров" disabled={stationPage===0} onClick={()=>onFrame((stationPage-1)*10)}/></div>}<p>{selectedReading?.relative_to_typical!=null?<>на <span style={{color:percent(selectedReading)!==undefined?loadColor(percent(selectedReading)!):undefined}}>{Math.round(Math.abs(selectedReading.relative_to_typical)*100)}%</span> {selectedReading.relative_to_typical<0?'меньше':'больше'}<br/>чем обычно</>:'Сравнение недоступно'}</p>{stationPages>1&&<div className="floating-station-pages" aria-label={`Страница ${stationPage+1} из ${stationPages}`}><IconButton label="Следующие 10 кадров" disabled={stationPage+1>=stationPages} onClick={()=>onFrame((stationPage+1)*10)}/></div>}</div></>:<div className="floating-empty">Остановка недоступна на текущем маршруте.<button onClick={()=>setMode('pick')}>Выбрать станцию</button></div>;
 return <section ref={windowRef} role="dialog" aria-label={title} className={`floating-chart ${wide?'floating-wide':''} ${collapsed?'is-collapsed':''} ${closing?'is-closing':''}`} style={{left:x,top:y,width,height,zIndex:30+raised,'--expanded-width':`${wide?700:mode==='station'?350:310}px`} as CSSProperties} onPointerDown={()=>setRaised(Date.now()%1000000)} onAnimationEnd={e=>{if(e.target===e.currentTarget&&e.animationName==='floating-out')onClose();}}>
  <header className="floating-header" onPointerDown={startDrag} onPointerMove={moveDrag} onPointerUp={event=>{drag.current=undefined;if(event.currentTarget.hasPointerCapture(event.pointerId))event.currentTarget.releasePointerCapture(event.pointerId);}} onPointerCancel={()=>{drag.current=undefined;}}>
   <IconButton label={collapsed?'Развернуть график':'Свернуть график'} onClick={()=>setCollapsed(!collapsed)}/>
   <span onMouseEnter={e=>{if(mode==='station')showName(e.currentTarget,title);}} onMouseLeave={()=>setNameTip(undefined)}>{title}</span><IconButton label="Закрыть график" onClick={()=>setClosing(true)}/>
  </header>
  <div className="floating-body" inert={collapsed||closing} aria-hidden={collapsed}>{content}</div>
  {nameTip&&!collapsed&&!closing&&createPortal(<div className="floating-name-popup" role="tooltip" style={{left:nameTip.x,top:nameTip.y}}>{nameTip.text}</div>,nameTip.overlay)}
 </section>;
}
