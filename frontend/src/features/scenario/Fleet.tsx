import { useEffect, useRef, useState } from 'react';
import type { Calculation, Overrides } from '../../api/types';

export function Fleet({ calculation, index, onChange, onDirty, pending }: {
 calculation: Calculation; index: number; onChange: (o: Overrides) => void;
 onDirty: () => void; pending: boolean;
}) {
 const frame = calculation.frames[index];
 const current = frame.routes[0].evaluated.mean_vehicle_count;
 const [draft, setDraft] = useState<string>();
 const [error, setError] = useState('');
 const callback = useRef(onChange); callback.current = onChange;
 const value = draft ?? (current.status === 'available' ? String(current.value) : '');
 const effectiveKey = JSON.stringify(frame.window);
 useEffect(() => {
  if (draft === undefined) return;
  const v = Number(draft);
  if (!draft.trim() || !Number.isInteger(v) || v < 0 || v > 200) {
   setError('Количество трамваев: целое число от 0 до 200.'); return;
  }
  setError('');
  const timer = setTimeout(() => callback.current({ effective_window: JSON.parse(effectiveKey), fleet: { kind: 'absolute', vehicle_count: v }, ...(calculation.descriptor.kind === 'scenario' ? {factors: calculation.descriptor.overrides.factors} : {}) }), 450);
  return () => clearTimeout(timer);
 }, [draft, effectiveKey]);
 const change = (next: string) => { setDraft(next===''?'':String(Math.min(200,Math.max(0,Math.floor(Number(next)))))); onDirty(); };
 return <div className="fleet-editor" aria-busy={pending}>
  <div className="fleet-step">
   <button aria-label="Уменьшить выпуск" disabled={Number(value) <= 0} onClick={() => change(String(Math.max(0, Math.floor(Number(value)) - 1)))}>-</button>
   <input type="number" aria-label="Количество трамваев" placeholder="Н/Д" value={value} min={0} max={200} step={1} onChange={e => change(e.target.value)} />
   <button aria-label="Увеличить выпуск" disabled={Number(value) >= 200} onClick={() => change(String(Math.min(200, Math.floor(Number(value)) + 1)))}>+</button>
  </div>
  {error && <p className="fleet-error" role="alert">{error}</p>}
 </div>;
}
