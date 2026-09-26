# Подключение ML к Go backend v1.2

Обученная модель пока не передана. Synthetic demo не подтверждает WAPE или качество реального прогноза.

Поставка: `artifacts/<version>/boardings.onnx`, `feature_schema.json`, `golden_vectors.json`, `model_manifest.json`, quality и training config. Используемый CPU Runtime: 1.24.1, Go wrapper `github.com/yalue/onnxruntime_go` 1.26.0. Артефакты immutable; API-реплики должны видеть одинаковые файлы.

Input `features`, float32 `[N,F]`; output `boardings`, float32 `[N,1]`. Runtime metadata сохраняются в `bundle.model`:

```json
{
  "version":"boardings-v1","feature_schema_version":"features-v1",
  "path":"boardings-v1/boardings.onnx","sha256":"<64 hex>",
  "schema_path":"boardings-v1/feature_schema.json","schema_sha256":"<64 hex>",
  "golden_path":"boardings-v1/golden_vectors.json","golden_sha256":"<64 hex>",
  "input":"features","output":"boardings",
  "columns":["hour","weekday","month","lead_hours","route_code","historical_profile"],
  "postprocessing":"clamp_zero","release_status":"published",
  "train_cutoff":"2026-08-31T19:00:00Z"
}
```

`feature_schema.json` runtime projection:

```json
{"version":"features-v1","columns":["hour","weekday","month","lead_hours","route_code","historical_profile"]}
```

Go вычисляет `hour`, `weekday`, `month`, `lead_hours` для каждого целевого часа: Москва, weekday понедельник=0, lead от pinned origin. Остальные columns — конечные подготовленные float32-compatible числа в `hours[].features`. Scaling, categorical mapping, imputation и masks готовит offline ETL; они должны совпадать с обучением. Для преобразованных календарных признаков используйте другое имя, чтобы Go не заменил их raw calendar.

Golden vectors: `{"rows":[[18,1,9,18,0,610]],"expected":[610]}`. Expected — конечный Python prediction после postprocessing. Несколько реальных строк, нули, пропуски/маски и крайние lead times. Tolerance: `1e-4 + 1e-5*abs(reference)`. Поддержаны `identity`, `clamp_zero`, `expm1_clamp_zero`; не применяйте обратное преобразование дважды. Template-модели не публикуются.

Полная схема prepared bundle — `internal/domain/types.go`; исполняемый **синтетический** пример создаёт `go run ./cmd/fixtures --onnx`. Snapshot, RouteDetail/Geometry, WeatherPoint, QualityResponse и SourceList валидируются по OpenAPI. Нужны все часы coverage, без дублей и молчаливого заполнения нулями. `features_available_at` и погодный выпуск не позже origin, train cutoff не позже observations_complete_through. Publisher проверяет покрытие, timestamps и весь batch; отсутствие leakage внутри внешнего ETL подтверждает ML-команда.

`fleet/reference/typical` могут быть null. Нулевой fleet отличается от неизвестного. `synthetic_boardings` запрещён в replay/operational. `prediction_scopes=["route"]`, `stop_forecast_status=unavailable`. Остановочная модель — отдельный будущий адаптер с собственными prepared inputs, reference/typical и закреплёнными stop_model_version/stop_feature_schema_version. Текущий Publisher отклоняет bundle с route_stop, пока этот адаптер не подключён. Маршрутные значения не заменяют stop targets. Реальные route mapping, сеть, погода и все profiles передаёт ETL. Raw CSV parsing, обучение и скачивание внешних источников остаются в offline-контуре.

Публикация: скопировать artifacts на все реплики, подготовить bundle с новым snapshot ID и `expected_previous_snapshot`, затем запустить Publisher с write ролью БД:

```powershell
docker compose run --rm -v "${PWD}/incoming/bundle.json:/data/bundle.json:ro" publisher --bundle /data/bundle.json
```

Максимум bundle 32 MiB, 7440 часов, 256 model features и два raw calendar flags в каждом часе. Feature JSON — не более 64 KiB на час, ключ — не более 128 bytes. Повторное использование version IDs разрешено только при идентичном содержимом; feature_schema_version закрепляет checksum всего schema artifact, включая preprocessing metadata. Изменение его семантики требует нового ID, даже при прежнем списке columns. Ошибка проверки или конкурентная смена active pointer отклоняет публикацию. Старый snapshot остаётся доступен до истечения retention.

## Метаданные погоды после ревью

Bundle теперь обязательно содержит `weather_metadata`:

```json
{
  "weather_snapshot_id": "weather-v2",
  "provider": "open-meteo",
  "source_id": "open-meteo-forecast-source",
  "retrieved_at": "2026-08-31T20:00:00Z",
  "route_locations": {
    "route-1": "moscow-center",
    "route-2": "moscow-center"
  }
}
```

ID совпадает с provenance; provider — open-meteo или synthetic_mock в соответствующем режиме. Каждый маршрут coverage имеет location ID. SourceList содержит ровно одну запись с этим source_id, category=weather, version=weather_snapshot_id и тем же retrieved_at. Время получения не подменяется created_at прогнозного snapshot. Время выпуска и доступности каждого часа остаются в WeatherPoint. Повторное использование weather ID требует идентичных metadata и строк, даже если новый forecast snapshot создан позже.

Миграция 003 создаёт weather_snapshots и составной FK принадлежности направления маршруту. Старые погодные provenance восстановить достоверно автоматически невозможно. Для поставки из schema 2 подготовьте новую погодную версию с настоящими metadata, новые SourceList/forecast ID и опубликуйте bundle после миграции. Старый закреплённый weather endpoint без metadata возвращает 503. Для локального demo перегенерируйте fixture: `go run ./cmd/fixtures --onnx`.

Publisher также требует хотя бы один целый московский календарный день с day/hour, проверяет согласованность RouteDetail/Geometry и общих физических остановок до активации. При ошибке прежний active snapshot сохраняется.

Наличие stop_model_version и stop_feature_schema_version в provenance теперь допускается только парой. Эти поля не включают inference: отдельные stop inputs, storage/adapter, summary/CSV и интеграция ещё нужны. Enrich больше не стирает уже подготовленные stop_readings; route-only ответы остаются пустыми.

## Инварианты после обновлённого ревью

Origin, observations_complete_through и train_cutoff обязательны и ненулевые; train_cutoff <= observations_complete_through <= origin. Fleet provenance: unavailable требует fleet=null/proxy=false; manual_plan — fleet!=null/proxy=false; observed_vehicle_profile — fleet!=null/proxy=true. Absolute scenario снимает proxy flag, delta сохраняет baseline provenance. Перед обновлением API примените миграции 004/005; подробности и оставшиеся stop/refresher границы — [UPDATED_REVIEW_FIXES.md](UPDATED_REVIEW_FIXES.md).
