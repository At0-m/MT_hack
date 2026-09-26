# Frontend implementation guide v1.2

## 1. Source of truth

Типы и endpoint'ы — только из `openapi/openapi.yaml`. Не добавлять поля/endpoint'ы локально без изменения контракта.

## 2. App state

```text
forecastSnapshotId
routeId
routePatternId?
viewMode: day | week | month | custom
window
spatialDetail: route | route_stop
selectedFrameIndex
scenarioOverrides
selectedStopId?
```

## 3. Time UX

| Mode | Backend resolution | Frontend interaction |
|---|---|---|
| Day | hour | получает все hourly frames; wheel меняет локальный index |
| Week | day | 7 daily frames |
| Month | day | 28/29/30/31 daily frames |
| Custom | day only | произвольный диапазон дат |

`custom` никогда не запрашивается с `resolution=hour`. Для почасового просмотра выбранного дня переключиться в `day`.

Custom:

- границы выбираются календарными датами;
- может пересекать месяц/год и 29 февраля;
- максимум читается из `bootstrap.limits.max_custom_days`;
- frontend не обрезает range сам и не принимает partial response как полный;
- `422` показывает пользователю доступный coverage/limit из problem details, если backend их вернул.

## 4. Карта

Geometry загружается отдельно от forecast values:

```text
RouteGeometry
  route_pattern line
  route_stop points

CalculationResponse
  frames[].routes[].stop_readings[]
```

Join только по `route_stop_id`.

### Реальная остановочная метрика

При `spatial_detail=route_stop` `stop_readings[]` — реальные stop-level predictions. Не подменять их route-level значениями.

Если `capabilities.stop_forecasts=false`, UI скрывает/disable stop analytics и не делает `route_stop` запрос.

### Декоративные столбики перед остановкой

На одну остановку приходит одно значение. Frontend может отрисовать декоративный ramp/chain перед stop согласно:

```text
visualization.stop_columns.approach_effect = decorative
profile = ramp_to_stop
approach_length_m ≈ 20
```

Несколько визуальных сегментов не являются несколькими измерениями. Tooltip показывает один `StopReading`.

Высота строится по `stop_reading.evaluated.load_index` и общей `VisualizationPolicy`; цвет — по `load_level`/design token. Исходное число не clamp'ится.

## 5. Indicators / moods

Backend возвращает:

```text
key
variant
tone
icon
title
text
```

Иконка выбирается по структурированным полям, не по тексту.

Примеры:

```text
weather + rain
weather + clear
trend + up
trend + down
fleet + deficit
fleet + balanced
```

`variant=none` + `icon=none` означает нейтральное состояние без иконки.

Frontend не делает выводы вроде «дождь повысил спрос на 7%», если backend не вернул такое отдельное измеренное утверждение.

## 6. Summary

`POST /api/v1/summaries/query` получает полный calculation descriptor + `expected_calculation_id`.

UI states:

```text
idle
loading
ready(text)
unavailable(reason)
error(problem)
```

Summary для stop focus использует настоящий `StopReading`, когда stop forecast доступен.

## 7. Scenario

Fleet change:

```text
boardings baseline stays unchanged
load metrics recompute
```

Manual demand factors могут изменить boardings. После scenario response frontend полностью заменяет отображаемые evaluated metrics этим response, не смешивает old/new frame data.

## 8. Auth

Startup:

```text
GET /auth/session
  200 -> app
  401 -> login form
```

Login через `POST /auth/login`. Browser должен отправлять same-origin cookies (`credentials: include`, если требуется fetch-конфигурацией). Session token не читать и не хранить в JS.

Logout -> `POST /auth/logout` -> clear local app state -> login screen.

## 9. Export

Экспорт передаёт тот же `CalculationDescriptor` и `expected_calculation_id`. Frontend не собирает CSV сам из видимых карточек.

## 10. Ошибки, которые нельзя скрывать

- stop forecast unavailable;
- window outside snapshot coverage;
- custom range too long;
- custom hourly request invalid;
- snapshot expired/deactivated beyond retention;
- calculation mismatch;
- unavailable metric reason (`NO_SUPPLY`, `REFERENCE_MISSING` и т.п.).

## 11. Что нельзя делать

- HTTP request на каждый шаг day wheel;
- парсить indicator text для иконки;
- генерировать fictitious stop/segment values ради анимации;
- называть `load_index` физической заполненностью салона;
- silently clip custom range;
- считать, что сумма stop predictions обязана равняться route prediction, если backend явно не сообщил reconciliation policy.
