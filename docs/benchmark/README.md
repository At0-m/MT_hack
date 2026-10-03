# Benchmark report

Этот каталог является готовой заменой старого `docs/benchmark/`.

Для быстрого просмотра откройте [`summary.md`](summary.md). Полный отчёт: [`Tramflow_Benchmark_Report.pdf`](Tramflow_Benchmark_Report.pdf).

Ключевые подтверждённые результаты: 45-минутный soak при 400 RPS с p95 9,15 мс / p99 16,98 мс и одной business error на 1 080 001 запрос; короткий mixed-workload 500 RPS прошёл без ошибок; на 600 RPS зафиксированы 180 ошибок и нарушение error SLO.

Результаты относятся к `synthetic_mock` и прогретому cache. Они не являются измерением качества или latency реальной обученной ML-модели.

Старые короткие concurrency-бенчмарки из предыдущего `docs/benchmark` этим пакетом заменяются и не должны смешиваться с новой серией.
