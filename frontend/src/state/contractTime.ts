import type { Window, Selection } from '../api/types';

/** HTTP identity uses instants, not timestamp spelling (+03:00 versus Z).
 * Supported Moscow forecast buckets are fixed 1h/24h, matching Go's contract. */
export function sameWindow(a: Window, b: Window): boolean {
 const from=Date.parse(a.from), to=Date.parse(a.to);
 return Number.isFinite(from) && Number.isFinite(to) && from===Date.parse(b.from) && to===Date.parse(b.to);
}
export function contractWindows(s: Selection): Window[] {
 const start=Date.parse(s.window.from), end=Date.parse(s.window.to);
 const step=s.resolution==='hour'?3600000:86400000;
 if(!Number.isFinite(start)||!Number.isFinite(end)||end<=start||(end-start)%step!==0|| (end-start)/step>8784) throw new Error('Invalid frame interval.');
 return Array.from({length:(end-start)/step},(_,i)=>({from:new Date(start+i*step).toISOString(),to:new Date(start+(i+1)*step).toISOString()}));
}
