# k6: реальные контракты TramFlow

Запускать через `make smoke`, `make benchmark` или `scripts/benchmark.py` из корня. k6 работает в отдельном контейнере; JS не выполняется Node как HTTP-тест. Node используется только для чистых helper-функций в `tests/`.

## Файлы

`smoke.js` проверяет полный короткий маршрут: live/ready, unauthenticated session, login/session, bootstrap, routes/geometry, прогноз, scenario, CSV, summary и ошибки 400/401/409/422. `forecast-day.js`, `forecast-week.js`, `forecast-month.js`, `scenario.js`, `bootstrap.js`, `route-geometry.js`, `export.js`, `summary.js` используют общий runner. `mixed-workload.js` задаёт детерминированную пользовательскую смесь. `login.js` — отдельный медленный closed-loop auth test.

## Авторизация

Один login в setup создаёт `tramflow_session`. Cookie явно передаётся из setup в каждый запрос каждого VU. Сессии не создаются на каждый VU: сервер хранит максимум 20 сессий пользователя и ограничивает login до bcrypt. Cleanup выполняет logout.

Это тест **одного авторизованного пользователя** с сохранёнными per-user limits. Для справедливого измерения многопользовательского максимума нужен отдельный пул предварительно созданных учётных записей; здесь такой профиль не подменяется отключением защиты.

Ни session token, ни password, ни setup_data не попадают в handleSummary. Сырые k6 series используют статические name/endpoint tags, без URL/request_id/snapshot_id как labels. Только ожидаемые отрицательные ответы smoke исключаются из http_req_failed через per-request responseCallback.

## Forecast

Selection строится по bootstrap/coverage, а не по захардкоженной «сегодняшней дате». Для synthetic stress fixture пример запроса:

```json
{
  "selection": {
    "forecast_snapshot_id": "stress-october-2026-v1",
    "route_ids": ["demo-01"],
    "view_mode": "day",
    "window": {
      "from": "2026-09-30T21:00:00Z",
      "to": "2026-10-01T21:00:00Z"
    },
    "resolution": "hour",
    "spatial_detail": "route"
  }
}
```

Границы — московские полуночи в UTC-представлении, интервал `[from,to)`. День даёт 24 hourly frames. Week — 7 дней с day-resolution. Month — первое число до первого числа следующего месяца, включая 29/30/31 день; это не duration=30d.

Проверяются calculation ID `calc_` + 64 hex, snapshot, число/границы кадров, полный набор route IDs и конечные неотрицательные baseline/evaluated boardings. В smoke дополнительно проверяется эффект scenario event=1.1.

## Scenario

POST `/api/v1/scenarios/evaluate` использует тот же selection и:

```json
{
  "overrides": {
    "effective_window": {"from": "2026-09-30T21:00:00Z", "to": "2026-10-01T21:00:00Z"},
    "factors": {"event": 1.1}
  }
}
```

Нельзя заменять effective_window на произвольные start/end или отправлять только factor: тест должен соответствовать приложенному OpenAPI.

## Summary / CSV — replay, а не GET по ID

Из успешного forecast/scenario берутся `descriptor` и `calculation_id`. POST `/api/v1/exports` получает:

```javascript
{
  calculation: forecast.descriptor,
  expected_calculation_id: forecast.calculation_id,
  format: 'csv'
}
```

Для POST `/api/v1/summaries/query` вместо format используется `focus: {kind: 'overall'}`. Export проверяет content-type и X-Calculation-ID; smoke дополнительно считает CSV-строки. Неправильный expected_calculation_id должен давать 409.

Подготовка descriptor выполняется вне measurement. Поэтому export-итерация содержит один HTTP-запрос. Серверное вычисление replay остаётся частью измеряемого export endpoint.

## Geometry

Запрос `/api/v1/routes/<route>/geometry` обязательно содержит `network_version` из active snapshot provenance. Список маршрутов также pinned к network_version. Модель stop-level в исходной поставке отсутствует; GeoJSON не является числовым прогнозом по остановкам.

## Смешанный профиль

За каждые 100 business requests: day 60; bootstrap/routes/geometry по 5; scenario 15; week+month 5; summary+export 5. Перестановка распределяет типы по времени без длинных последовательных пачек. Одна итерация — один business request, включая export; заданные доли поэтому соответствуют долям RPS, а не VU.

## Параметры

| Env | По умолчанию | Смысл |
|---|---|---|
| `RATE` | 50 | offered business requests/s |
| `WARMUP` / `DURATION` | 2m / 5m | фазы constant-arrival-rate |
| `ROUTE_COUNT` | 1 | число маршрутов, максимум 10 |
| `ROUTE_IDS` | не задано | явный список, имеет приоритет |
| `DAY_OFFSET` | 0 | смещение day/week внутри coverage |
| `PRE_VUS` | max(20, ceil(RATE×0.5)) | заранее выделенные VU |
| `MAX_VUS` | max(100, RATE×2) | верхняя граница VU |
| `P95_MS` | 250 | business HTTP p95 threshold |
| `FORECAST_P95_MS` | 200 | day HTTP p95 threshold |
| `WALL_P95_MS` | 300 | client wall p95 threshold |
| `REQUEST_TIMEOUT` | 12s | deadline HTTP, не увеличивать выше drain без изменения runner |
| `EXPECTED_RUNTIME_MODE` | не задано | optional exact provenance assertion |
| `EXPECTED_MODEL_VERSION` | не задано | optional exact model assertion |

Перед ручным `k6 run` смонтируйте `/out` на запись: handleSummary пишет туда JSON. Штатный Python runner делает это автоматически, ограничивает ресурсы генератора и сохраняет исходный exit status.

Offline tests:

```bash
node --test loadtest/tests/*.test.mjs
```

Эти проверки проверяют календарь, coverage, mixed shares и response-validator. Они не означают, что k6 scripts уже были выполнены против контейнерного API.
