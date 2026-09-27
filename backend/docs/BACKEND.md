# Backend implementation guide v1.2

## 1. Source of truth

Публичный контракт — `openapi/openapi.yaml`. Runtime source of truth — PostgreSQL/PostGIS. ONNX/model artifacts immutable и загружаются по версии, закреплённой active snapshot.

## 2. Модули

```text
internal/auth
internal/catalog
internal/forecast
  FeatureBuilder
  RouteModelAdapter
  StopModelAdapter       # optional until stop model is published
  ModelRegistry
internal/scenario
internal/weather
internal/geometry
internal/indicators
internal/summary
internal/export
internal/snapshots
internal/storage/postgres
```

## 3. Forecast flow

```text
request
  ↓
validate auth + selection
  ↓
read immutable snapshot metadata from PostgreSQL
  ↓
validate coverage + max cells + spatial_detail
  ↓
read prepared runtime features
  ↓
route inference
  ↓
optional stop inference when spatial_detail=route_stop
  ↓
Scenario Engine
  ↓
aggregation
  ↓
indicators + visualization policy
  ↓
CalculationResponse
```

`spatial_detail=route_stop` разрешён только когда active snapshot реально содержит stop model/version и coverage. Иначе `422 STOP_FORECAST_UNAVAILABLE`.

## 4. Route vs stop models

Route model остаётся официальной моделью для `route/date/hour/boardings` и submission.

Stop model — отдельный artifact/model adapter. Backend не должен создавать stop predictions путём копирования route index. Stop response key:

```text
route_id + route_pattern_id + route_stop_id + frame.window
```

`route_stop_id` должен принадлежать выбранному route pattern и network_version snapshot.

Если stop model и route model используют разные feature schemas, обе версии pinned в snapshot. Если в будущем одна модель умеет оба scope, адаптер может ссылаться на один artifact, не меняя публичный API.

## 5. Scenario Engine

Scenario Engine — чистая детерминированная функция.

- fleet override не меняет predicted boardings;
- demand factors изменяют boardings согласно `CALCULATIONS.md`;
- при `route_stop` те же scenario semantics применяются к stop predictions;
- model weights не переобучаются на пользовательском запросе;
- сценарий всегда строится от baseline snapshot, не от предыдущего scenario response.

## 6. Временные режимы

Backend обязан валидировать:

```text
day    -> exactly 1 local day, resolution=hour
week   -> exactly 7 local days, resolution=day
month  -> exactly 1 local calendar month, resolution=day
custom -> local-midnight bounds, resolution=day only
```

`custom` duration <= `limits.max_custom_days` из bootstrap. Переходы Dec→Jan, Feb 29, 28/29/30/31 дней обрабатываются календарной библиотекой. Никаких silent clips.

Day endpoint возвращает все hourly frames. Frontend wheel не вызывает отдельный запрос на каждый час.

## 7. Stop map visualization

Backend отдаёт только **одно аналитическое значение на stop/frame** через `StopReading`.

Не создавать:

```text
20m -> 0.55
15m -> 0.63
10m -> 0.71
5m  -> 0.79
```

если таких данных нет. `VisualizationPolicy.stop_columns` сообщает, что frontend может визуально построить `ramp_to_stop` на длине около 20 м. Это rendering hint.

Geometry для stop берётся из `RouteGeometry`, join по `route_stop_id`.

## 8. IndicatorService

Backend определяет семантику индикатора и возвращает:

```text
key
variant
tone
icon
title
text
```

Совместимые варианты v1.2:

| key | variants |
|---|---|
| weather | `none`, `clear`, `cloudy`, `rain`, `snow`, `heat`, `cold`, `other`, `unknown` |
| fleet | `none`, `deficit`, `balanced`, `surplus`, `unknown` |
| trend | `none`, `up`, `down`, `flat`, `unknown` |
| peak | `none`, `peak`, `off_peak`, `unknown` |
| calendar | `none`, `weekday`, `weekend`, `holiday`, `unknown` |
| event | `none`, `active`, `inactive`, `unknown` |

Frontend не должен парсить текст. Backend не должен писать причинные утверждения, которых модель не доказывает.

## 9. PostgreSQL/PostGIS

Минимальные логические группы:

```text
network: routes, route_patterns, stops, route_stops
runtime: history aggregates, supply profiles, route/stop references
weather: weather_snapshots, weather_values
models: model_versions, feature_schema_versions
snapshots: forecast_snapshots, active_snapshot
security: users, sessions
quality: route/stop evaluation metadata
```

Hot cache может хранить decoded geometry, model sessions и повторяемые feature blocks. Cache miss не является 404.

## 10. Weather

Open-Meteo находится за `WeatherProvider`. User request не должен зависеть от синхронного внешнего HTTP-вызова.

```text
Open-Meteo -> refresher -> validate -> PostgreSQL weather snapshot -> active forecast snapshot
```

При отказе внешнего источника продолжает обслуживаться предыдущий валидный snapshot либо предусмотренный fallback.

## 11. Auth

```text
POST /auth/login   -> validate -> create DB session -> HttpOnly cookie
GET  /auth/session -> restore after refresh
POST /auth/logout  -> revoke DB session + expire cookie
```

Пароль/session token не хранить в `localStorage`.

## 12. Quality

Route quality и stop quality не смешиваются. `ValidationRecord.prediction_scope`:

```text
route
route_stop
```

Официальный competition WAPE относится к route/hour target, если организаторы не зададут другое. Stop-model score показывается отдельно.

## 13. Тесты P0

- day/week/month/custom boundary validation;
- custom cross-month/cross-year/leap-day;
- custom hourly request -> `422`;
- stop request when model unavailable -> `422`;
- route_stop_id belongs to route_pattern/network version;
- stop forecast is not copied route value in production adapter;
- scenario fleet changes load metrics but not boardings;
- indicator key/variant compatibility;
- snapshot atomic activation;
- cache miss rehydrates from PostgreSQL;
- calculation hash golden vector;
- CSV replay uses exact descriptor.
