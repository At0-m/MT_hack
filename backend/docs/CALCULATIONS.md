# Нормативные расчёты и идентичность результата v1.2

## 1. Атомарные прогнозы

Route scope:

```text
B_route,h = expected boardings for route/hour
```

Stop scope, когда опубликована stop model:

```text
B_stop,h = expected boardings for route_stop/hour
```

`boardings` — ожидаемое число посадок/валидаций, не число людей одновременно в салоне.

## 2. Исторический выпуск

Для истории:

```text
observed_vehicle_count =
  COUNT(DISTINCT garage_number)
  по route/date/hour
  среди validation_result = 1
```

Это наблюдаемая proxy, потому что вагон без успешной валидации может не попасть в счётчик. Для future используем versioned supply profile либо manual plan, но не будущие события test.csv.

Обозначения:

- `B_h` — route или stop boardings в часовом bucket;
- `N_h` — эффективное число вагонов маршрута;
- `R_h` — historical reference boardings-per-vehicle-hour для того же scope;
- `T` — target load index;
- `Δt_h = 1h`.

## 3. Load index

Для полного множества часов H одного scope:

```text
boardings                  = Σ B_h
vehicle_hours              = Σ N_h * Δt_h
mean_vehicle_count         = vehicle_hours / Σ Δt_h
boardings_per_vehicle_hour = Σ B_h / vehicle_hours
reference_boardings        = Σ N_h * R_h * Δt_h
load_index                 = Σ B_h / reference_boardings
```

Route и stop используют один тип формулы, но разные reference profiles. Stop `load_index` означает интенсивность посадок на конкретной остановке относительно её исторического эталона; это не cabin occupancy.

Не усреднять часовые ratios арифметически.

## 4. Missing / zero

| Ситуация | Результат |
|---|---|
| `B=0`, `N>0`, reference>0 | load index = 0 |
| `N=0` | `NO_SUPPLY` для supply ratios |
| неизвестный N | `SUPPLY_UNKNOWN` |
| reference отсутствует | `REFERENCE_MISSING` |
| reference=0 | `REFERENCE_ZERO` |
| неполный обязательный forecast window | запрос не выдаётся как полный |

`boardings` может быть доступен, когда supply metric недоступен.

## 5. Scenario Engine

Для каждого затронутого часа:

```text
B'_h = B_h * k_weather * k_event * k_season
```

Fleet:

```text
absolute -> N'_h = configured vehicle_count
delta    -> N'_h = N_h + delta
```

Fleet change не меняет `B_h`. Demand factors меняют `B_h`. Правило применяется и к route, и к stop forecast values.

Scenario всегда строится от baseline snapshot.

## 6. Required fleet

```text
N_required,h = ceil(B'_h / (T * R_h))
```

Для периода возвращается максимум hourly requirement. Для stop scope это локальный analytical constraint и не отдельная диспетчерская рекомендация.

## 7. Load levels

Backend возвращает `LoadLevel`; frontend не вычисляет собственные thresholds.

Политика v1.2:

```text
< 0.75       normal
[0.75,1)     elevated
[1,1.25)     high
>=1.25       very_high
```

Это продуктовая шкала относительно historical reference, не норматив вместимости.

## 8. Stop columns и decorative approach effect

Аналитическая сущность:

```text
StopReading.value = one real stop prediction/metric
```

Визуальная цепочка перед stop:

```text
▂ ▄ ▆ █ ●
```

не имеет отдельных backend values. Frontend применяет display profile `ramp_to_stop` к одному stop value. Не экспортировать декоративные sample points и не использовать их в summary/quality.

## 9. Indicators

`UiIndicator.variant` — структурированное семантическое состояние. Frontend не парсит text.

Backend не утверждает причинное влияние внешнего фактора без отдельного измеренного анализа. Формулировка «погодный snapshot содержит осадки» допустима; «дождь повысил спрос на 7%» требует отдельного доказательства.

## 10. Time modes

```text
day    -> 1 local day, hourly frames
week   -> 7 local days, daily frames
month  -> 1 local calendar month, daily frames
custom -> N local calendar days, daily frames only
```

Custom bounds — local midnight, timezone `Europe/Moscow`. Максимум берётся из `limits.max_custom_days`. Month/year transitions и leap day считаются календарной библиотекой.

Никакого silent clipping.

## 11. calculation_id v2

`calculation_id` — hash полного normalized descriptor, не persisted resource.

Canonical text:

```text
tramflow-calculation-v2
snapshot=<forecast_snapshot_id>
routes=<sorted ids>
view=<day|week|month|custom>
from=<UTC Z>
to=<UTC Z>
resolution=<hour|day>
spatial_detail=<route|route_stop>
kind=<forecast|scenario>
```

Для scenario добавляются:

```text
effective_from=<UTC Z>
effective_to=<UTC Z>
fleet=<unchanged|absolute:N|delta:N>
factor_weather_milli=<integer>
factor_event_milli=<integer>
factor_season_milli=<integer>
```

Все строки заканчиваются LF. `calculation_id = "calc_" + hex(SHA256(ASCII(text)))`.

Изменение `spatial_detail` меняет calculation ID, потому что route-only и route+stop response — разные воспроизводимые расчёты.

Golden vector — `examples/golden-canonical.json`.

## 12. Route/stop consistency

До отдельного reconciliation policy нельзя предполагать:

```text
Σ stop predictions == route prediction
```

Route model остаётся authoritative для official submission. Stop model имеет собственную quality evaluation. Если команда введёт reconciliation, его version должен быть pinned в snapshot и описан отдельным изменением контракта/политики.
