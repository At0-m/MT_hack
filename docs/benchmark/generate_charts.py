#!/usr/bin/env python3
"""Render charts from the supplied summary.csv; does not run a benchmark.
Dependency: matplotlib. Run this script from any working directory.
"""
from pathlib import Path
import csv
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

root = Path(__file__).resolve().parent
with (root / 'summary.csv').open(encoding='utf-8', newline='') as f:
    rows = list(csv.DictReader(f))
c = [int(r['concurrency']) for r in rows]
rps = [float(r['rps']) for r in rows]

fig, ax = plt.subplots(figsize=(10.8, 4.8))
ax.plot(c, rps, marker='o', linewidth=2, markersize=7)
for x, y in zip(c, rps):
    ax.annotate(f'{y:.1f}', (x, y), xytext=(0, 11), textcoords='offset points', ha='center', fontsize=12)
ax.set(title='Tramflow: пропускная способность', xlabel='Параллельные запросы', ylabel='Запросы в секунду (RPS)', xticks=c, ylim=(0, 750), xlim=(2, 34))
ax.title.set_fontsize(17)
ax.xaxis.label.set_fontsize(12)
ax.yaxis.label.set_fontsize(12)
ax.tick_params(labelsize=11)
ax.grid(True, alpha=0.2)
fig.text(0.5, 0.015, '5 000 запросов на точку; 25 000 запросов суммарно; 0 зарегистрированных ошибок.', ha='center', fontsize=10)
fig.tight_layout(rect=(0, 0.045, 1, 1))
fig.savefig(root / 'throughput.png', dpi=220)
plt.close(fig)

fig, ax = plt.subplots(figsize=(10.8, 4.8))
for key, label, marker in [('p50_ms','p50','o'), ('p95_ms','p95','s'), ('p99_ms','p99','^')]:
    ax.plot(c, [float(r[key]) for r in rows], marker=marker, linewidth=2, markersize=6, label=label)
for x, r in zip(c, rows):
    y = float(r['p95_ms'])
    ax.annotate(f'{y:.2f}', (x, y), xytext=(0, -16), textcoords='offset points', ha='center', fontsize=10)
ax.set(title='Tramflow: задержка HTTP-запросов', xlabel='Параллельные запросы', ylabel='Задержка, мс', xticks=c, ylim=(0, 85), xlim=(2,34))
ax.title.set_fontsize(17)
ax.xaxis.label.set_fontsize(12)
ax.yaxis.label.set_fontsize(12)
ax.tick_params(labelsize=11)
ax.grid(True, alpha=0.2)
ax.legend(fontsize=11, loc='upper left')
fig.text(0.5, 0.015, 'Подписи значений: p95. Каждая точка — отдельный короткий прогон из summary.csv.', ha='center', fontsize=10)
fig.tight_layout(rect=(0, 0.045, 1, 1))
fig.savefig(root / 'latency.png', dpi=220)
plt.close(fig)
print('Created throughput.png and latency.png from summary.csv')
