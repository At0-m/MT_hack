import { useEffect, useRef } from 'react';
import type { Calculation } from '../../api/types';
import type { DisplayFrame } from './serviceDay';
import { dateLabel } from './dates';
export function Timeline({ calculation, frames, index, onChange, wheel = false }: {
 calculation: Calculation; frames?: DisplayFrame[]; index: number; onChange: (i: number) => void; wheel?: boolean;
}) {
 const ref = useRef<HTMLDivElement>(null), latest = useRef({ index, onChange });
 latest.current = { index, onChange };
 const windows=frames?.map(f=>f.calculation.frames[f.index].window)??calculation.frames.map(f=>f.window);
 const count = windows.length;
 useEffect(() => {
  const el = ref.current!; let accumulated = 0;
  const handler = (e: WheelEvent) => {
   e.preventDefault(); e.stopPropagation(); accumulated += e.deltaY;
   if (Math.abs(accumulated) < 25) return;
   latest.current.onChange(Math.max(0, Math.min(count - 1, latest.current.index + Math.sign(accumulated)))); accumulated = 0;
  };
  el.addEventListener('wheel', handler, { passive: false });
  return () => el.removeEventListener('wheel', handler);
 }, [count]);
 const hourly = calculation.descriptor.selection.view_mode === 'day';
 const label = (i: number, format = hourly ? 'HH:mm' : 'dd.MM') => dateLabel(windows[i].from, format);
 return <div className={wheel ? `wheel ${hourly ? 'wheel-hours' : 'wheel-days'}` : 'timeline'} ref={ref}>
  {wheel ? <>
   <div className="wheel-items" role="slider" tabIndex={0} aria-label="Временной кадр" aria-valuemin={0} aria-valuemax={count - 1} aria-valuenow={index} aria-valuetext={label(index)} onKeyDown={e => {
    const next = e.key === 'Home' ? 0 : e.key === 'End' ? count - 1 : ['ArrowDown', 'ArrowRight'].includes(e.key) ? index + 1 : ['ArrowUp', 'ArrowLeft'].includes(e.key) ? index - 1 : undefined;
    if (next !== undefined) { e.preventDefault(); onChange(Math.max(0, Math.min(count - 1, next))); }
   }}>
    {[-2, -1, 0, 1, 2].map(offset => {
     const i = index + offset;
     return i >= 0 && i < count ? <button key={offset} className={`wheel-row wheel-row-${offset + 2}`} aria-label={label(i)} aria-current={offset === 0 ? 'time' : undefined} onClick={() => onChange(i)}>{label(i, hourly ? 'HH' : 'dd')}</button> : null;
    })}
   </div>
   {hourly ? <><span className="wheel-colon" aria-hidden="true">:</span><span className="wheel-minute" aria-hidden="true">00</span></> : <span className="wheel-month">{label(index, 'MMMM')}</span>}
  </> : <><input aria-label="Временной кадр" type="range" min={0} max={count - 1} step={1} value={index} onChange={e => onChange(Number(e.target.value))} /><div className="timeline-labels"><span>{label(0)}</span><strong>{label(index)}</strong><span>{label(count - 1)}</span></div></>}
 </div>;
}
