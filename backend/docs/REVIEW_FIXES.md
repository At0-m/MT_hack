# Исправления BACKEND_REVIEW.md — 26.09.2026

Ревью использовано как список замечаний. Исходный OpenAPI v1.2 не менялся.

| Пункт | Результат |
|---|---|
| R1 | Почасовая проверка выпуска до агрегации; среднее N не сравнивается с максимальным required. |
| R2 | Добавлены парные stop metadata; Enrich сохраняет готовые stop_readings. Полный stop pipeline остаётся отдельной незавершённой функцией. |
| R3 | Отдельная immutable weather_snapshots, реальные поля из bundle, проверка SourceList и общий location для маршрутов. |
| R4 | Общий DefaultSelection используется Publisher, readiness и bootstrap; непригодный кандидат не активируется. |
| R5 | Индикаторы учитывают frame.window/effective_window, сохраняют погодную категорию и объясняют частичное влияние на totals. |
| R6 | LRU до 16 native sessions, ссылки на выполняющиеся/ожидающие запросы, безопасное повторное открытие старой версии. |
| R7 | Добавлены trend, WMO-категории и календарные варианты с явными правилами ниже. |
| R8 | Offline-загрузчик Open-Meteo/refresh и реальный сквозной погодный bundle пока не поставлены; Python CSV reader не объявляется refresher. |
| R9 | Сверка detail/geometry, принадлежности направлений, глобальных occurrence IDs и общих физических остановок; конфликт stops отклоняется, добавлен составной FK. |

## Правила индикаторов

- При неизвестном N/R хотя бы одного часа fleet=unknown. При полных данных deficit означает недостаток хотя бы в одном часу; surplus — хотя бы один час с запасом при отсутствии дефицита; balanced — точное соответствие каждый час. Нулевые N/спрос учитываются отдельно. Максимальный дефицит и количество часов хранятся только внутри расчёта. Публичный required остаётся максимумом часовой потребности.
- Trend: relative_to_typical >5% — up, <-5% — down, включительно ±5% — flat. Недоступный типовой профиль — none.
- Weather использует WMO weather_code каждого часа: 0 clear; 1–3 cloudy; дождь/морось/ливни rain; снег snow; туман/гроза other. При кодах 0–3 температура ≥30°C даёт heat, ≤−15°C — cold. Неизвестные коды и смешанные категории периода — unknown. Категория не выводится из средней температуры или суммы осадков. Missing — none. Это правила отображения, не оценки причин спроса.
- Calendar использует закреплённые features.is_holiday и features.is_weekend, если ETL их передал; иначе суббота/воскресенье определяются в Москве. Праздник без подтверждённого флага не придумывается. Смешанный календарный период — unknown. Корректность официальных переносов в flags подтверждает ETL.
- Обычно приходят 5 индикаторов: weather/fleet/trend/peak/calendar. При действующем ручном event-коэффициенте event=active занимает место peak, сохраняя контрактный максимум 5. Season дописывается в calendar, weather — в weather; исходные категории сохраняются. За effective_window поправка не показывается.

## Проверки

Полный набор `go test ./cmd/... ./internal/... ./migrations` и `go vet` прошли на Windows. В Ubuntu WSL2 прошёл полный набор с `-race` и настоящим PostgreSQL 16/PostGIS: миграции 1–3, CAS, сохранение active при сдвинутом coverage/no-day/no-hour, запрещённая связь чужого route/pattern и неизменность weather metadata при повторном использовании.

HTTP-тест проверяет одинаковые provider/source/location/retrieved_at двух прогнозных снимков и двух маршрутов. Все ответы проверяются по исходному OpenAPI. Добавлены регрессии ложного surplus/deficit, unknown/zero supply, частичного сценария, WMO/trend/calendar, конфликтующей географии и погодных ссылок.

Native ONNX Runtime 1.24.1 на Windows: прежние golden/concurrent tests плюс 96 версий от 4 конкурентных workers, ограничение 16 и повторное открытие исходной версии через сохранённый handle. Fake blocking predictor отдельно проверяет запрет закрытия во время Run; этот тест прошёл с WSL race detector. Native Windows `-race` не собрался: Zig linker не разрешил WaitOnAddress/WakeByAddress*. Native Go race поэтому не заявляется; native concurrency проверена обычным ONNX запуском.

Ранее опубликованный короткий synthetic HTTP load test не повторён и не является замером исправленной версии. Docker build/Compose с CPU/RAM quotas и минимальной API ролью остаётся непроверенным: Docker Desktop ранее падал при инициализации dockerInference.

## Оставшаяся интеграция

Stop pipeline требует собственного model manifest, inputs/reference/typical, хранения, inference, расчётов stop_readings, stop-summary/export и интеграционного теста двух разных остановок. Пока capabilities.stop_forecasts=false, route_stop запросы и публикации недоступны. Парные поля metadata этого не изменяют.

Для погоды нужен владелец offline-загрузчика: получать provider run/доступность/время получения и location mapping, собирать immutable weather_metadata и SourceList, подготовить признаки, затем ValidateBundle/Publish с новым snapshot ID и expected_previous_snapshot. Реальный исторический CSV сам по себе не доказывает operational forecast availability. Сквозная проверка реальной ML/погодной поставки остаётся незавершённой до такой поставки.

Долг исходного контракта view_modes maxItems=3 и согласование CSV.md остаются у команды; в рамках исправлений спецификация не переписывалась.

## Обновление

Миграцию 003 применяет Publisher `--migrate`. Новая публикация требует weather_metadata; старый weather endpoint без metadata отдаёт 503 вместо выдуманного provenance. Для старой поставки schema 2 нужен новый weather ID и честные source metadata; подробности в ML_HANDOFF.md. Перегенерируйте локальный synthetic bundle командой `go run ./cmd/fixtures --onnx`. Миграция не меняет active pointer; несовместимые старые связи route/pattern нужно исправить в источнике до миграции.
