# Фактические проверки — контракт 1.2, 26.09.2026

Проверена Go-реализация по последнему архиву MT_HACK_ALL_ROADS_LEAD_TO_FINALE.zip.

## Пройдено

- Сборка и тесты Go 1.26.2 на Windows и Ubuntu WSL2.
- Ответы всех JSON endpoints проверены по поставленному OpenAPI; strict JSON отклоняет duplicate keys, лишние поля и недопустимые null.
- Московские календарные окна day/week/month/custom, переход года и високосный февраль; custom/hour и отсутствие stop model дают 422.
- Canonical v2 golden forecast/stop-scope и scenario/export hashes; точный CSV replay, 409 при несовпадении calculation_id.
- Расчёт отношений по суммам знаменателей, различение неизвестного/нулевого выпуска и эталона, максимальный required fleet, ограниченное окно сценария.
- Совместимость indicator key/variant, отсутствие копий маршрутных значений в stop_readings.
- Реальный PostgreSQL 16/PostGIS: миграции дважды, валидация и публикация bundle, PostGIS GeoJSON, одинаковые расчёты из двух холодных кэшей, сохранение active при ошибке кандидата, конкурентный CAS с одним победителем, retention/410, unknown/404, закрытая БД/503.
- Login/session/logout, HttpOnly/SameSite cookie, ETag после auth, CORS и сессии между двумя PostgreSQL pools.
- Native ONNX Runtime 1.24.1 на Windows: synthetic MatMul golden parity и 200 вызовов с отдельными буферами от 8 конкурентных workers.
- `go vet` без замечаний; Compose configuration валидируется.
- `go test -race` на Ubuntu WSL2 с реальным PostgreSQL/PostGIS прошёл без гонок (native ONNX проверен отдельно на Windows).

Тест PostgreSQL создаёт отдельную случайную БД и удаляет только её. Без TEST_DATABASE_URL он явно пропускается. Native тест требует `--onnx` fixtures, build tag onnx, C compiler и ONNX_LIBRARY_PATH.

## Нагрузочный замер

Полный HTTP forecast endpoint, opaque cookie session, настоящий PostgreSQL, один маршрут и 24 hourly frames. 3000 запросов, 16 workers: **0 ошибок, p95 13.293 ms**, около 1787 requests/s за 1.68 s. API peak RSS — 29260 KiB. Полный результат: `performance.json`.

Это короткий локальный прогон в WSL2 на 12 логических CPU, без CPU/memory quotas, на synthetic_mock и преимущественно прогретом prediction cache. Он не подтверждает sustained throughput, скорость обученной ONNX-модели или требования под контейнерными ограничениями. PostgreSQL всё равно читается на каждом запросе.

## Не подтверждено

Docker Desktop на этой машине падает при инициализации dockerInference. Сборка image и Compose end-to-end не выполнены; контейнерные CPU/RSS/load checks остаются отдельной проверкой после восстановления Docker.

Обученная модель, транспортные prepared inputs и остановочная ML-модель не переданы. Реальные WAPE/backtest, stop forecasts и качество источников не заявлены. Optional StopModelAdapter пока отсутствует; API и Publisher сообщают это явно. Локальный native MatMul проверяет интерфейс и память, не качество прогноза.

## После BACKEND_REVIEW.md

Проверки исправлений, новые регрессии, native LRU lifecycle и точные оставшиеся ограничения — в [REVIEW_FIXES.md](REVIEW_FIXES.md). Старый performance.json сохранён как исторический замер; после исправлений нагрузка повторно не измерялась.

## После BACKEND_REVIEW_UPDATED.md

Новые исправления и фактические проверки Go 1.26.8, Linux native ONNX + race с PostGIS и максимальной сетки под cgroup 2 GiB / 2 CPU: [UPDATED_REVIEW_FIXES.md](UPDATED_REVIEW_FIXES.md). Этот раздел дополняет исторические ограничения выше; Docker image/Compose остаются непроверенными.

## После BigTech V3

Актуальные auth/migration/model-cache правила и новые проверки: [BIGTECH_V3_FIXES.md](BIGTECH_V3_FIXES.md). Этот документ сохраняет исторические результаты предыдущей версии.
