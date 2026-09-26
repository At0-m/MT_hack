# Backend: инструкция реализации

## 1. Стек и границы

Базовая реализация — Go, `net/http`, ONNX Runtime CPU через изолированный адаптер C API. **PostgreSQL + PostGIS — обязательная runtime dependency и source of truth.** Обучение — Python. Отдельный HTTP inference-сервис, Redis, Kafka или собственная система JWT не требуются.

PostgreSQL хранит runtime-данные, snapshot registry и active pointer. ONNX, feature schema, manifest и golden vectors остаются immutable файлами. Локальный `/data/current.json` не является registry и не используется как fallback при отказе БД.

Официальный C API ONNX Runtime позволяет создать окружение/сессию и выполнить модель; интеграция Go должна явно учитывать native-библиотеку и управление памятью. Это не утверждение о наличии официального Go SDK. Допустим community-wrapper, но только внутри адаптера с зафиксированными версиями wrapper + native runtime. Ссылки: `SOURCES.md`, W5/W6.

Редакция с PostgreSQL/PostGIS не меняет HTTP paths, JSON-схемы, ID расчёта и формулы. Выбор файлового runtime-хранилища вместо PostgreSQL больше не является разрешённой альтернативой.

## 2. Модули

```text
cmd/api
cmd/publisher
migrations
internal/storage/postgres
internal/artifacts
internal/httpapi
internal/snapshot
internal/features
internal/inference
internal/forecast
internal/scenario
internal/aggregation
internal/catalog
internal/weather
internal/export
internal/observability
```

Приблизительные внутренние границы:

```go
type BoardingsModel interface {
    // Ordered finite float32 rows according to feature_schema.
    Predict(ctx context.Context, rows [][]float32) ([]float32, error)
}
```

`SnapshotRegistry` реализуется поверх PostgreSQL. Repositories читают историю, профили, погоду и географию по version IDs из snapshot, а `ArtifactStore` открывает immutable модель по проверенным пути и checksum из БД.

`ForecastService` принимает selection и закреплённый snapshot. `ScenarioEngine` получает часовые прогнозы, профиль предложения, эталоны и overrides; он остаётся чистым модулем без SQL и собственного lifecycle. `CsvExporter` вызывает тот же расчётный сервис, а не выполняет независимый SQL/агрегацию.

API использует отдельную роль БД с правами чтения operational data. Импорт, миграции и активация выполняются ролью Publisher/миграций. При добавлении производного SQL-кэша права записи ограничиваются именно этим кэшем.

В `reference/goengine` есть исполняемое ядро формул и тесты. Это не сгенерированный HTTP backend: обработчики, snapshot loading, ONNX и реальное построение признаков нужно реализовать.

## 3. Порядок обработки forecast

1. Ограничить размер body и concurrency; декодировать JSON без неизвестных/дублирующихся полей.
2. Проверить схему и бизнес-условия selection.
3. Найти **запрошенный опубликованный** snapshot через PostgreSQL registry, проверить его доступность/retention и закрепить version IDs. Никакого fallback на current или `/data/current.json`.
4. Получить часовой диапазон; проверить покрытие каждого маршрута и lead time.
5. Построить ключи базовых предсказаний. Проверить кэш.
6. Для отсутствующих в кэше часов прочитать подготовленную историю/погоду/профили из PostgreSQL по закреплённым версиям и собрать признаки. Промах кэша не означает отсутствия snapshot.
7. Выполнить один или ограниченное число batch inference.
8. Проверить количество/порядок результатов и конечность raw output.
9. Применить единый postprocessing модели из manifest; после него ожидаются конечные неотрицательные значения. Отрицательный raw score допускается только если manifest явно предусматривает его обработку.
10. Выполнить формулы baseline/scenario на часах, затем агрегировать.
11. Нормализовать descriptor, вычислить `calculation_id`.
12. Вернуть полный согласованный ответ.

Часовые значения одного маршрута/целевого времени не должны меняться от того, запросили их отдельно, внутри суток или месяца. Признаки зависят от pinned origin/часа, не от границы текущего UI-запроса.

## 4. Матрица операций

| Операция | Владелец |
|---|---|
| `/health/live` | HTTP-процесс |
| `/health/ready` | PostgreSQL/PostGIS + миграции + опубликованный snapshot + готовая ONNX-сессия |
| `/bootstrap` | Каталог + active snapshot + capabilities |
| `/snapshots/{id}` | SnapshotRegistry |
| `/routes`, `/routes/{id}` | RouteCatalog |
| `/routes/{id}/geometry` | PostGIS → GeoJSON закреплённой network_version |
| `/forecasts/query` | ForecastService |
| `/scenarios/evaluate` | ForecastService + ScenarioEngine |
| `/weather` | WeatherRepository закреплённого snapshot |
| `/model-quality`, `/data-sources` | Версионированные metadata/отчётность в PostgreSQL; большие отчёты могут быть linked artifacts |
| `/exports` | Тот же CalculationService + CsvExporter |

Проверка готовности не требует, чтобы Open-Meteo или OpenFreeMap отвечали прямо сейчас. Она учитывает соединение с PostgreSQL, применённые миграции/наличие PostGIS, доступный опубликованный snapshot и пригодную ONNX-сессию/артефакты. Невыполнение обязательной проверки означает `503`. Это использует существующий `Health.checks`, без новой схемы ответа.

Структура БД и extension проверяются при старте/смене схемы; не запускать тяжёлые проверки или миграции на каждом health-запросе. `/health/live` проверяет живость процесса и не зависит от доступности БД.

В `bootstrap` active pointer и metadata snapshot разрешаются одним согласованным чтением из БД, например JOIN. Все последующие чтения внутри запроса используют полученные ID, а не повторный поиск current. Уже закреплённый клиентом snapshot не заменяется новым.

## 5. Время и валидация

RFC3339, явный offset, целые часы. На выходе UTC. `[from,to)`. `from < to`.

`view_mode` ограничивает ширину окна; `snapshot.coverage` ограничивает допустимые цели относительно `forecast_origin_at`. Поле `resolution=day` требует московских полуночей.

Проверки, не покрываемые генерацией типов:

- окно в покрытии всех запрошенных маршрутов;
- не больше 7440 внутренних часовых ячеек;
- не больше 960 возвращаемых `(кадр,маршрут)` ячеек;
- scenario window входит в query window;
- произведение факторов в диапазоне;
- delta выпуска применима к каждому затронутому часу;
- нет отрицательного/превышающего лимит результата выпуска;
- источник погоды и истории не нарушает доступность на origin;
- точная согласованность CSV и calculation hash.

Числовые ограничения и отсутствие поля — разные ошибки. JSON `null` нельзя автоматически превращать в 0. `NaN`/Infinity не являются допустимыми JSON-числами.

## 6. ONNX integration

Model A — регрессор:

```text
input:  features, tensor(float32), [N,F]
output: boardings, tensor(float32), [N,1]
```

Названия, размеры и порядок колонок закреплены ML manifest. Если исходный экспорт выдаёт `[N]` или другое имя, перед релизом приводим артефакт к выбранному контракту либо явно меняем manifest и адаптер совместно. Нельзя угадывать выход по «первому tensor».

Модель берётся из immutable filesystem, путь/версия/SHA-256 — из metadata PostgreSQL. Сессии подготавливаются при старте и подготовке версии, а не заново на каждом прогнозе; затем переиспользуются. Смена модели — только после smoke test и golden vectors.

Все обслуживающие реплики должны иметь доступ к нужным artifacts. Реплика, ещё не подготовившая новую активную модель, не сообщает готовность и не подставляет предыдущую. При обслуживании сохранённого старого snapshot используется именно его модель; отсутствие сессии в памяти не означает `404` и обрабатывается менеджером сессий как подготовка нужной версии с ограниченной конкуренцией. Не удерживать соединение БД во время загрузки ONNX или native inference.

Начальные параметры для измерений: CPU provider, 1 intra-op thread, ограничение 2 конкурентных inference. Это стартовая настройка, а не обещанный optimum. ONNX Runtime имеет собственные потоки и native-memory, поэтому нельзя оценивать RAM только по Go heap.

Не допускать параллельных запросов с общим изменяемым tensor buffer. Если wrapper привязывает buffers к сессии, использовать безопасный pool или сериализацию соответствующей сессии.

HTTP context cancellation не должна освобождать buffers, пока native inference их использует. После таймаута слот concurrency возвращается только когда вычисление фактически закончено/безопасно остановлено.

Порог совместимости Python/ONNX для стартового golden test: `abs(diff) <= 1e-4 + 1e-5*abs(reference)`. Если конкретная модель требует иной tolerance, он фиксируется в manifest и обосновывается; нельзя молча ослаблять тест.

## 7. Кэши

Базовый прогноз:

```text
forecast_snapshot_id + route_id + target_hour
```

Этот ключ закрепляет все версии и origin через immutable snapshot. Кэширует только `B_h`, без сценария. Локальный кэш ответа по normalized descriptor допустим отдельно. Общий кэш ограничен **байтами**, не только числом записей; стартовый бюджет 128 MiB подлежит замеру.

Immutable metadata и GeoJSON тоже можно кэшировать по ID/версии. Active pointer и доступность/удаление snapshot определяются PostgreSQL, а не временем жизни локальной map. Для P0 active pointer читается из БД при bootstrap; кэш не становится вторым registry.

Не включать `selectedFrameIndex` в серверный кэш: его нет в запросе. Не включать fleet overrides в ключ Model A: они не влияют на спрос.

Не хранить единственную копию пользовательского результата в памяти. Любой поддержанный descriptor должен воспроизводиться после очистки кэша на другой реплике: PostgreSQL → закреплённые inputs → ONNX при необходимости → Scenario Engine.

Cache miss сам по себе не возвращает `404`/`410`. Неизвестный или удалённый snapshot определяется БД; потеря соединения с ней — `503`, а не отсутствие сущности. Производный прогнозный SQL-кэш допустим, но не обязателен; пользовательские сценарии сохранять не требуется.

## 8. PostgreSQL, публикации и weather refresh

### 8.1. Runtime data и PostGIS

Обязательные логические сущности перечислены в `ARCHITECTURE.md`, раздел 4. Миграции должны фиксировать version IDs, ключи и связи между snapshot, моделью, историей, погодой, профилями и сетью. Они не должны зависеть от локальных файлов `current.json`.

Для географии:

```text
routes.geometry    geometry(LineString, 4326)
stops.location     geometry(Point, 4326)
```

`route_stops` сохраняет `(network_version, route_pattern_id, stop_id, sequence, route_stop_id)`. Реальные PK/FK должны включать версию сети там, где одна сущность существует в нескольких версиях. Geometry направлений хранится отдельно при необходимости; внешний `RouteGeometry` не меняется.

PostGIS создаётся миграцией/оператором до запуска API. Импорт проверяет тип, SRID, координаты, ссылки и порядок остановок. GeoJSON формируется из запрошенной версии, с прежними feature IDs и properties. Для геозапросов предусмотреть spatial index; для истории/погоды/профилей — индексы по version/route/time, соответствующие запросам. PostGIS не создаёт отсутствующий stop-level target.

Именованные настройки backend: `DATABASE_URL`, `ARTIFACTS_ROOT`, предел пула соединений, таймаут получения соединения/запроса и флаг запуска Publisher. Это конфигурация реализации, не новые поля HTTP API. Секреты не коммитить; подключения и очереди ограничить. Совокупный бюджет соединений всех реплик и Publisher должен соответствовать настройкам БД.

### 8.2. Подготовка кандидата

Один Publisher получает/строит новую версию данных. Погода не мутируется на месте. Новый weather snapshot может переиспользовать модель и историю, но требует нового forecast snapshot.

Последовательность:

1. Получить внешние данные/результат ETL и проверить формат, полноту, ограничения времени.
2. Импортировать новые подготовленные версии в PostgreSQL как ещё не опубликованные; незавершённые версии недоступны обычному API.
3. При необходимости записать immutable ML/static artifacts в окончательный versioned path, проверить SHA-256, tensor schema и golden vectors.
4. Проверить все ссылки, список маршрутов, покрытие, временную доступность, готовность artifacts для API-реплик и ограничения ресурсов.
5. Создать полный snapshot и активировать его короткой транзакцией.

В replay используется исторический forecast origin; текущий интернет-прогноз не подставляется на историческую дату. Обновление realtime-погоды не меняет выбранный replay snapshot.

### 8.3. Атомарная активация

Логическая модель: `forecast_snapshots` плюс одна запись `active_forecast_snapshot` для данного развёртывания. Это схема реализации, а не новые HTTP-ресурсы.

Псевдокод, **не готовая SQL-миграция**:

```text
BEGIN
  lock publication slot
  read active_forecast_snapshot
  assert it equals expected_previous_snapshot
  assert new candidate and all referenced versions are complete
  insert/validate published forecast snapshot metadata
  record deactivated_at for previous active snapshot
  update active_forecast_snapshot to candidate
COMMIT
```

В P0 запись выполняет singleton Publisher. При возможных конкурентных запусках использовать один согласованный PostgreSQL advisory lock для транзакции активации, например `pg_advisory_xact_lock`, плюс уникальные ID и проверку ожидаемого предыдущего указателя. Повтор публикации того же ID с иным содержимым отклоняется. Проверка предыдущего указателя не даёт поздно закончившейся устаревшей сборке молча заменить более новую.

Transaction-level advisory lock действует до конца транзакции; он не является защитой длительной предварительной загрузки. Все публикующие процессы должны соблюдать один протокол. Не выполнять weather download, обучение, копирование модели или golden inference внутри транзакции активации. Техническая справка: [PostgreSQL — Advisory Locks](https://www.postgresql.org/docs/current/explicit-locking.html#ADVISORY-LOCKS).

При любой ошибке до COMMIT старый active pointer остаётся действующим. Изменяемые служебные поля состояния/деактивации отделены от неизменяемого набора расчётных входов snapshot. Незавершённый snapshot не читается через публичные endpoints даже по известному ID.

Сначала обеспечивается наличие immutable файлов, затем фиксируется pointer в БД. Сбой может оставить orphan artifacts, но не snapshot, ссылающийся на ещё не записанную модель. Distributed transaction между PostgreSQL и filesystem не вводится.

### 8.4. Чтение, удаление и несколько реплик

Active pointer и metadata в bootstrap читаются согласованно; далее запрос использует только закреплённые версии. Отдельные повторные SELECT current посередине расчёта запрещены. При `Read Committed` последовательные запросы могут видеть разные committed состояния, поэтому границей согласованности является один раз выбранный immutable snapshot, а не предположение «все SELECT автоматически увидят одну публикацию». Техническая справка: [PostgreSQL — Transaction Isolation](https://www.postgresql.org/docs/current/transaction-iso.html).

Хранить старые snapshots, их входные версии и ML artifacts не меньше retention из API; минимум — 24 часа после деактивации. Не удалять active, shared dependencies обслуживаемых snapshots или входы выполняющегося запроса. Сохранять tombstone для ответа `410`. Cache TTL не заменяет retention registry.

Все API-реплики читают один PostgreSQL и общий/заранее доставленный набор immutable artifacts. Sticky sessions не нужны. Файл, существующий только на локальном диске Publisher, не считается доставленным модели на другие реплики.

### 8.5. Запуск и отказ БД

Запуск: применить миграции → проверить PostGIS → импортировать/проверить версии → подготовить model artifacts → опубликовать snapshot → запустить/подготовить API и ONNX-сессии.

Отсутствие PostgreSQL, миграций, опубликованного snapshot или обязательной модели означает not ready. Внутренняя ошибка БД не преобразуется в `404`. Откат на `/data/current.json` запрещён.

Сбой Weather Refresher/Publisher при доступной БД не останавливает обслуживание предыдущего snapshot. Сбой самой БД — другой режим: запросы, которым требуется registry/данные, получают `503`, readiness снимается. Скрытой автономной файловой реализации нет.

Типы геометрии и методы сериализации следует сверять с [PostGIS — Data Management](https://postgis.net/docs/using_postgis_dbmanagement.html). Эти ссылки — техническая справка реализации; обязательность PostgreSQL/PostGIS установлена решением команды.

## 9. HTTP, безопасность и эксплуатация

Приложение и API размещаются под одним HTTPS origin. В закрытом пилоте — Basic auth на gateway или эквивалентный защищённый контур. Собственного `/auth/login` нет. Mock demo/demo запрещён для реального развёртывания.

Сырые события, `crd_hashcode`, данные карт, device IDs и garage numbers не выдаются и не логируются. Внешние URL/координаты не принимаются от клиента для произвольного proxy — иначе возникает лишняя поверхность злоупотребления.

`application/problem+json`: стабильные `code`, `status`, `request_id`, понятный `detail`. Production сообщения ru-RU. Gateway должен сохранять JSON-формат ошибок на `/api/v1/*`, включая 401.

Стартовые лимиты: body 64 KiB, до 10 маршрутов, до 744 часов/7440 внутренних ячеек, до 960 выходных ячеек, CSV до 7440 строк и 10 MiB. Фактические значения возвращаются в bootstrap.

Ответы forecast/scenario/export: `Cache-Control: no-store`. Геометрия/справочники/immutable metadata/weather: `private, no-cache` + ETag. Проверка авторизации выполняется и при 304.

Rate limit и admission control — по пользователю/контурной идентичности, не только IP. При насыщении возвращается 429/503, не создаётся неограниченная очередь goroutines.

## 10. Генерация Go-контракта

Конфигурация: `tools/oapi-codegen.yaml`.

```bash
mkdir -p generated/go
go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v2.7.2 \
  -config tools/oapi-codegen.yaml openapi.yaml
```

Эта команда **не выполнялась в пакете**: загрузка внешних зависимостей недоступна. Версия приведена по проверенной документации проекта. После генерации нужно установить/зафиксировать runtime-зависимости и скомпилировать код в модуле настоящего backend.

Сгенерированный сервер не реализует все бизнес-проверки. `oneOf`, диапазоны и временные инварианты требуют работающей request validation, а не только Go structs.

## 11. Порядок реализации

Сначала — миграции PostgreSQL/PostGIS, repositories и импорт тестового versioned каталога/истории. Ответить frontend по прежним mock-структурам, но для целевого backend уже через PostgreSQL. Затем — snapshot registry, атомарная публикация и один ONNX batch для суток. После этого — формулы, агрегация недели/месяца, CSV, weather refresh и профилирование.

Mock без БД остаётся только средством frontend-разработки, не альтернативной production-архитектурой. Сам mock, исходники reference engine и тесты не входят в переданный семифайловый архив; ссылки на них относятся к исходному полному пакету.

Первое техническое доказательство — **реальный экспортированный ONNX запускается внутри целевого Go/Docker окружения и проходит golden test**. Не откладывать интеграцию native runtime до дня защиты.

## 12. Приёмка

Backend должен проходить те же fixtures и негативные случаи, что исходный mock. Дополнительно обязательны следующие интеграционные проверки; в этой документационной редакции они **не выполнялись**:

- ONNX parity, отсутствие будущих признаков, реальные IDs и два конкурентных inference без повреждения buffers.
- Запуск на PostgreSQL/PostGIS, чтение версионированной геометрии с сохранением прежнего GeoJSON и feature IDs.
- Очистка всех локальных кэшей → тот же descriptor/calculation ID и воспроизводимый результат на другой реплике.
- Ошибка импорта/артефакта/golden test/транзакции → active pointer не изменён; старый snapshot доступен.
- Две конкурентные публикации → ни частичного snapshot, ни нескольких active pointers, ни молчаливого затирания неожиданно сменившегося current.
- Bootstrap во время активации → согласованный старый либо новый snapshot, без смешения версий.
- Сохранённый старый snapshot и CSV доступны в течение retention; после удаления — `410`, неизвестный ID — `404`.
- Отказ PostgreSQL → readiness `503` и корректная ошибка зависимых запросов; локальный current.json не используется.
- API-реплики на общей БД работают без sticky sessions; отсутствие artifact на реплике не приводит к использованию другой модели.

Нагрузочный тест измеряет полный endpoint, ожидание пула/SQL, inference, память API и ресурсы БД отдельно. Наличие этих требований не является результатом их выполнения.
