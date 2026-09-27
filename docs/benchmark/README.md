# Готовые материалы для отчёта

- `Tramflow_Benchmark_Report.pdf` — двухстраничный отчёт.
- `throughput.png`, `latency.png` — готовые графики для вставки.
- `summary.md` — текст и таблица для README/заявки.
- `summary.csv` — предоставленные численные результаты без изменения точности.
- `load-c16.json`, `smoke-summary.json` — предоставленные JSON-результаты.
- `generate_charts.py` — повторная генерация PNG из CSV; требует matplotlib.
- `provenance.json` — происхождение и ограничения данных.

Все материалы построены по уже предоставленным замерам. Новые измерения, ресурсы API и временные ряды Grafana не генерировались.

Повторно построить графики: `python3 generate_charts.py`.
