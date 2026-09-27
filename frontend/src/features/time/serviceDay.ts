import type { Calculation, Overrides, Selection } from '../../api/types';
import { iso, moscow } from './dates';

export type DisplayFrame = { calculation: Calculation; index: number };
export function nextDaySelection(selection: Selection): Selection | undefined {
 if (selection.view_mode !== 'day') return;
 return { ...selection, window: { from: selection.window.to, to: iso(moscow(selection.window.to).plus({ days: 1 })) } };
}
export function overridesForDay(selection: Selection, overrides?: Overrides): Overrides | undefined {
 if (!overrides) return;
 const from = moscow(overrides.effective_window.from), to = moscow(overrides.effective_window.to);
 if (from >= moscow(selection.window.from) && to <= moscow(selection.window.to)) return overrides;
}
/** Keep the original response and its identity for every frame, including after midnight. */
export function serviceFrames(calculation: Calculation, nextDay?: Calculation): DisplayFrame[] {
 const frames = calculation.frames.map((_, index) => ({ calculation, index }));
 if (calculation.descriptor.selection.view_mode !== 'day' || !nextDay) return frames;
 return [...frames.filter(f => moscow(f.calculation.frames[f.index].window.from).hour >= 6),
  ...(nextDay?.frames.map((_, index) => ({ calculation: nextDay, index })) ?? [])
   .filter(f => moscow(f.calculation.frames[f.index].window.from).hour < 2)];
}
