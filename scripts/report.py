#!/usr/bin/env python3
"""Generate evidence, not invented performance claims. Missing data is not zero."""
from __future__ import annotations
import argparse
import csv
import json
import math
import statistics
from pathlib import Path

def values(summary: dict, name: str) -> dict:
    return summary.get("metrics", {}).get(name, {}).get("values", {})

def stats(items: list[float]) -> dict:
    if not items: return {}
    return {"avg": statistics.mean(items), "max": max(items), "min": min(items), "count": len(items)}

def parse_prometheus(text: str) -> dict[str, float]:
    result = {}
    for line in text.splitlines():
        if not line or line.startswith("#"): continue
        try:
            name, value = line.rsplit(None, 1)
            number = float(value)
            if math.isfinite(number): result[name] = number
        except ValueError:
            continue
    return result

def percentile_from_histogram(before: dict, after: dict, metric: str, q: float = .95) -> float | None:
    import re
    buckets = {}
    for name, value in after.items():
        if not name.startswith(metric + "_bucket{"): continue
        found = re.search(r'le="([^"]+)"', name)
        if not found: continue
        boundary = float(found.group(1))
        delta = value - before.get(name, 0)
        if delta < 0: return None 
        buckets[boundary] = buckets.get(boundary, 0) + delta
    total = buckets.get(float("inf"), 0)
    if total <= 0: return None
    target, previous_bound, previous_count = total * q, 0.0, 0.0
    for boundary, count in sorted(buckets.items()):
        if count >= target:
            if math.isinf(boundary): return previous_bound
            if count == previous_count: return boundary
            return previous_bound + (boundary - previous_bound) * (target - previous_count) / (count - previous_count)
        previous_bound, previous_count = boundary, count
    return None

def render(directory: Path) -> dict:
    manifest = json.loads((directory / "environment.json").read_text())
    summary_path = directory / "k6-summary.json"
    summary = json.loads(summary_path.read_text()) if summary_path.exists() else {}
    code = int((directory / "k6.exitcode").read_text()) if (directory / "k6.exitcode").exists() else None
    if manifest["run"]["script"] in {"smoke", "login"}:
        assertions = [test.get("ok") for metric in summary.get("metrics", {}).values() for test in metric.get("thresholds", {}).values()]
        passed = bool(code == 0 and assertions and all(v is True for v in assertions))
        result = {"format":"tramflow-functional-v1", "functional_passed":passed, "http_slo_passed":False, "exit_code":code,
                  "note":"Functional/low-rate auth check only. Not a service capacity benchmark."}
        (directory/"report.json").write_text(json.dumps(result,indent=2)+"\n")
        (directory/"benchmark.md").write_text("# Functional evidence\n\nStatus: **" + ("PASS" if passed else "FAIL / NOT VALIDATED") + "**.\n\nNo service-capacity RPS claim. See k6-summary.json, environment.json and final-state.json.\n")
        return result
    count = values(summary, "business_requests{phase:measurement}").get("count", 0)
    success = values(summary, "business_successes{phase:measurement}").get("count", 0)
    lat = values(summary, "business_latency{phase:measurement}")
    wall = values(summary, "business_client_wall{phase:measurement}")
    errors = values(summary, "business_failures{phase:measurement}").get("rate")
    drops = values(summary, "dropped_iterations{scenario:measure}").get("count", 0)
    start = values(summary, "measurement_request_start_unix_ms").get("min")
    end = values(summary, "measurement_request_end_unix_ms").get("max")
    elapsed = (end - start) / 1000 if start is not None and end is not None and end > start else None
    duration = manifest["run"]["duration_seconds"]
    all_thresholds = [test.get("ok") for metric in summary.get("metrics", {}).values() for test in metric.get("thresholds", {}).values()]
    http_pass = bool(count and code == 0 and all_thresholds and all(v is True for v in all_thresholds) and drops == 0)
    resources = {}
    stats_path = directory / "system.csv"
    if stats_path.exists() and elapsed:
        with stats_path.open() as f:
            rows = [row for row in csv.DictReader(f) if start <= float(row["timestamp_ms"]) <= end]
        for cid in {row["container_id"] for row in rows}:
            subset = [row for row in rows if row["container_id"] == cid]
            resources[cid] = {"name": subset[0]["name"], "service": subset[0]["service"],
                "cpu_quota_percent": stats([float(row["cpu_quota_percent"]) for row in subset if row["cpu_quota_percent"]]),
                "docker_working_set_bytes": stats([float(row["memory_usage_bytes"]) for row in subset])}
    samples_path = directory / "metrics.ndjson"
    runtime = {}
    if samples_path.exists() and elapsed:
        samples = [json.loads(line) for line in samples_path.read_text().splitlines() if line]
        samples = [s for s in samples if start <= s["timestamp_ms"] <= end]
        for cid in {s["container_id"] for s in samples}:
            subset = sorted((s for s in samples if s["container_id"] == cid), key=lambda s:s["timestamp_ms"])
            first,last = subset[0]["metrics"],subset[-1]["metrics"]
            mem = [s["metrics"]["process_resident_memory_bytes"] for s in subset if "process_resident_memory_bytes" in s["metrics"]]
            def delta(metric):
                a,b=first.get(metric),last.get(metric)
                return b-a if a is not None and b is not None and b>=a else None
            hits,misses = delta("forecast_cache_hits_total"),delta("forecast_cache_misses_total")
            runtime[cid] = {"sample_count":len(subset),"rss_bytes":stats(mem),"rss_first_bytes":mem[0] if mem else None,"rss_last_bytes":mem[-1] if mem else None,
                "cache_cell_hit_ratio": hits/(hits+misses) if hits is not None and misses is not None and hits+misses else None,
                "onnx_inference_p95_estimate_seconds":percentile_from_histogram(first,last,"onnx_inference_duration_seconds"),
                "db_query_p95_estimate_seconds":percentile_from_histogram(first,last,"db_query_duration_seconds"),
                "max_process_swap_bytes":max((s["metrics"].get("process_swap_bytes",0) for s in subset),default=0),
                "max_cgroup_swap_bytes":max((s["metrics"].get("container_swap_current_bytes",0) for s in subset),default=0),
                "swap_observed": all("process_swap_bytes" in s["metrics"] for s in subset)}
    result = {"format":"tramflow-report-v1","http_slo_passed":http_pass,"exit_code":code,"requests":count,"successful_requests":success,
        "target_rps":manifest["run"]["rate"],"requests_per_scheduled_second":count/duration if duration else None,
        "completion_window_seconds":elapsed,"completed_rps_including_drain":count/elapsed if elapsed else None,
        "successful_rps_including_drain":success/elapsed if elapsed else None,"latency_ms":lat,"client_wall_ms":wall,
        "business_error_rate":errors,"dropped_iterations":drops,"resources":resources,"runtime_samples":runtime,
        "resource_verdict":"REVIEW_REQUIRED: CPU, RAM plateau, swap, DB contention and generator saturation must be reviewed; HTTP pass alone is not sustainable capacity."}
    (directory/"report.json").write_text(json.dumps(result,indent=2)+"\n")
    def fmt(v, digits=2): return "not measured" if v is None else f"{v:.{digits}f}"
    provenance = manifest.get("provenance",{})
    text = ["# Performance evidence", "",f"Status: **{'HTTP SLO PASS' if http_pass else 'FAIL / NOT VALIDATED'}**. Resource verdict: **review required**.","",
        f"Commit: `{manifest.get('commit')}`; dirty: `{manifest.get('dirty')}`; UTC: `{manifest.get('utc')}`.",
        f"Source SHA-256: `{manifest.get('source_sha256')}`.",
        f"Runtime mode: **{provenance.get('runtime_mode','unknown')}**; model: `{provenance.get('model_version','unknown')}`; snapshot: `{provenance.get('forecast_snapshot_id','unknown')}`.",
        "Synthetic data / MatMul results do not establish trained-model performance or WAPE.","",
        "## Measurement window", "",f"Script: `{manifest['run']['script']}`. Target: {manifest['run']['rate']} RPS. Warm-up: {manifest['run']['warmup']}. Scheduled measurement: {duration}s.",
        "Authentication, setup and warm-up are excluded from business percentiles. Each measured iteration issues one business HTTP request. No retries.","",
        "| Metric | Value |", "|---|---:|",f"| Completed business requests | {count} |",f"| Successful business requests | {success} |",
        f"| Requests / scheduled second | {fmt(result['requests_per_scheduled_second'])} |",f"| Completed RPS, incl. final drain | {fmt(result['completed_rps_including_drain'])} |",
        f"| Successful RPS, incl. final drain | {fmt(result['successful_rps_including_drain'])} |",f"| p50 / p95 / p99 HTTP ms | {fmt(lat.get('med'))} / {fmt(lat.get('p(95)'))} / {fmt(lat.get('p(99)'))} |",
        f"| p95 client wall ms (incl. connection overhead) | {fmt(wall.get('p(95)'))} |",f"| Business error ratio (HTTP + semantic) | {fmt(errors,6)} |",f"| Dropped iterations | {drops} |",f"| k6 exit code | {code} |", "",
        "## Containers: measurement phase only", "", "CPU is Docker CPU% divided by the actual CPU quota in cores: 140% / 2 cores = 70% of the quota.",
        "Docker memory is its working-set-style reading, not process RSS. Native ONNX memory is included in process RSS below.", "",
        "| Container | CPU quota avg/max % | Docker memory max MiB | Samples |", "|---|---:|---:|---:|"]
    for entry in resources.values():
        cpu,mem=entry['cpu_quota_percent'],entry['docker_working_set_bytes']
        text.append(f"| {entry['name']} | {fmt(cpu.get('avg'))} / {fmt(cpu.get('max'))} | {fmt(mem.get('max',0)/1048576)} | {mem.get('count',0)} |")
    text += ["", "## Backend internals", "", "Approximate native/DB p95 uses differences of Prometheus histogram buckets between in-window scrapes. No observations means **not measured**, not 0 ms.", ""]
    for cid,entry in runtime.items():
        text += [f"### `{cid[:12]}`", f"RSS first/last/max MiB: {fmt(entry['rss_first_bytes']/1048576 if entry['rss_first_bytes'] is not None else None)} / {fmt(entry['rss_last_bytes']/1048576 if entry['rss_last_bytes'] is not None else None)} / {fmt(entry['rss_bytes'].get('max',0)/1048576)}.",
            f"Cache hourly-cell hit ratio: {fmt(entry['cache_cell_hit_ratio'],4)}. ONNX p95 estimate seconds: {fmt(entry['onnx_inference_p95_estimate_seconds'],6)}. DB p95 estimate seconds: {fmt(entry['db_query_p95_estimate_seconds'],6)}.",
            f"Swap observed: {entry['swap_observed']}; process max bytes: {entry['max_process_swap_bytes']}; cgroup max bytes: {entry['max_cgroup_swap_bytes']}.",""]
    text += ["## Required interpretation", "", "This is one measured point, not proof of maximum sustainable RPS. Review system.csv/metrics.ndjson for CPU headroom, GC, DB pool waiting, RSS plateau, restart/OOM and swap. Repeated 2/4-CPU sweeps and a 30-60 minute soak are separate runs.",
        "Shared-host contention and the load generator's CPU/memory must be disclosed. Absence of resource samples invalidates resource claims. Kernel page caches were not flushed.", "",
        "Artifacts: environment.json, bootstrap.json, k6-summary.json, k6.exitcode, system.csv, metrics.ndjson, raw.json.gz (when enabled), metrics-before/after, final-state.json, backend.log."]
    (directory/"benchmark.md").write_text("\n".join(text)+"\n")
    return result

if __name__=="__main__":
    parser=argparse.ArgumentParser();parser.add_argument("directory",type=Path);args=parser.parse_args();render(args.directory)
