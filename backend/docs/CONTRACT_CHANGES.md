# Принятые изменения контракта 1.2

Источник — последний архив MT_HACK_ALL_ROADS_LEAD_TO_FINALE.zip. OpenAPI не переписывался под реализацию.

- Canonical v2 включает spatial_detail; golden forecast/scenario hashes проверяются тестами.
- day/week/month — точные московские календарные интервалы; custom только day, до 31 дня, без обрезания.
- RouteReading.stop_readings вместо map_bars; остановочные значения не копируются с маршрутов.
- Coverage prediction_scopes/stop_forecast_status, capabilities stop_forecasts/segment_forecasts/decorative_approach_effect.
- Visualization.stop_columns — декоративный frontend ramp, не аналитический прогноз сегмента.
- UiIndicator.variant; key/variant compatibility проверяется отдельно от JSON schema.
- Quality: prediction_scope и unit относятся к отдельным ValidationRecord, не к всему QualityResponse.

Остановочный адаптер пока не реализован: trained ML artifacts не переданы. API явно возвращает 422 STOP_FORECAST_UNAVAILABLE и route-only capabilities.
