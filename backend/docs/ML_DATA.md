# ML / data contract v1.2

## 1. Подтверждённый конкурсный target

Из известных файлов:

```text
labels_day_train.csv: route;date;hour;boardings
labels_day_test.csv:  route;date;hour;boardings
test_submission.csv: route;date;hour;prediction
```

Поэтому route model target:

```text
route/date/hour -> boardings
```

Это официальный prediction scope для submission/WAPE.

## 2. Raw event dictionary

| Поле | Использование |
|---|---|
| `tran_no` | event id / dedup |
| `device_no` | validator device, не вагон |
| `tran_date_time` | основной event timestamp |
| `begin_date_time` | служебное |
| `input_date_time` | ненадёжное, не использовать |
| `crd_hashcode` | обезличенный card hash для внутренних агрегатов |
| `validation_result` | 1 = successful boarding |
| `tran_type_id` | transaction type |
| `place_id` | площадка/депо, не stop |
| `good_type` | ticket type |
| `pass_route` | noisy passenger route chain |
| `ngpt_route` | source of route id/name |
| `bus_exit_no` | vehicle schedule/output id |
| `garage_number` | tram physical/garage number |

## 3. Stop-level dataset

Stop forecasts становятся first-class scope, но **не выводятся из `place_id`**. ML-команда должна подключить отдельный stop-level source с минимумом:

```text
route_id
route_pattern_id or direction key
route_stop_id / stop mapping
observed timestamp
boardings target or defensible stop-level target
```

До этого stop capability выключена.

При публикации stop model должны существовать:

```text
stop_boardings.onnx (or documented equivalent artifact)
stop_feature_schema.json
stop_model_manifest.json
stop_golden_vectors.json
stop_quality.json
```

Stop target должен быть описан отдельно и не маскироваться под route label.

## 4. Route/stop model independence

Публичный API не фиксирует, одна это ML-модель или две. Runtime contract требует только:

```text
route prediction artifact/version
optional route_stop prediction artifact/version
```

Если одна модель поддерживает оба scope, `model_version` и `stop_model_version` могут ссылаться на один release id. Если модели разные — версии независимы.

## 5. Leakage

Forecast origin обязателен. Любой target-relative feature можно использовать только если информация была доступна к `observations_complete_through`.

Нельзя использовать:

- будущие boardings из target hour;
- `COUNT(DISTINCT garage_number)` из будущего target hour как известный supply;
- будущие labels/test targets;
- фактическую будущую погоду, если runtime получает только forecast weather.

## 6. Route features

Приоритет:

```text
route encoding
hour / weekday / calendar
historical profiles
lagged/rolling features available at origin
lead time
weather snapshot features
manual/event/season factors only where policy allows
```

Feature order и preprocessing — только из `feature_schema.json`.

## 7. Stop features

Будущая stop model может использовать:

```text
route + direction/pattern
route_stop_id / sequence
calendar/time
stop historical profile
route-level demand context
weather
other proven external factors
```

Но только признаки, реально доступные в forecast origin. Наличие geometry не создаёт passenger target само по себе.

## 8. Supply profile

Historical proxy:

```text
COUNT(DISTINCT garage_number)
by route/date/hour
for successful validations
```

Это proxy observed active vehicles, не гарантированно полный выпуск. Future supply берётся из versioned historical profile или manual plan.

## 9. Model artifact contract

Route release:

```text
route_boardings.onnx
route_model_manifest.json
route_feature_schema.json
route_golden_vectors.json
route_quality.json
training-config.json
```

Stop release аналогичен с `stop_` prefix.

Manifest должен содержать SHA-256, schema version, tensor contract, train cutoff, route coverage, lead range, preprocessing/postprocessing и library versions.

## 10. Quality

Route и stop quality измеряются отдельно:

```text
prediction_scope = route
unit = boardings_per_route_hour

prediction_scope = route_stop
unit = boardings_per_route_stop_hour
```

Competition WAPE относится к route scope. Stop score не смешивается с competition score.

Backtest должен быть temporal и воспроизводить forecast origin. Random shuffle не подходит для утверждения о future forecasting.

## 11. Weather

Open-Meteo features хранятся в versioned weather snapshot. Runtime user request не вызывает внешний API синхронно.

Для далёкого горизонта использовать только policy, на которой модель обучалась: forecast/climatology/missing mask. Не подменять unknown temperature нулём без schema rule.

## 12. Readiness checklist

Route:

- route mapping подтверждён;
- target aggregation совпадает labels;
- no future leakage;
- Python/ONNX/Go parity;
- WAPE measured;
- manifest/golden vectors готовы.

Stop:

- external stop source provenance описан;
- route_pattern/route_stop mapping устойчив;
- stop target определён;
- independent temporal quality measured;
- artifact/version published;
- snapshot capability switched only after validation.
