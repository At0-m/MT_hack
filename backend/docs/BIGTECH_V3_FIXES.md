# Исправления BACKEND_REVIEW_BIGTECH_V3.md

Дата: 2026-09-27. Ревью сопоставлено с исходниками; рекомендации документа не используются как разрешение на внешние публикации или изменение согласованного frontend-контракта.

## Подтверждённые исправления

| Пункт | Изменение |
| --- | --- |
| P1-1: account lockout | Пароль проверяется независимо от account failure state. Неверные credentials увеличивают счётчик; успешные сбрасывают его. После 10 ошибок меняется ответ на неверный пароль, но правильный пароль проходит. До bcrypt остаются IP limiter, два concurrent slots и общий PostgreSQL budget 120 попыток/мин на все API-реплики. |
| P1-2: proxy fail-open | Missing/invalid/слишком длинный/полностью trusted XFF использует peer IP как coarse bucket. Пустой адрес не считается unlimited. Нужны корректный XFF и gateway rate limiting. |
| P1-3: Inf/NaN после формул | Publisher и runtime проверяют numeric admission bounds; Publisher рассчитывает каждый advertised view для каждого маршрута и custom. Derived metrics проверяются до сериализации. Предсказания проверяются для всей сетки chunk-ами 128. |
| P1-4: ONNX mutex | Per-version loading reservation coalesces загрузку. Checksum, nativeOpen, golden inference и victim.Close идут вне глобального mutex. Кэшированные модели продолжают acquire/release; ожидание той же версии учитывает context. |
| P2-1: rotation | Password update и отзыв всех user sessions атомарны, под тем же user advisory lock, что создание session. CreateSession повторно проверяет hash, использованный bcrypt, чтобы вход со старым паролем не создал session после rotation. |
| P2-2: weather semantics | missing требует null temperature/precipitation/code; climatology запрещает provider run/code; forecast требует ненулевой provider run. Проверки availability/origin сохраняются. |
| P2-3: migration integrity | Проверяется точный непрерывный ordered set и SHA-256 каждого SQL. Миграция 006 вводит checksums; drift и пропуски запрещены. CRLF нормализуется в LF для одинаковых hashes Windows/Linux. |
| P2-4: geometry | Новая публикация требует геометрию каждого маршрута; сохраняются detail/geometry consistency проверки. |
| P2-5: indicators | Сохранён контракт v1.2 с пятью слотами. Event при активном factor имеет явный приоритет над peak; политика описана в UI_POLICY.md. Числовые frame metrics и summary peak сохраняются. Шестой слот потребует согласованной contract revision. |
| P2-6: DB timeouts | API statement timeout 5 с; Publisher/migrations/GC — 300 с, в пределах общего context 5 мин. Отдельный OpenMaintenance pool. |
| P2-7: model memory | Streaming SHA вместо полной Go-копии ONNX. Admission: до 4 sessions/reservations включая Close; ONNX до 128 MiB, schema/golden до 1 MiB каждый. Failed Close сохраняет admission slot до shutdown. Это не измерение/гарантия native working memory. |
| P2-8: observability | HTTP log включает actual status, route pattern, request ID, duration, user ID. Auth audit: outcome, hashes subject/client; password/token не логируются. Model lifecycle log: count/load duration/evicted version. |
| P2-9: capability acceptance | Publisher проверяет day/week/month, если advertised, и custom для каждого coverage route: календарь, coverage/resolution, calculation, JSON schema. Geometry, weather metadata и quality/sources валидируются до activation. Readiness остаётся лёгким. |

Общий pre-auth budget ограничивает стоимость password spray, но при общей перегрузке может вернуть 429 любому клиенту. Это общий service admission limit, а не блокировка конкретного аккаунта. Полная защита от распределённого DoS требует gateway и эксплуатационной настройки. Post-auth failure state не позволяет пропускать bcrypt: лимиты до проверки пароля обязательны.

## Числовые границы

Технические admission limits не являются оценкой вместимости трамвая:

- baseline hourly boardings: 0..1e9;
- fleet: 0..200; reference/typical: 0..1e9;
- положительные fleet/reference/typical: не меньше 1e-6; нули сохраняют отдельную семантику;
- temperature: -100..100 °C; precipitation: 0..1000 mm на час;
- уже согласованные scenario factors: 0.5..2, произведение 0.25..4.

NaN/Inf запрещены; значения не округляются и не заменяются тихо. Runtime проверяет также все derived metrics, weather sums, typical-relative values и deltas. Publisher отклоняет кандидата до записи/переключения active pointer. Если ETL нужны другие пределы, их нужно пересмотреть вместе с numeric regression tests.

## Обновление и эксплуатация

Перед новой API-сборкой запустить Publisher с write-role: `go run ./cmd/publisher --migrate`. Теперь требуется **schema 6**. Миграции 001–005 не изменялись.

У legacy schema не было checksum evidence. Миграция 006 сначала проверяет непрерывность ledger, затем закрепляет текущие SQL hashes. Она не доказывает, что именно эти bytes применялись раньше; существующую БД нужно проверить при rollout. После adoption изменение уже применённого SQL обнаруживается. API не обслуживает schema 5.

Новая publication acceptance строже: короткий forecast с одним днём должен объявлять только day, не week/month. Если advertised month не помещается в coverage, кандидат отклоняется. Старые уже опубликованные snapshots автоматически не перепроверяются; подготовьте валидный новый bundle.

Dev Compose API/Publisher получили read_only, cap_drop=ALL, no-new-privileges и ограниченный /tmp tmpfs. Dev COOKIE_SECURE=false и sslmode=disable не превращены в production-настройки. Image digest pinning, SBOM/image scan и production manifest остаются эксплуатационной работой.

## Проверки

Обычные Go-тесты и vet прошли. Регрессии проверяют large/subnormal profiles, huge weather/boardings, geometry, weather modes, advertised resolution, derived Inf; вход после серии failed passwords и сброс state; coarse proxy bucket; статус/audit без password; отсутствие cross-model блокировки при load и Close.

Linux native ONNX + `-race` с настоящим PostgreSQL 16/PostGIS прошёл. В изолированной случайной тестовой БД проверены gaps/checksum drift/legacy adoption, shared auth budget, правильный пароль после 11 ошибок на другой реплике, session revocation и отказ session creation по старому verified hash. Unit-тесты model manager используют управляемые блокирующие loader/Close; native suite отдельно проверяет golden parity, concurrent buffers и eviction/reload.

Предельный HTTP-прогон новой версии: 10 routes × 744 hours × 256 features, синтетическая native MatMul; отдельная API-роль без Publisher permissions. API в WSL2 cgroup 2 GiB / 2 CPU: cold 4,2826 с, hot 0,0591 с, scenario 0,062 с; publication 4,875 с. Peak API cgroup memory 63 365 120 bytes (60,4 MiB); OOM=0. Из 32 concurrent cold max requests один принят, 31 получил ожидаемый 429; по 181 live/ready probes, ошибок 0. [Сырые результаты](v3-boundary-performance.json).

Это не Docker image test, не обученная модель и не sustained production load. Старые отчёты сохранены как исторические замеры. Новый файл явно описывает ограничения измерения.

## Что остаётся открытым

- Stop-level storage/model/adapter/inference/summary/CSV pipeline.
- Automatic Open-Meteo ETL/refresher.
- Real trained ONNX, WAPE/backtest и нагрузка этой модели.
- Native memory требует проверки на реальных моделях; четыре small synthetic sessions не доказывают bounded RSS для любого графа. Active model pinning и Prometheus/RSS metrics пока отсутствуют.
- Filesystem artifact GC, hard watchdog для зависшего native kernel.
- Docker image/Compose E2E локально не подтверждены; CI запускается после пользовательского push.
- Следующая revision контракта: view_modes maxItems=3 и решение о шестом indicator slot.

Новые исправления не объявляют эти компоненты готовыми.
