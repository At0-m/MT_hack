import { useEffect, useRef, useState } from 'react';
import type { Calculation, Overrides } from '../../api/types';
import { dateLabel } from '../time/dates';
import { factorKeys, parseFactors, type FactorDraft } from './factorValues';

const names = { weather: 'Погода', event: 'События', season: 'Сезон' };

export function Factors({ calculation, allowed, pending, onDirty, onChange, onReset }: {
  calculation: Calculation;
  allowed: string[];
  pending: boolean;
  onDirty: () => void;
  onChange: (overrides: Overrides) => void;
  onReset: () => void;
}) {
  const previous = calculation.descriptor.kind === 'scenario'
    ? calculation.descriptor.overrides : undefined;
  const [draft, setDraft] = useState<FactorDraft>({
    weather: String(previous?.factors?.weather ?? 1),
    event: String(previous?.factors?.event ?? 1),
    season: String(previous?.factors?.season ?? 1),
  });
  const [edited, setEdited] = useState(false);
  const callback = useRef(onChange);
  callback.current = onChange;

  // Preserve the fleet override and its explicit window, including a single hour.
  const effectiveWindow = previous?.effective_window ?? calculation.descriptor.selection.window;
  const windowKey = JSON.stringify(effectiveWindow);
  const fleetKey = JSON.stringify(previous?.fleet ?? null);
  const allowedKey = JSON.stringify(allowed);
  const factors = parseFactors(draft);

  useEffect(() => {
    if (!edited) return;
    const values = parseFactors(draft);
    if (!values) return;
    const permitted = JSON.parse(allowedKey) as string[];
    const fleet = JSON.parse(fleetKey) as Overrides['fleet'] | null;
    const timer = setTimeout(() => {
      callback.current({
        effective_window: JSON.parse(windowKey),
        ...(fleet ? { fleet } : {}),
        factors: {
          weather: permitted.includes('weather') ? values.weather : 1,
          event: permitted.includes('event') ? values.event : 1,
          season: permitted.includes('season') ? values.season : 1,
        },
      });
    }, 450);
    return () => clearTimeout(timer);
  }, [draft, edited, windowKey, fleetKey, allowedKey]);

  return <div className="factors-editor" aria-busy={pending}>
    <h2>Поправки к прогнозу</h2>
    <p>{dateLabel(effectiveWindow.from, 'dd.MM HH:mm')} — {dateLabel(effectiveWindow.to, 'dd.MM HH:mm')} МСК</p>
    {factorKeys.filter(key => allowed.includes(key)).map(key => <label key={key}>
      <span>{names[key]}</span>
      <input
        aria-label={names[key]}
        type="number"
        min="0.5"
        max="2"
        step="0.001"
        value={draft[key]}
        onChange={event => {
          setDraft(value => ({ ...value, [key]: event.target.value }));
          setEdited(true);
          onDirty();
        }}
      />
    </label>)}
    <p className="caption">Прогноз обновляется автоматически после изменения коэффициентов.</p>
    <p className="caption">Ручной сценарий. Расчёт выполняет сервер.</p>
    {!factors && <p role="alert">Каждый коэффициент: 0,5–2, до трёх знаков после запятой. Произведение: 0,25–4.</p>}
    {pending && <p role="status">Пересчитываем прогноз…</p>}
    <div className="factor-actions">
      <button disabled={pending} onClick={onReset}>Исходный прогноз</button>
    </div>
  </div>;
}
