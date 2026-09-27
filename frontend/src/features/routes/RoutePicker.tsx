import { IncomingIcon } from '../../assets/IncomingIcon';
import { useEffect, useRef, useState } from 'react';
import { api, errorText } from '../../api/client';
import type { Bootstrap, Calculation, Selection, Schema } from '../../api/types';
import { validateSelection } from '../time/dates';
import { assertCalculation } from '../../state/calculation';
import { metric } from '../../ui/common';

export function RoutePicker({
  routeDetail,
  bootstrap,
  selection,
  index,
  onChoose,
}: {
  routeDetail: Schema['RouteDetail'];
  bootstrap: Bootstrap;
  selection: Selection;
  index: number;
  onChoose: (id: string) => void;
}) {
  const [search, setSearch] = useState('');
  const searchInput = useRef<HTMLInputElement>(null);
  const [active, setActive] = useState(0);
  const [comparison, setComparison] = useState<Calculation>();
  const [error, setError] = useState('');
  const [retry, setRetry] = useState(0);
  const [sort, setSort] = useState(true);
  const [details, setDetails] = useState<Record<string, Schema['RouteDetail']>>({ [routeDetail.route.route_id]: routeDetail });
  useEffect(() => {
    const controller = new AbortController();
    setDetails({ [routeDetail.route.route_id]: routeDetail });
    // Ten small immutable route descriptions, not ten forecast computations.
    void Promise.allSettled(bootstrap.routes.filter(r => r.route_id !== routeDetail.route.route_id).map(async r => {
      const detail = await api.route(r.route_id, bootstrap.active_snapshot.provenance.network_version, controller.signal);
      if (!controller.signal.aborted && detail.network_version === bootstrap.active_snapshot.provenance.network_version)
        setDetails(previous => ({ ...previous, [r.route_id]: detail }));
    }));
    return () => controller.abort();
  }, [bootstrap, routeDetail]);

  useEffect(() => {
    const ctrl = new AbortController();
    setComparison(undefined);
    setError('');
    const s = { ...selection, route_ids: bootstrap.routes.filter(r=>{try{validateSelection({...selection,route_ids:[r.route_id],spatial_detail:'route'},bootstrap);return true;}catch{return false;}}).map(r=>r.route_id), spatial_detail: 'route' as const };
    try {
      validateSelection(s, bootstrap);
      void api.calculate(s, undefined, ctrl.signal).then((c) => {
        assertCalculation(c, { kind: 'forecast', selection: s });
        if (!ctrl.signal.aborted) setComparison(c);
      }).catch((e) => {
        if (!ctrl.signal.aborted) setError(errorText(e));
      });
    } catch (e) {
      setError(errorText(e));
    }
    return () => ctrl.abort();
  }, [bootstrap, selection, retry]);

  const readings = comparison?.frames[index]?.routes;
  const routes = bootstrap.routes.filter((r) =>
    (r.name + ' ' + r.route_number + ' ' + r.route_id).toLocaleLowerCase('ru').includes(search.toLocaleLowerCase('ru'))
  );

  const selectedRouteId = selection.route_ids[0];
  routes.sort((a, b) => {
    if (a.route_id === selectedRouteId) return -1;
    if (b.route_id === selectedRouteId) return 1;
    const av = readings?.find((r) => r.route_id === a.route_id)?.evaluated.load_index.value;
    const bv = readings?.find((r) => r.route_id === b.route_id)?.evaluated.load_index.value;
    if (av == null) return bv == null ? 0 : 1;
    if (bv == null) return -1;
    return sort ? bv - av : av - bv;
  });

  return (
    <div className="route-picker">
      <div className="route-title">
        <h2>Маршруты</h2>
        <button className="sort-btn" onClick={() => setSort(!sort)} aria-label={sort ? 'Сортировка по убыванию' : 'Сортировка по возрастанию'}>
          <span>Загруженность</span>
          <IncomingIcon name={sort?"down.png":"up.png"} className="sort-placeholder"/>
        </button>
      </div>
      <div className="search-row" onClick={()=>searchInput.current?.focus()}>
        <input
          ref={searchInput}
          style={{width:`${Math.max(5,search.length)}ch`}}
          aria-label="Поиск маршрута"
          placeholder="Поиск"
          value={search}
          onChange={(e) => { setSearch(e.target.value); setActive(0); }}
          onKeyDown={(e) => {
            if (e.key === 'ArrowDown') { e.preventDefault(); setActive(Math.min(routes.length - 1, active + 1)); }
            if (e.key === 'ArrowUp') { e.preventDefault(); setActive(Math.max(0, active - 1)); }
            if (e.key === 'Enter' && routes[active]) onChoose(routes[active].route_id);
          }}
        />
        <IncomingIcon name="search.png" className="search-placeholder"/>
        {search && <button className="search-clear" aria-label="Очистить поиск" onClick={() => { setSearch(''); setActive(0); searchInput.current?.focus(); }}><IncomingIcon name="cross.png"/></button>}
        <img className="route-search-line" src="/assets/icons/40359.svg" alt="" />
      </div>
      {error && <p role="alert" className="route-error">{error}<button onClick={() => setRetry(retry + 1)}>Повторить сравнение</button></p>}
      <div className="route-options">
        {!routes.length && <p className="no-routes">Ничего не найдено</p>}
        {routes.map((r, i) => {
          const reading = readings?.find((v) => v.route_id === r.route_id);
          const stops = details[r.route_id]?.patterns[0]?.stops;
          const isSelected = selectedRouteId === r.route_id;
          const isActive = i === active;

          return (
            <button
              className={`route-option ${isActive ? 'active' : ''} ${isSelected ? 'selected' : ''}`}
              aria-pressed={isSelected}
              key={r.route_id}
              onClick={() => onChoose(r.route_id)}
              onMouseEnter={() => setActive(i)}
            >
              <span className="route-heading"><strong>Трамвай №{r.route_number}</strong>{reading?.indicators.some(item=>item.key==='trend'&&(item.variant==='up'||item.variant==='down'))&&<IncomingIcon name={reading.indicators.find(item=>item.key==='trend')?.variant==='up'?'up.png':'down.png'} className="route-trend-icon"/>}</span>
              <span className={isSelected ? 'route-chosen' : 'route-choose'}>
                {isSelected ? 'ВЫБРАН' : 'ВЫБРАТЬ'}
              </span>
              <img className="route-divider" src="/assets/icons/25557.svg" alt="" />
              <span className="route-details">
                <span title={stops?.[0]?.name}>отправление: {stops?.[0]?.name ?? 'нет данных'}</span>
                <span title={stops?.at(-1)?.name}>конечная: {stops?.at(-1)?.name ?? 'нет данных'}</span>
                <span>количество трамваев на маршруте: {metric(reading?.evaluated.mean_vehicle_count)}</span>
              </span>
            </button>
          );
        })}
      </div>
      <p className="sr-only">Сравнение исходных прогнозов за один кадр, без сценарных поправок.</p>
    </div>
  );
}
