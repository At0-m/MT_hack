# Исправления BACKEND_REVIEW_UPDATED.md

Проверено 2026-09-26. Документ ревью использован как список проверяемых замечаний; изменения контракта и внешние публикации из него автоматически не выполнялись.

## Исправления

| Замечание | Результат |
| --- | --- |
| P1-1: память и дорогие запросы | Чтение лёгких данных отдельно от feature JSON; признаки читаются только для cache misses, порциями до 128 строк. Бюджет расчётов: 32 единицы на процесс, 30 на пользователя; одна единица — до 256 часовых ячеек. Максимальный запрос занимает 30 единиц. Перегрузка возвращает 429 с Retry-After. |
| P1-2: health под нагрузкой | GET live обходит пользовательский semaphore; ready использует свои два слота и timeout 2 с. |
| P1-3: login за proxy | Явные TRUSTED_PROXY_CIDRS; forwarded chain читается только от доверенного peer. На IP — 60 попыток/мин на реплику; на account hash — 10/мин в общей PostgreSQL, включая неизвестные аккаунты. Без корректного X-Forwarded-For от доверенного proxy остаётся account limiter. |
| P1-4: Go/dependencies | Go 1.26.8, kin-openapi 0.149.0, pgx 5.11.0, x/crypto 0.57.0; повторные тесты и govulncheck. |
| P1-5: schema provenance | Новая immutable claim содержит version, SHA-256 целого schema artifact и columns. Дополнительно проверяются старые model manifests с этим schema ID. Семантическое изменение требует нового ID. |
| P2-1: максимум summary | Максимальный индекс берётся из выбранных frames/routes; boardings total остаётся суммой за период. Регрессии для route/overall. |
| P2-2: absolute proxy | Absolute fleet всегда proxy=false; delta сохраняет proxy baseline. |
| P2-3: sessions | При создании сессии удаляется до 512 просроченных записей; не более 20 сессий на пользователя, конкурентное создание сериализовано. Maintenance также удаляет просроченные сессии. |
| P2-4: Publisher round trips | Четыре INSERT на час отправляются pgx.Batch порциями по 128 часов. Для 7440 часов — 58 batches. CAS/активация остаются в одной атомарной транзакции. |
| P2-5: native cancellation | Context cancellation вызывает ONNX RunOptions.Terminate; буферы/session освобождаются после завершения Run. Это cooperative cancellation, не гарантированный hard kill неисправного native kernel. |
| P2-6: ошибки БД | Private cause сохраняется и логируется с request ID/operation; клиент получает безопасный код. Отсутствие active snapshot отличается от отказа БД. Паники логируются со stack. |
| P2-7: ORT download | Linux архив 1.24.1 проверяется по фиксированному SHA-256 перед распаковкой в Docker и CI. |
| P2-8: GC | Реализована явная reference-aware уборка больших DB profiles/погоды/географии. Общие версии сохраняются, tombstones и manifests остаются; после tombstone действует минутный grace period. Файловый artifact GC ещё отсутствует. |
| P2-9: fleet provenance | unavailable требует fleet=null/proxy=false; manual_plan — известный fleet/proxy=false; observed_vehicle_profile — известный fleet/proxy=true. |
| P2-10: cutoff | Нулевые origin/complete-through/train-cutoff отклоняются; train_cutoff <= complete_through <= origin. |

## Проверки

Пройдены обычные тесты и vet на Windows, native ONNX тесты на Windows, полный Linux native ONNX + `-race` с настоящим PostgreSQL 16/PostGIS в WSL2. В тестах проверяются shared login limiter, session cap, CAS/immutable schema, GC общих версий, health при заполненных user slots, summary/proxy, холодное чтение признаков 128/40 и отсутствие полного чтения признаков при hot cache. Тест cancellation проверяет вызов terminate и ожидание завершения; патологический native kernel отдельно не моделировался.

`govulncheck@v1.8.0`: 0 уязвимостей вызываемых символов, 0 уязвимостей импортируемых пакетов. Есть module-level предупреждение GO-2026-5932 об unmaintained `x/crypto/openpgp`; этот пакет приложение не импортирует, исправленной версии advisory не указывает. Это не утверждение об отсутствии любых уязвимостей во всех зависимостях.

Предельный HTTP-тест: 10 маршрутов, 744 часа, 256 признаков; настоящая native MatMul с явно синтетическими данными. PostgreSQL использует отдельную API-роль без Publisher permissions. Процесс API ограничен cgroup **2 GiB / 2 CPU** в WSL2; это не Docker image test и не замер обученной модели.

| Метрика | Замер |
| --- | --- |
| Publisher | 4,099 с |
| Cold forecast | 4,5296 с, HTTP 200 |
| Hot forecast | 0,0648 с, HTTP 200 |
| Partial scenario | 0,0579 с, HTTP 200 |
| API cgroup memory peak | 64 393 216 bytes, около 61,4 MiB |
| OOM / OOM kill | 0 / 0 |
| Одновременные холодные max requests | 1 принят, 31 получил ожидаемый 429 |
| Live / ready при перегрузке | по 189 запросов, 0 ошибок |

Сырые результаты: [boundary-performance.json](boundary-performance.json). Генерация fixture: `go run ./cmd/fixtures --onnx --stress`; создаёт ignored stress bundle/model и отдельный synthetic artifact. Для публикации нужен native Publisher с `--allow-synthetic`; stress fixture предназначен для изолированной тестовой БД.

В корне репозитория подготовлен `.github/workflows/backend.yml`: PostgreSQL/PostGIS, verified ORT, native/race, vet, govulncheck, Docker runtime build. Workflow ещё не запускался в GitHub. Локальный Docker Desktop остаётся неисправен; успешную сборку Docker/Compose не заявляем.

## Обновление

Перед запуском новой API-версии применить миграции **004 и 005** через Publisher write-role: `go run ./cmd/publisher --migrate`. Migration 004 создаёт общий login limiter и выдаёт права стандартной роли tramflow_api; custom API-роли нужны SELECT/INSERT/UPDATE/DELETE на login_limits, SELECT остальных runtime таблиц и SELECT/INSERT/DELETE на user_sessions. Migration 005 выносит calendar flags для лёгкого чтения и backfill существующих features.

За TLS gateway настроить TRUSTED_PROXY_CIDRS только для фактических адресов proxy; gateway должен формировать корректный X-Forwarded-For и ограничивать login traffic. Секреты остаются в environment, не в Git.

Maintenance выполняется отдельно с Publisher ролью: `go run ./cmd/publisher --expire --gc-data`. Команда удаляет данные после retention; на рабочей базе автоматически не запускалась. Artifact files нужно пока сохранять. Для включения cron/расписания требуется выбрать эксплуатационное окружение.

## Открытые границы

- **BLOCKER-1 остаётся:** stop feature storage/adapter/model, агрегация, summary/CSV и integration test разных остановок ещё нужны. Route-only API честно возвращает STOP_FORECAST_UNAVAILABLE. Одной передачи stop ONNX недостаточно.
- **BLOCKER-2 остаётся:** автоматического Open-Meteo fetcher/refresher нет. Publisher принимает подготовленную погоду; нужен отдельный ETL-компонент с provider run/availability metadata и публикацией snapshots.
- **P2-8 частично закрыт:** DB profiles очищаются; manifests/version claims и файловые ONNX artifacts сохраняются. Без координации реплик и lease кандидатов автоматическое удаление artifact files не включалось.
- Нет реальных WAPE/backtest и нагрузки обученной модели. Cooperative ONNX termination не заменяет process watchdog при зависшем native kernel. Docker image/Compose и новый CI должны быть проверены после восстановления Docker/пуша.

Исправленный route backend проверен в описанных условиях; весь продукт production-ready пока не объявляется.
