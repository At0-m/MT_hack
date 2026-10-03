# Запуск TramFlow

Гайд соответствует версии с восстановленным первоначальным frontend. **Docker Compose запускает backend и БД; сайт запускается отдельно через Vite.** Готовый production-запуск frontend в контейнере здесь не заявляется. Полный запуск текущей версии через Docker на этой машине при подготовке гайда не выполнялся; команды сверены с compose.yaml, Dockerfile и конфигурацией frontend.

## Что установить

- Git.
- Docker Desktop с Linux containers и Docker Compose v2; Docker Engine должен быть запущен.
- Python 3.10+ для scripts/init_env.py.
- Node.js, совместимый с Vite 8 (актуальная LTS), и pnpm.
- Интернет при первой сборке контейнеров, установке frontend-зависимостей и загрузке подложки карты.

Go на хосте не требуется для Docker-запуска: backend собирается внутри образа. Для локальных Go-команд требуется Go 1.26.8.

Проверка в PowerShell:

```powershell
git --version
docker version
docker compose version
python --version
node --version
pnpm --version
```

## 1. Получить проект

Для нового checkout:

```powershell
git clone --branch dev https://github.com/At0-m/MT_hack.git
cd MT_hack
```

Если репозиторий уже есть, перейдите в него. Только при чистом рабочем дереве обновите ветку:

```powershell
cd C:\Users\nikit\OneDrive\Desktop\MT_hack
git status
git switch dev
git pull --ff-only origin dev
```

Все следующие backend-команды выполняются из корня MT_hack.

## 2. Создать настройки

```powershell
python scripts/init_env.py
```

Скрипт создаёт приватный корневой .env со случайными паролями, но не перезаписывает существующий. Откройте .env локально в редакторе: API_USER и API_PASSWORD нужны для входа. Не публикуйте содержимое файла.

Разрешите Origin исходного frontend. В первой PowerShell-сессии:

```powershell
$env:CORS_ORIGIN = 'http://localhost:5173'
```

Переменная используется Docker Compose при создании контейнера API. Для постоянной настройки добавьте `CORS_ORIGIN=http://localhost:5173` в корневой .env; shell override не нужен, если эта строка уже есть. Используйте localhost и в браузере: адрес 127.0.0.1 имеет другой Origin.

COOKIE_SECURE=false в локальном HTTP demo; для публичного HTTPS deployment нужны отдельные настройки TLS/Origin и secure cookie.

## 3. Запустить backend

```powershell
docker compose up -d --build --wait --wait-timeout 300
docker compose ps -a
```

Первая сборка скачивает Go-зависимости, PostGIS, ONNX Runtime и образы. Seed применяет миграции, создаёт пользователя и публикует synthetic stress demo. Завершившийся seed с кодом 0 — нормальное состояние. db и api должны стать healthy.

Проверка API:

```powershell
Invoke-RestMethod http://localhost:8080/health/live
Invoke-RestMethod http://localhost:8080/health/ready
```

Адрес http://localhost:8080 — API gateway: в корне он возвращает JSON, **это не сайт**.

Seed не перезаписывает произвольный существующий active snapshot. Если в используемой БД уже другая версия, сначала проверьте логи и выбирайте нужную поставку. Для отдельной чистой demo-БД можно задать другое имя Compose-проекта до запуска:

```powershell
$env:COMPOSE_PROJECT_NAME = 'tramflow-review'
```

Используйте это же имя при дальнейших up/logs/down. Новый project не освобождает порт 8080: если он занят, сначала остановите прежний стек без удаления volume либо настройте другой порт.

## 4. Запустить исходный frontend

Откройте вторую PowerShell-сессию:

```powershell
cd C:\Users\nikit\OneDrive\Desktop\MT_hack\frontend
pnpm install --frozen-lockfile
```

Если pnpm не установлен, используйте поддерживаемый вашей установкой Node способ установки; например:

```powershell
npm install -g pnpm
```

Создайте frontend/.env.local, если его ещё нет:

```powershell
if (-not (Test-Path .env.local)) {
    Copy-Item .env.example .env.local
}
```

Настройки в этом файле:

```dotenv
VITE_API_MODE=real
API_PROXY_TARGET=http://localhost:8080
```

Затем:

```powershell
pnpm dev
```

Оставьте процесс работающим и откройте **http://localhost:5173**. Введите API_USER и API_PASSWORD из **корневого** .env. Vite перенаправляет /api на gateway, который передаёт запросы Go API.

Не используйте pnpm dev:mock для проверки реального backend: этот режим обслуживается тестовым API. В mock логин demo/demo, данные синтетические. Панель искусственных ошибок доступна только на dev-сервере с ?dev=1.

## 5. Проверить работу

1. Авторизоваться и дождаться загрузки каталога маршрутов и расчёта.
2. Выбрать маршрут и дату в пределах coverage snapshot, проверить карту и кадры времени.
3. Открыть аналитику, изменить выпуск и проверить серверный пересчёт.
4. Проверить CSV для подтверждённого расчёта.

Если данные synthetic, это demo. Остановочные значения показываются только при наличии соответствующего scope/capability; route-only backend не обязан возвращать stop predictions. Восстановленный frontend не содержит добавленного ранее редактора коэффициентов погоды/событий/сезона.

## 6. Логи и типовые ошибки

```powershell
docker compose logs --tail 100 db seed api gateway
```

| Симптом | Что проверить |
|---|---|
| Docker Engine недоступен | Docker Desktop запущен, выбран Linux containers |
| API не ready | Логи db/seed/api, миграции, ONNX artifact и active snapshot |
| Seed сообщает другой active snapshot | Не удалять данные; выбрать согласованную поставку или отдельный Compose project |
| 403 при входе/POST | CORS_ORIGIN=http://localhost:5173, адрес браузера localhost; пересоздать API после изменения настройки |
| Пароль не подходит | API_USER/API_PASSWORD из корневого .env; demo/demo относится только к frontend mock |
| API gateway показывает JSON | Открыть сайт на :5173, а не :8080 |
| Vite proxy ECONNREFUSED | API готов на :8080, API_PROXY_TARGET верный |
| 5173 занят | Остановить предыдущий Vite; strictPort не переключит порт автоматически |
| Нет stop forecasts / дата вне покрытия | Проверить bootstrap capabilities и coverage; не подставлять значения произвольно |
| `bad interpreter`, `/bin/sh^M` | Shell-скрипты должны иметь LF; восстановить LF для соответствующих .sh, затем пересобрать образ |

После изменения CORS_ORIGIN в той же первой PowerShell-сессии:

```powershell
docker compose up -d --force-recreate api gateway
```

Ошибки интерфейса проверяйте также в браузерной консоли и Network. Не отправляйте .env, cookie или пароли в общий чат.

## Остановка и следующий запуск

Frontend: Ctrl+C во второй сессии. Backend:

```powershell
docker compose down
```

Эта команда сохраняет именованный volume БД. Следующий запуск:

```powershell
docker compose up -d --wait --wait-timeout 300
```

Используйте прежние Compose project, CORS_ORIGIN и .env. Не используйте down -v для обычного перезапуска.

## Проверки, импорт и отчёт

Frontend из frontend/:

```powershell
pnpm build
pnpm test
pnpm lint
```

Версия frontend после отката отличается от ранее проверенной интеграционной версии; результаты старого browser suite не доказывают работоспособность текущей версии.

Приём 15-минутных CSV описан в [критерии данных](CRITERION_COEFFICIENTS.md), погода/occupancy — в [ML_EXTERNAL_INPUTS.md](../backend/docs/ML_EXTERNAL_INPUTS.md). Реальные артефакты требуют [ML-контракта](../backend/docs/ML_HANDOFF.md), подготовленного bundle и отдельной публикации. Запуск demo не подключает автоматически все 30 переданных моделей.

Ссылки для жюри: [текст формы](SUBMISSION_FORM.md), [benchmark PDF](benchmark/Tramflow_Benchmark_Report.pdf), [исходные результаты](benchmark/summary.csv).

### Новый benchmark + Grafana

После успешного `make smoke` можно запустить наблюдаемый benchmark:

```powershell
make benchmark-observed RATE=50 DURATION=2m WARMUP=30s BENCH_SCRIPT=mixed-workload
```

Команда поднимает основной backend вместе с Prometheus, Grafana и postgres-exporter, а затем запускает k6 на существующем стеке. k6 отправляет временные ряды в Prometheus через remote-write и помечает каждый запуск уникальным `testid`. Откройте:

- Grafana: `http://localhost:3000/d/tramflow-perf/tramflow-benchmark-observability`
- Prometheus: `http://localhost:9091`

Логин Grafana — `admin`, пароль находится в корневом `.env` в `GRAFANA_PASSWORD`. Не публикуйте его.

Dashboard показывает client-side k6 RPS/p95/p99/semantic error ratio, а также API RPS, server-side p95, ошибки, CPU/RSS, forecast p95, cache hit ratio, storage p95, DB pool, PostgreSQL connections, swap и native ONNX метрики. Вверху dashboard можно выбрать `testid` конкретного запуска. Для synthetic demo панели native inference могут оставаться без данных — это ожидаемо.

Численный итог прогона сохраняется в `benchmarks/results/<run>/benchmark.md` и `report.json`. Grafana предназначена для временных рядов; при расхождении итоговых чисел источником benchmark-результата считаются k6 summary и сохранённые артефакты run directory.

Остановить только monitoring-сервисы:

```powershell
make observability-down
```
