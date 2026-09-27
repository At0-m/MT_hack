import { IncomingIcon } from '../assets/IncomingIcon';
import { useEffect, useRef, useState } from 'react';
import { api, ApiError, errorText, MOCK, DEV_TOOLS } from '../api/client';
import type { Schema, StopFocus } from '../api/types';
import { Auth } from '../features/auth/Auth';
import { useDispatcher } from '../state/useDispatcher';
import { TransportMap } from '../features/map/TransportMap';
import { Calendar } from '../features/time/Calendar';
import { Timeline } from '../features/time/Timeline';
import { serviceFrames } from '../features/time/serviceDay';
import { dateLabel } from '../features/time/dates';
import { Popup, metric } from '../ui/common';
import { Fleet } from '../features/scenario/Fleet';
import { RoutePicker } from '../features/routes/RoutePicker';
import { Analytics } from '../features/analytics/Analytics';
import { FloatingChart } from '../features/charts/FloatingChart';
import { Indicators } from '../features/analytics/Indicators';
import { loadColor } from '../features/map/columns';

function Dispatcher({onLogout}:{onLogout:()=>Promise<void>}){
  const d=useDispatcher();
  const [index,setIndex]=useState(0);
  const [popup,commitPopup]=useState<string>();
  const [popupClosing,setPopupClosing]=useState(false);
  const nextPopup=useRef<string|undefined>(undefined);
  const setPopup=(next:string|undefined)=>{
    nextPopup.current=next;
    if(popup&&next!==popup){setPopupClosing(true);return;}
    setPopupClosing(false);commitPopup(next);
  };
  useEffect(()=>{
    if(!popupClosing)return;
    const timer=window.setTimeout(()=>{commitPopup(nextPopup.current);setPopupClosing(false);},520);
    return()=>window.clearTimeout(timer);
  },[popupClosing]);
  const [stationClosing,setStationClosing]=useState(false);
  const [panel,setPanel]=useState<'overall'|StopFocus>();
  const [pattern,setPattern]=useState<string>();
  const [exportError,setExportError]=useState('');
  const [exporting,setExporting]=useState(false);
  const [toolsOpen,setToolsOpen]=useState(false);
  const [chartWindows,setChartWindows]=useState<number[]>([]);
  const [nextChartId,setNextChartId]=useState(0);

  const c=d.confirmed?.calculation;
  const selection=c?.descriptor.selection;
  const b=d.bootstrap;
  const dateSelection=d.pendingSelection??selection;
  const frames=c?serviceFrames(c,d.confirmed?.nextDay):[];
  const displayIndex=Math.min(index,Math.max(0,frames.length-1));
  const activeCalculation=frames[displayIndex]?.calculation??c;
  const frameIndex=frames[displayIndex]?.index??0;
  const reading=activeCalculation?.frames[frameIndex]?.routes.find(r=>r.route_id===selection?.route_ids[0]);
  const blocked=d.pending||d.dirty;
  const stop=panel&&panel!=='overall'?panel:undefined;
  const selectedFeature=d.confirmed?.geometry.features.find(f=>f.properties.kind==='stop'&&f.properties.route_stop_id===stop?.route_stop_id&&f.properties.route_pattern_id===stop?.route_pattern_id);
  const title=stop&&selectedFeature&&'name' in selectedFeature.properties?selectedFeature.properties.name:'Общая загруженность';
  const toggle=(p:string)=>setPopup(popup===p?undefined:p);

  useEffect(()=>{setExportError('');},[activeCalculation?.calculation_id]);

  const exportCsv=async()=>{
    if(!activeCalculation||blocked)return;
    setExporting(true);
    setExportError('');
    try{
      const blob=await api.export(activeCalculation.descriptor,activeCalculation.calculation_id);
      const url=URL.createObjectURL(blob);
      const a=document.createElement('a');
      a.href=url;
      a.download=`transport-${activeCalculation.calculation_id.slice(5,13)}.csv`;
      a.click();
      setTimeout(()=>URL.revokeObjectURL(url),1000);
    }catch(e){
      setExportError(errorText(e));
    }finally{
      setExporting(false);
    }
  };

  return (
    <main className={`dispatcher ${panel?'drawer-open':''}`}>
      {b&&<TransportMap config={b.map} geometry={d.confirmed?.geometry} calculation={activeCalculation} index={frameIndex} selected={stationClosing?undefined:stop} patternId={pattern} onSelect={s=>{setStationClosing(false);setPanel(s);setPopup(undefined);}}/>}
      <div className="ui-overlay">
        {c&&selection&&b&&<>
          <nav className="topbar" aria-label="Управление расчётом">
            <button data-popup-trigger className="top-control time-control" aria-label="Выбор временного кадра" aria-expanded={popup==='time'} onClick={()=>toggle('time')}>
              <span className="digits">{dateLabel(activeCalculation!.frames[frameIndex].window.from,selection.view_mode==='day'?'HH:mm':'dd.MM')}</span>
            </button>
            <button data-popup-trigger className="top-control date-control" aria-label="Выбор периода" aria-expanded={popup==='calendar'} onClick={()=>toggle('calendar')}>
              {dateSelection!.view_mode==='day'?<span className="digits">{dateLabel(d.pendingSelection?dateSelection!.window.from:activeCalculation!.frames[frameIndex].window.from)}</span>:<span className={`date-mode-label date-mode-label--${dateSelection!.view_mode}`}><IncomingIcon name="calendar.png" className="date-calendar-placeholder"/><span>{{week:'неделя',month:'месяц',custom:'выборочный'}[dateSelection!.view_mode as 'week'|'month'|'custom']}</span></span>}
            </button>
            <button data-popup-trigger className="top-control route-control" aria-label="Выбор маршрута" aria-expanded={popup==='routes'} onClick={()=>toggle('routes')}>
              Маршрут<span className="digits">{'\u2009'}{d.confirmed!.route.route.route_number}</span>
            </button>
            <button data-popup-trigger className="top-control fleet-control" disabled={!b.capabilities.scenario_fleet} aria-label="Управление выпуском" aria-expanded={popup==='fleet'} onClick={()=>toggle('fleet')}>
              <IncomingIcon name="train.png" className="fleet-icon-placeholder"/><span className="digits">{reading?.evaluated.mean_vehicle_count.status==='available'?metric(reading.evaluated.mean_vehicle_count):'Н/Д'}</span>
            </button>
            <button className="top-control index-control" aria-label="Общая аналитика" onClick={()=>{setStationClosing(false);setPanel('overall');setPopup(undefined);}}>
               <span className="index-symbol" aria-hidden="true" style={{'--index-color':reading?.evaluated.load_index.status==='available'?loadColor(reading.evaluated.load_index.value):undefined} as React.CSSProperties}><span/><span/><span/></span>
               <span className="digits" style={{color:reading?.evaluated.load_index.status==='available'?loadColor(reading.evaluated.load_index.value):undefined}}>{reading?.evaluated.load_index.status==='available'?metric(reading.evaluated.load_index,true):'Н/Д'}</span>
            </button>
          </nav>
          {popup&&<Popup key={popup} label={{time:'Временные кадры',calendar:'Выбор периода',routes:'Маршруты',fleet:'Сценарий выпуска'}[popup]!} className={`popup-${popup}${popupClosing?' popup-leaving':''}`} onClose={()=>setPopup(undefined)}>
            {popup==='time'&&<Timeline calculation={c} frames={frames} index={displayIndex} onChange={setIndex} wheel/>}
            {popup==='calendar'&&<Calendar selection={selection} bootstrap={b} onApply={(s,close=true)=>{setIndex(0);setPanel(undefined);if(close)setPopup(undefined);void d.calculate(s);}}/>}
            {popup==='routes'&&<RoutePicker routeDetail={d.confirmed!.route} bootstrap={b} selection={selection} index={frameIndex} onChoose={routeId=>{setPanel(undefined);setPattern(undefined);setPopup(undefined);setIndex(0);void d.calculate({...selection,route_ids:[routeId]});}}/>}
            {popup==='fleet'&&<Fleet key={activeCalculation!.frames[frameIndex].window.from} calculation={activeCalculation!} index={frameIndex} pending={d.pending} onDirty={d.markDirty} onChange={o=>void d.calculate(selection,o)}/>}
          </Popup>}
          <Indicators items={reading?.indicators??[]}/>
          {panel&&<Analytics key={typeof panel==='string'?panel:panel.route_stop_id} calculation={activeCalculation!} index={frameIndex} frames={frames} displayIndex={displayIndex} onFrame={setIndex} focus={stop} title={title} blocked={blocked} stopSupported={b.capabilities.stop_forecasts} onCloseStart={()=>setStationClosing(true)} onClose={()=>{setPanel(undefined);setStationClosing(false);}}/>}
        </>}
          {activeCalculation&&d.confirmed&&chartWindows.map(id=><FloatingChart key={id} ordinal={id} geometry={d.confirmed!.geometry} calculation={activeCalculation} frameIndex={frameIndex} frames={frames} displayIndex={displayIndex} onFrame={setIndex} onClose={()=>setChartWindows(w=>w.filter(value=>value!==id))}/>)}
          <div className={`utility-bar ${toolsOpen?'is-open':''}`}>
            <button className="utility-toggle" aria-label={toolsOpen?'Закрыть панель инструментов':'Открыть панель инструментов'} aria-expanded={toolsOpen} aria-controls="utility-actions" onClick={()=>setToolsOpen(!toolsOpen)}>
             <IncomingIcon name={toolsOpen?"cross.png":"wrench.png"} className="utility-icon-placeholder"/>
            </button>
            <div id="utility-actions" className="utility-actions" inert={!toolsOpen}>


            {d.confirmed&&d.confirmed.route.patterns.length>1&&(
              <select aria-label="Направление маршрута" value={pattern??''} onChange={e=>{setPattern(e.target.value||undefined);setPanel(undefined);}}>
                <option value="">Все направления</option>
                {d.confirmed.route.patterns.map(p=><option key={p.route_pattern_id} value={p.route_pattern_id}>{p.name}</option>)}
              </select>
            )}
            {c&&<button disabled={blocked||exporting||!b?.capabilities.csv_export} onClick={()=>void exportCsv()}>CSV</button>}
            <button disabled={d.pending} onClick={()=>{setPanel(undefined);setPopup(undefined);setIndex(0);void d.initialize();}}>ОБНОВИТЬ</button>
            <button disabled={!c||!d.confirmed} onClick={()=>{setChartWindows(w=>[...w,nextChartId]);setNextChartId(id=>id+1);}}>ВЫНЕСТИ ГРАФИК</button>
            <button onClick={()=>void onLogout().catch(e=>setExportError(errorText(e)))}>ВЫЙТИ</button>
            </div>
          </div>
          {(d.pending||d.dirty||d.error||exportError)&&<div className="status-message" role={d.error||exportError?'alert':'status'}>
            {d.pending?'Загрузка согласованного расчёта…':d.error||exportError||'Изменения ещё не подтверждены. Показан предыдущий расчёт.'}
            {d.error&&<><button onClick={()=>void d.retry()}>Повторить</button>{MOCK&&<button onClick={()=>{document.cookie='mt_mock_fault=; Path=/; Max-Age=0; SameSite=Strict';location.reload();}}>Сбросить демо-ошибку</button>}{c&&<button onClick={()=>{d.discard();setPopup(undefined);}}>Отменить изменения</button>}</>}
          </div>}
          {!b&&!d.pending&&!d.error&&<p className="status-message">Подготовка данных…</p>}
          {MOCK&&DEV_TOOLS&&<details className="mock-tools"><summary>ДЕМО · синтетические данные · Проверки демо</summary><select aria-label="Демонстрационная ошибка" defaultValue={document.cookie.split('; ').find(value=>value.startsWith('mt_mock_fault='))?.split('=')[1]??''} onChange={e=>{document.cookie=`mt_mock_fault=${e.target.value}; Path=/; SameSite=Strict`;location.reload();}}>
            {[['','Без ошибок'],['no-stops','Нет остановочных прогнозов'],['unknown-fleet','Неизвестный выпуск'],['slow','Медленный ответ'],['partial','Неполный ответ'],['error','Ошибка 503'],['mismatch','Summary/CSV: 409'],['gone','Summary/CSV: 410'],['summary-unavailable','Сводка недоступна'],['expired','Сессия истекла']].map(([v,l])=><option value={v} key={v}>{l}</option>)}
          </select></details>}
      </div>
    </main>
  );
}

export function App(){
  const [session,setSession]=useState<Schema['AuthSession']>();
  const [checking,setChecking]=useState(true);
  const [error,setError]=useState('');
  const [expired,setExpired]=useState('');

  const check=async()=>{
    setChecking(true);
    setError('');
    try{
      setSession(await api.session());
    }catch(e){
      if(!(e instanceof ApiError&&e.problem.status===401))setError(errorText(e));
    }finally{
      setChecking(false);
    }
  };

  useEffect(()=>{
    void check();
    const expire=()=>{setSession(undefined);setExpired('Сессия истекла. Войдите снова.');};
    window.addEventListener('session-expired',expire);
    return()=>window.removeEventListener('session-expired',expire);
  },[]);

  if(checking)return <main className="startup" role="status">Проверка сессии…</main>;
  if(error)return <main className="startup"><p role="alert">{error}</p><button onClick={()=>void check()}>Повторить</button></main>;
  return session?<Dispatcher onLogout={async()=>{await api.logout();setSession(undefined);setExpired('');}}/>:<Auth message={expired} onLogin={s=>{setSession(s);setExpired('');}}/>;
}
