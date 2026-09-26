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

Максимум bundle 32 MiB, 7440 часов, 256 features. Повторное использование version IDs разрешено только при идентичном содержимом. Ошибка проверки или конкурентная смена active pointer отклоняет публикацию. Старый snapshot остаётся доступен до истечения retention.
