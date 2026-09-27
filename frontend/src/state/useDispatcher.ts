import { useCallback, useEffect, useRef, useState } from 'react';
import { api, errorText } from '../api/client';
import type { Bootstrap, Calculation, Geometry, Overrides, Schema, Selection } from '../api/types';
import { LatestRequest, assertCalculation } from './calculation';
import { nextDaySelection, overridesForDay } from '../features/time/serviceDay';
import { validateSelection } from '../features/time/dates';

type Confirmed = { calculation: Calculation; nextDay?: Calculation; geometry: Geometry; route: Schema['RouteDetail'] };

/** Choose a supported scope per route, not from the global capability alone. */
export function supportedSelection(selection: Selection, b: Bootstrap): Selection {
  const canStop = b.capabilities.stop_forecasts && b.capabilities.forecast_scopes.includes('route_stop') &&
    selection.route_ids.every(id => b.active_snapshot.coverage.find(c => c.route_id === id)?.prediction_scopes.includes('route_stop'));
  return { ...selection, spatial_detail: canStop ? 'route_stop' : 'route' };
}

export function useDispatcher() {
  const [bootstrap, setBootstrap] = useState<Bootstrap>();
  const [confirmed, setConfirmed] = useState<Confirmed>();
  const [pending, setPending] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const requests = useRef(new LatestRequest());
  const bootstrapRequest = useRef(new LatestRequest());
  const attempt = useRef<{ selection: Selection; overrides?: Overrides } | undefined>(undefined);
  const confirmedRef = useRef<Confirmed | undefined>(undefined);
  const bootstrapRef = useRef<Bootstrap | undefined>(undefined);
  confirmedRef.current = confirmed;
  bootstrapRef.current = bootstrap;

  const calculate = useCallback(async (input: Selection, overrides?: Overrides, suppliedBootstrap?: Bootstrap) => {
    const b = suppliedBootstrap ?? bootstrapRef.current;
    if (!b) return;
    const selection = supportedSelection(input, b);
    const request = requests.current.next();
    attempt.current = { selection, overrides };
    setPending(true); setDirty(true); setError(''); setNotice('');
    try {
      validateSelection(selection, b);
      const routeId = selection.route_ids[0];
      const version = b.active_snapshot.provenance.network_version;
      const previous = confirmedRef.current;
      const reuse = previous?.geometry.route_id === routeId && previous.geometry.network_version === version;
      const [route, geometry] = reuse ? [previous.route, previous.geometry] : await Promise.all([
        api.route(routeId, version, request.signal), api.geometry(routeId, version, request.signal),
      ]);
      const stopCount = geometry.features.filter(f => f.properties.kind === 'stop').length;
      validateSelection(selection, b, stopCount);
      if (route.network_version !== version || geometry.network_version !== version || geometry.route_id !== routeId) {
        throw new Error('Geometry does not match the pinned route/network version.');
      }
      let nextSelection = nextDaySelection(selection);
      let boundaryNotice = '';
      if (nextSelection) {
        try { validateSelection(nextSelection, b, stopCount); }
        catch {
          nextSelection = undefined;
          boundaryNotice = '\u0421\u043b\u0435\u0434\u0443\u044e\u0449\u0438\u0439 \u0434\u0435\u043d\u044c \u0432\u043d\u0435 \u043f\u043e\u043a\u0440\u044b\u0442\u0438\u044f. \u041f\u043e\u043a\u0430\u0437\u0430\u043d\u044b \u0432\u0441\u0435 24 \u0447\u0430\u0441\u0430 \u0432\u044b\u0431\u0440\u0430\u043d\u043d\u043e\u0433\u043e \u0434\u043d\u044f.';
        }
      }
      if (overrides && !overridesForDay(selection, overrides) && !(nextSelection && overridesForDay(nextSelection, overrides))) {
        throw new Error('Scenario effective_window is outside the displayed period.');
      }
      const query = async (s: Selection) => {
        const o = overridesForDay(s, overrides);
        const result = await api.calculate(s, o, request.signal);
        assertCalculation(result, o ? { kind: 'scenario', selection: s, overrides: o } : { kind: 'forecast', selection: s }, geometry);
        return result;
      };
      // A covered next day is part of the service-day view: failures are explicit,
      // never silently replaced with synthetic or repeated observations.
      const [calculation, nextDay] = await Promise.all([
        query(selection), nextSelection ? query(nextSelection) : Promise.resolve(undefined),
      ]);
      if (request.isCurrent()) {
        setConfirmed({ calculation, nextDay, geometry, route });
        setDirty(false); setNotice(boundaryNotice);
      }
    } catch (e) {
      if (request.isCurrent()) setError(errorText(e));
    } finally {
      if (request.isCurrent()) setPending(false);
    }
  }, []);

  const initialize = useCallback(async () => {
    requests.current.cancel();
    const r = bootstrapRequest.current.next();
    setPending(true); setDirty(true); setError('');
    try {
      const b = await api.bootstrap(r.signal);
      if (!r.isCurrent()) return;
      bootstrapRef.current = b; setBootstrap(b);
      await calculate({ ...b.default_selection, route_ids: [b.default_selection.route_ids[0]] }, undefined, b);
    } catch (e) {
      if (r.isCurrent()) { setError(errorText(e)); setPending(false); }
    }
  }, [calculate]);

  useEffect(() => {
    void initialize();
    return () => { requests.current.cancel(); bootstrapRequest.current.cancel(); };
  }, [initialize]);

  return {
    bootstrap, confirmed, pending, pendingSelection: pending ? attempt.current?.selection : undefined,
    dirty, error, notice, calculate, initialize,
    markDirty: () => { requests.current.cancel(); setPending(false); setDirty(true); },
    retry: () => attempt.current ? calculate(attempt.current.selection, attempt.current.overrides) : initialize(),
    discard: () => { requests.current.cancel(); setPending(false); setDirty(false); setError(''); },
  };
}
