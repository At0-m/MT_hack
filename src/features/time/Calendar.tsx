import { IncomingIcon } from '../../assets/IncomingIcon';
import { useState } from 'react';
import type { Bootstrap, Selection, Schema } from '../../api/types';
import { makeWindow, moscow, validateSelection } from './dates';
export const modeNames = { day: 'День', week: 'Неделя', month: 'Месяц', custom: 'Свой' };

export function Calendar({ selection, bootstrap, onApply }: {
 selection: Selection; bootstrap: Bootstrap; onApply: (s: Selection, close?: boolean) => void;
}) {
 const [mode, setMode] = useState(selection.view_mode);
 const [start, setStart] = useState(moscow(selection.window.from).toISODate()!);
 const [end, setEnd] = useState(moscow(selection.window.to).minus({ days: 1 }).toISODate()!);
 const [month, setMonth] = useState(moscow(selection.window.from).startOf('month'));
 const [pickingEnd, setPickingEnd] = useState(false);
 const [error, setError] = useState('');
 let range: Schema['TimeWindow'] | undefined;
 try { range = makeWindow(mode, start, end); } catch { /* Draft remains editable. */ }
 const first = month.minus({ days: month.weekday - 1 });
 const rowCount = Math.max(5, Math.ceil(((month.daysInMonth ?? 31) + month.weekday - 1) / 7));
 const weeks = Array.from({ length: rowCount }, (_, row) =>
  Array.from({ length: 7 }, (_, col) => first.plus({ days: row * 7 + col })));
 const apply = (from: string, to: string, nextMode = mode, close = true) => {
  try {
   const next = { ...selection, view_mode: nextMode, resolution: nextMode === 'day' ? 'hour' : 'day', window: makeWindow(nextMode, from, to) } as Selection;
   validateSelection(next, bootstrap);
   onApply(next, close);
  } catch (e) { setError((e as Error).message); }
 };
 const choose = (day: string) => {
  setError('');
  if (mode === 'custom' && pickingEnd) {
   const from = day < start ? day : start, to = day < start ? start : day;
   setStart(from); setEnd(to); setPickingEnd(false); apply(from, to);
  } else {
   setStart(day); setEnd(day); setPickingEnd(mode === 'custom');
   if (mode !== 'custom') apply(day, day);
  }
 };
 return <div className="calendar">
  <div className="calendar-head">
   <button aria-label="Предыдущий месяц" onClick={() => setMonth(month.minus({ months: 1 }))}><IncomingIcon name="down.png" className="arrow-left"/></button>
   <strong title={month.toFormat('yyyy')}>{month.setLocale('ru').toFormat('LLLL')}</strong>
   <button aria-label="Следующий месяц" onClick={() => setMonth(month.plus({ months: 1 }))}><IncomingIcon name="down.png" className="arrow-right"/></button>
  </div>
  <label className="mode-select"><span aria-hidden="true">Выбранный интервал:{'\u2009'}{modeNames[mode].toLocaleLowerCase('ru')}</span><select aria-label="Режим периода" value={mode} onChange={e => {
   const nextMode = e.target.value as Schema['ViewMode'];
   setMode(nextMode); setPickingEnd(false); setError('');
   apply(start, end, nextMode, false);
  }}>
   {Object.entries(modeNames).map(([id, name]) => <option key={id} value={id}>{id === 'custom' ? name : name.toLocaleLowerCase('ru')}</option>)}
  </select></label>
  <div className="calendar-weekdays">{['пн', 'вт', 'ср', 'чт', 'пт', 'сб', 'вс'].map(d => <span key={d}>{d}</span>)}</div>
  <div className="calendar-grid" role="group" aria-label="Календарь">
   {weeks.map((days, row) => {
    const selected = days.map(d => !!range && d >= moscow(range.from) && d < moscow(range.to));
    const from = selected.indexOf(true), to = selected.lastIndexOf(true);
    return <div className="calendar-week" key={row}>
     {from >= 0 && <span aria-hidden="true" className="calendar-range" style={{ gridColumn: `${from + 1} / ${to + 2}` }} />}
     {days.map((d, col) => <button key={d.toISODate()} aria-label={d.toISODate()!} aria-pressed={selected[col]} style={{ gridColumn: col + 1, gridRow: 1 }} className={`${d.month !== month.month ? 'other-month' : ''} ${selected[col] ? 'range-day' : ''}`} onClick={() => choose(d.toISODate()!)}>{d.toFormat('dd')}</button>)}
    </div>;
   })}
  </div>
  <p className="sr-only" role="status">{month.toFormat('yyyy')}. {pickingEnd ? 'Выберите последнюю дату диапазона.' : ''}</p>
  {error && <p className="calendar-error" role="alert">{error}</p>}
 </div>;
}
