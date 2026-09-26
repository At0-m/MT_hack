# TramFlow backend — Go, контракт v1.2

REST API на Go для маршрутного прогноза посадок: PostgreSQL/PostGIS, native ONNX CPU, сценарии выпуска/спроса, агрегация, GeoJSON и политику отображения остановок, UI indicators, summary, CSV, собственный login/session/logout.

Все 17 операций обслуживаются по `openapi/openapi.yaml`. PostgreSQL — источник истины для snapshots, inputs и сессий; память — ограниченный disposable cache. Publisher валидирует кандидат и атомарно активирует его через PostgreSQL advisory lock и проверку previous snapshot. API не обучает модель и не выдаёт географию за stop-level target.

## Запуск через Docker

Требуются Docker Compose и Go 1.26.8 для генерации fixture (или приложенный generated demo bundle).

```powershell
./scripts/dev.ps1
```

Скрипт создаёт `.env` с случайными паролями, запускает PostGIS, применяет миграции, создаёт пользователя, публикует synthetic demo только при пустом registry, собирает API. Адрес `http://localhost:8080`; логин и пароль в `.env`. При повторном запуске existing active snapshot сохраняется.

Локальный Docker Desktop на этой машине падает при инициализации `dockerInference`. Код контейнеров подготовлен; проверки PostgreSQL выполнены отдельно в WSL. Docker image/Compose end-to-end пока не подтверждены — это не результат успешной Docker-сборки.

## Разработка без контейнера

Нужен PostgreSQL 16 с PostGIS. Задайте `DATABASE_URL` для Publisher (write role), `API_PASSWORD`, затем:

```powershell
go run ./cmd/fixtures --onnx
go run ./cmd/publisher --migrate --create-user dispatcher
go run ./cmd/publisher --bundle testdata/demo-bundle.json --allow-synthetic
$env:COOKIE_SECURE = 'false'
$env:CORS_ORIGIN = 'http://localhost:5173'
go run ./cmd/api
```

`go run` без build tag поддерживает synthetic demo; real ONNX требует CGO и `go build -tags onnx`, `ONNX_LIBRARY_PATH` к native Runtime 1.24.1. Docker собирает именно native вариант. Compiler wrapper и native version зафиксированы в go.mod/Dockerfile. Для API используйте отдельную роль: SELECT runtime таблиц, SELECT/INSERT/DELETE в `user_sessions`, SELECT/INSERT/UPDATE/DELETE в `login_limits`; миграции/Publisher — другая роль. Пример выдачи прав в scripts/db-init.sh и migrations 002–005. После обновления сначала примените `go run ./cmd/publisher --migrate` с write-role.

Production: TLS gateway, `COOKIE_SECURE=true` (default), точный `CORS_ORIGIN`, credentials только у Publisher при provisioning. Cookie HttpOnly/SameSite=Strict, token хранится в БД как SHA-256; пароль — bcrypt. Login ограничен 60 попытками/мин/IP на реплику, 10/мин/account в общей БД и двумя конкурентными bcrypt. За gateway задайте его точные CIDRs в `TRUSTED_PROXY_CIDRS` и общий gateway limiter. Только доверенный peer разрешает разбор X-Forwarded-For; при отсутствующей корректной цепочке используется account limiter. POST с чужим Origin или Sec-Fetch-Site=cross-site отклоняется. Для same-origin frontend используйте reverse proxy; для localhost dev CORS явно разрешён.

## Фронтенду

1. POST `/api/v1/auth/login` с `username/password`, затем запросы с `credentials: 'include'`.
2. GET `/api/v1/auth/session` при refresh; POST `/api/v1/auth/logout` завершает сессию в БД.
3. GET `/api/v1/bootstrap`: default_selection, версия snapshot/network и limits. Не используйте даты demo как реальные.
4. POST `/api/v1/forecasts/query` или `/scenarios/evaluate`: отправляйте pinned selection; получайте весь массив frames для локального переключения времени.
5. Summary и экспорт: полный descriptor и `expected_calculation_id`. Несовпадение возвращает 409.

Периоды строго календарные, границы — московские полуночи, ответы — UTC, интервал `[from,to)`:

- `day`: ровно один день, `resolution=hour`, все 24 кадра.
- `week`: ровно 7 дней, `resolution=day`.
- `month`: от первого числа до первого числа следующего месяца, `resolution=day`.
- `custom`: произвольный период до `limits.max_custom_days` (сейчас 31), только `resolution=day`. Hourly custom возвращает 422.

В selection обязательно `spatial_detail=route` или `route_stop`. Сейчас модель остановок отсутствует: bootstrap возвращает `stop_forecasts=false`, coverage — `prediction_scopes=["route"]`, `stop_forecast_status=unavailable`. Запрос `route_stop` возвращает `422 STOP_FORECAST_UNAVAILABLE`; в маршрутном ответе `stop_readings=[]`.

`VisualizationPolicy.stop_columns` содержит rendering hint: привязка к stop_position, декоративный ramp около 20 м. Backend не рассчитывает аналитические значения вдоль этого участка и не копирует маршрутный индекс в stop predictions. GeoJSON содержит только географию; join остановок по route_stop_id. Индикаторы имеют структурированные key/variant/tone/icon, summary route/overall использует те же метрики. Focus route_stop пока unavailable.

Идентификатор расчёта использует canonical v2 с spatial_detail. Export/summary проверяют полный descriptor и expected_calculation_id, поэтому идентификаторы v1 не подходят. Запросы не обрезаются: максимум 10 маршрутов, 7440 исходных часовых ячеек и 960 возвращаемых route/frame cells, точное покрытие берётся из snapshot.

В исходной v1.2 schema snapshot.view_modes ограничен тремя пунктами; demo публикует day/week/month, custom доступен через capabilities.custom_period.

CSV: UTF-8 BOM, semicolon, CRLF, null — пустая ячейка; точные колонки в docs/CSV.md. В исходной поставке CSV.md отсутствует, поэтому здесь зафиксирована собственная таблица колонок. CSV пользователя не является competition submission.

## Проверки и производительность

```powershell
go run ./cmd/fixtures --onnx
go test ./cmd/... ./internal/... ./migrations
go vet ./cmd/... ./internal/... ./migrations
docker compose --profile test run --rm tests
go run ./cmd/loadtest -url http://localhost:8080 -n 3000 -c 16
```

Integration suite требует `TEST_DATABASE_URL` администратора **тестового** PostgreSQL: создаёт отдельную случайную БД, проверяет и удаляет только её. Нет DB env — тест явно skipped. Native tests требуют build tag onnx и DLL/SO. Смотрите [обновлённый отчёт](docs/UPDATED_REVIEW_FIXES.md) и [предельный замер](docs/boundary-performance.json): native/race с PostGIS пройдены; max-size HTTP проверен под cgroup 2 GiB / 2 CPU на синтетической модели. Старые docs/TEST_REPORT.md и docs/performance.json сохраняют исторические результаты. Реальную ML-нагрузку и Docker end-to-end ещё нужно проверить.

Cache budget 64 MiB с консервативным 512-byte accounting на entry, максимум два native inference, 32 пользовательских HTTP requests, DB pool 8. Полные признаки загружаются только для cache misses, по 128 строк. Бюджет параллельных расчётов — 32 единицы на процесс и 30 на пользователя, до 256 часовых ячеек на единицу; превышение даёт 429. GET health/live обходит этот лимит, ready имеет два собственных слота и timeout 2 с. Model sessions максимум 16, с LRU-вытеснением свободных сессий; native RSS не равен Go heap. Context cancellation запрашивает ONNX RunOptions.Terminate; освобождение буферов ждёт завершения Run. Параметры конфигурации: DATABASE_URL, DB_MAX_CONNECTIONS (1..32), ARTIFACTS_ROOT, OPENAPI_PATH, HTTP_ADDR, CORS_ORIGIN, COOKIE_SECURE, TRUSTED_PROXY_CIDRS, ONNX_LIBRARY_PATH.

## ML-поставка и границы готовности

Настоящей обученной модели и prepared transport data пока нет. Demo синтетический, WAPE и эффекты внешних источников не измерены. Подключение ML описано в docs/ML_HANDOFF.md. Backend готовит runtime boundaries, но не заменяет работу ML/ETL-команды: импортирует versioned prepared inputs, проверяет hashes/golden vectors и запускает ONNX. Raw ingestion, weather downloading, обучение и backtest — offline-поставка.

Retention сохраняет старые inputs/models; Publisher `--expire` создаёт tombstones для 410 после declared retention. `--expire --gc-data` явно удаляет большие DB profiles, не используемые retained snapshots, с минутным grace period. Tombstones/manifests/version claims и artifact files сохраняются; файловый GC ещё нужен. Отказ PostgreSQL даёт 503 без файлового fallback и логирует private cause. На новую модель требуется повторить реальные WAPE/parity и container load/RSS/CPU checks. Stop pipeline и автоматический Open-Meteo refresher остаются открытыми компонентами.

## Структура кода

`cmd/api` запускает сервер; `cmd/publisher` проверяет и публикует offline bundle; `cmd/fixtures` создаёт demo; `cmd/loadtest` измеряет полный HTTP endpoint.

`internal/httpapi` содержит отдельные обработчики bootstrap/catalog/forecast/weather/export/session, middleware и проверку JSON. `internal/engine` — календарные ограничения, canonical hash, расчёт метрик, сценарии, индикаторы и summary. `internal/inference` управляет ONNX sessions; `internal/storage` — SQL, миграции, публикация и сессии; `internal/domain` — типизированный публичный JSON и prepared bundle.

Документы ARCHITECTURE/BACKEND/CALCULATIONS/FRONTEND/ML_DATA и OpenAPI скопированы из последнего архива `MT_HACK_ALL_ROADS_LEAD_TO_FINALE.zip`. Инструкция запуска и отчёт проверок описывают фактическую реализацию.

Исправления BACKEND_REVIEW.md, изменения формата bundle и оставшиеся интеграционные задачи описаны в [docs/REVIEW_FIXES.md](docs/REVIEW_FIXES.md). При обновлении обязательны миграция 003 и weather_metadata; порядок перехода — в [docs/ML_HANDOFF.md](docs/ML_HANDOFF.md).
