# CSV экспорта v1

Отсутствующий в архиве CSV.md восстановлен как контракт **этой реализации**. Согласуйте колонки с фронтендом при необходимости; HTTP API и calculation_id сохранены.

Одна строка — один `(frame, route)`, порядок тот же, что в JSON `frames`. CSV не является submission для соревнования. UTF-8 BOM, `;`, CRLF, десятичная точка, RFC4180 quoting. Недоступные числовые показатели — пустая ячейка.

Колонки по порядку:

```text
calculation_id;forecast_snapshot_id;route_id;from;to;baseline_boardings;evaluated_boardings;baseline_vehicle_hours;evaluated_vehicle_hours;baseline_load_index;evaluated_load_index;required_vehicle_count;fleet_source;fleet_is_proxy;weather_mode
```

Числа не округляются до экспорта; содержат те же значения, что JSON. Отрицательные дельты допустимы в JSON; в этом CSV экспортируются исходные baseline/evaluated. Отдельные totals в CSV не добавляются, чтобы не удвоить суммы. Показатели периода вычисляются по CALCULATIONS.md, а не усреднением индексов строк.
