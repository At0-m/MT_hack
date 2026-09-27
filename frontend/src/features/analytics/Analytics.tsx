import { IncomingIcon } from '../../assets/IncomingIcon';
import { useEffect, useRef, useState } from 'react';
import type { CSSProperties } from 'react';
import type { Calculation, StopFocus } from '../../api/types';
import { selectReading } from '../../state/calculation';
import type { DisplayFrame } from '../time/serviceDay';
import { dateLabel } from '../time/dates';
import { metric } from '../../ui/common';
import { PlainIndicators } from './Indicators';
import { loadColor } from '../map/columns';
import { Summary } from './Summary';

function StationTitle({title}:{title:string}) {
 const viewport=useRef<HTMLHeadingElement>(null),text=useRef<HTMLSpanElement>(null);
 const [travel,setTravel]=useState(0);
 useEffect(()=>{
  let alive=true;
  const measure=()=>{if(alive&&viewport.current&&text.current)setTravel(text.current.offsetWidth>viewport.current.clientWidth?text.current.offsetWidth+48:0);};
  const observer=new ResizeObserver(measure);
  if(viewport.current)observer.observe(viewport.current);
  if(text.current)observer.observe(text.current);
  void document.fonts.ready.then(measure);measure();
  return()=>{alive=false;observer.disconnect();};
 },[title]);
 return <h1 ref={viewport} title={title} aria-label={title} className={travel?'station-title scrolling':'station-title'} style={{'--title-travel':`${travel}px`,'--title-duration':`${Math.max(8,travel/55)}s`} as CSSProperties}>
  <span className="station-title-track"><span ref={text}>{title}</span>{travel>0&&<span aria-hidden="true">{title}</span>}</span>
 </h1>;
}

export function Analytics({ calculation, index, frames: allFrames, displayIndex: globalIndex, onFrame: onGlobalFrame, focus, title, stopSupported, blocked, onCloseStart, onClose }: {
 calculation: Calculation; index: number; frames: DisplayFrame[]; displayIndex: number; onFrame: (i: number) => void; focus?: StopFocus;
 title: string; blocked: boolean; stopSupported: boolean; onCloseStart: () => void; onClose: () => void;
}) {
 const drawerRef=useRef<HTMLElement>(null);
 const closing=useRef(false);
 const closeDrawer=()=>{
  if(closing.current)return;
  closing.current=true;onCloseStart();
  const element=drawerRef.current;
  if(!element){onClose();return;}
  const from=getComputedStyle(element).transform;
  const animation=element.animate([{transform:from==='none'?'translateX(0)':from},{transform:'translateX(110%)'}],{duration:760,easing:'cubic-bezier(.4,0,.2,1)',fill:'forwards'});
  void animation.finished.then(onClose).catch(()=>{closing.current=false;});
 };
 const routeId = calculation.descriptor.selection.route_ids[0];
 const reading = selectReading(calculation, index, routeId, focus);
 const unsupported = !!focus && !stopSupported;
 const paginated = calculation.descriptor.selection.view_mode === 'custom' && allFrames.length > 31;
 const page = paginated ? Math.floor(globalIndex / 31) : 0;
 const pageCount = Math.ceil(allFrames.length / 31);
 const offset = paginated ? page * 31 : 0;
 const frames = paginated ? allFrames.slice(offset, offset + 31) : allFrames;
 const displayIndex = globalIndex - offset;
 const onFrame = (i:number) => onGlobalFrame(offset + i);
 const indexColor = reading?.evaluated.load_index.status === 'available' ? loadColor(reading.evaluated.load_index.value,calculation.visualization.thresholds) : undefined;
 const ceiling = calculation.visualization.scale_max;
 const hourly = calculation.descriptor.selection.view_mode === 'day';
 const frameLabel = (i: number) => dateLabel(frames[i].calculation.frames[frames[i].index].window.from, hourly ? 'HH:mm' : 'dd.MM');
 return <aside ref={drawerRef} className={`analytics analytics-stop ${focus ? '' : 'analytics-overall'}`} aria-label={title}>
  <button className="drawer-close" aria-label="Закрыть аналитику" onClick={closeDrawer}>
    <IncomingIcon name="down.png" className="drawer-icon-placeholder arrow-right"/>
  </button>
  <div className="drawer-scroll"><div className="drawer-content">
   <header className="analytics-heading">{focus?<StationTitle title={title}/>:<h1 title={title}>{title}</h1>}<img className="heading-line" src={`/assets/icons/${focus ? 'ca39e.svg' : 'faa36.svg'}`} alt="" />{focus && <span className="heading-station"><IncomingIcon name="train_white.png" className="station-icon-placeholder"/></span>}</header>
   {unsupported ? <p className="analytics-unavailable" role="status">Остановочные прогнозы недоступны.</p> : <>
    {focus && <h2 className="chart-heading">Индекс загруженности</h2>}
    <div className="chart" role="group" aria-label="График индекса по кадрам">
     <img className="chart-frame" src="/assets/icons/ed3ac.svg" alt="" />
     <div className="station-chart-grid" aria-hidden="true">{[20,40,60,80,100].map(v=><span key={v} style={{bottom:`${v}%`}}/>)}</div>
     <div className="chart-axis">{[1,.8,.6,.4,.2].map(v=><span key={v} style={{bottom:`${v*100}%`}}>{Math.round(ceiling*v*100)}</span>)}</div>
     <div className="chart-columns">{frames.map((f, i) => {
      const r = selectReading(f.calculation, f.index, routeId, focus), rawValue = r?.evaluated.load_index;
      const value = rawValue;
      return <button key={f.calculation.frames[f.index].window.from} className={`chart-column ${i === displayIndex ? 'selected' : ''}`} aria-label={`${frameLabel(i)}: ${metric(value, true)}`} aria-pressed={i === displayIndex} onClick={() => onFrame(i)}>
       <span className={value?.status === 'available' ? 'bar' : 'missing'} style={{ height: value?.status === 'available' ? `${Math.min(100,Math.max(0, value.value / ceiling * 100))}%` : '8px', '--scale-height': `${Math.max(1, value?.value ?? 0) / Math.max(.01, value?.value ?? 1) * 100}%`, background: i !== displayIndex ? '#bbc5d2' : value?.status === 'available' ? `linear-gradient(to top, #00ff00 0, #00ff00 ${Math.min(100, calculation.visualization.thresholds[0] / Math.max(.01,value.value) * 100)}%, #f6ff00 ${Math.min(100, calculation.visualization.thresholds[1] / Math.max(.01,value.value) * 100)}%, #ffb44a ${Math.min(100, calculation.visualization.thresholds[2] / Math.max(.01,value.value) * 100)}%, #ff0000 ${Math.min(100, calculation.visualization.scale_max / Math.max(.01,value.value) * 100)}%)` : 'transparent' } as CSSProperties} />
        {i !== displayIndex && <span className="chart-value" style={{bottom:value?.status==='available'?`${Math.min(100,value.value/ceiling*100)}%`:'8px',color:'#bbc5d2'}}><span>{metric(value,true)}</span><small>{frameLabel(i)}</small></span>}
      </button>;
     })}</div>
     <div className="chart-ticks">{Array.from(new Set((calculation.descriptor.selection.view_mode==='week'?[0,3,6]:[0,.25,.5,.75,1].map(v=>Math.round((frames.length-1)*v))).filter(i=>i<frames.length))).map(i=><span key={i} style={{left:`${i/Math.max(1,frames.length-1)*100}%`}}>{frameLabel(i)}</span>)}</div>
    </div>
    <div className="analytics-timeline">
     <img className="timeline-rail" src={hourly?"/assets/icons/station-timeline-rail.svg":"/assets/icons/station-timeline-rail-days.svg"} alt="" /><span className="timeline-bus" aria-hidden="true" style={{left:`${(52+366*displayIndex/Math.max(1,frames.length-1))/470*100}%`}}><IncomingIcon name="train_white.png"/></span>
     {!hourly&&<><IncomingIcon name="calendar.png" className="timeline-calendar-start"/><IncomingIcon name="calendar.png" className="timeline-calendar-end"/></>}
     <input type="range" aria-label="Временной кадр аналитики" min={0} max={frames.length - 1} step={1} value={displayIndex} onChange={e => onFrame(Number(e.target.value))} />
     <div className="analytics-time-labels"><span>{frameLabel(0)}</span><strong><span className="timeline-date">{dateLabel(calculation.frames[index].window.from,'dd')}{' '}<span className="timeline-month">{dateLabel(calculation.frames[index].window.from,'MMMM')}</span>{hourly?`, ${dateLabel(calculation.frames[index].window.from,'HH:mm')}`:''}</span></strong><span>{frameLabel(frames.length - 1)}</span></div>
     {paginated && <div className="timeline-pages" aria-label="Страницы периода">
      <button aria-label="Предыдущая страница" disabled={page===0} onClick={()=>onGlobalFrame((page-1)*31)}><IncomingIcon name="down.png" className="arrow-left"/></button>
      <span aria-live="polite">страница {page+1}</span>
      <button aria-label="Следующая страница" disabled={page+1>=pageCount} onClick={()=>onGlobalFrame((page+1)*31)}><IncomingIcon name="down.png" className="arrow-right"/></button>
     </div>}
    </div>
    <h2 className="assessment-title">Оценка загруженности</h2>
    <div className="assessment">
      <strong className="big-index" style={{ color: indexColor, fontSize:metric(reading?.evaluated.load_index,true).length>3?85:undefined }}>{metric(reading?.evaluated.load_index, true)}</strong>
      <p className="comparison">{reading?.relative_to_typical != null ? <>на <span style={{ color: indexColor }}>{Math.round(Math.abs(reading.relative_to_typical) * 100)}%</span> {reading.relative_to_typical < 0 ? 'меньше' : 'больше'}<br />чем обычно</> : 'Сравнение недоступно'}</p>
    </div>
   </>}
   {!focus&&<><PlainIndicators items={reading && 'indicators' in reading ? reading.indicators : []}/><Summary calculation={calculation} blocked={blocked}/></>}
  </div></div>
 </aside>;
}
