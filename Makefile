SHELL := /bin/bash
.SHELLFLAGS := -eu -o pipefail -c
.DEFAULT_GOAL := help
RATE ?= 50
DURATION ?= 5m
WARMUP ?= 2m
REPLICAS ?= 1
BENCH_SCRIPT ?= forecast-day
RATES ?= 50,100,200,300,400

.PHONY: help init up down smoke benchmark sweep stress soak cold scale-check observability test-devops microbench profile reset-demo
help:
	@printf '%s\n' 'make up                 - build and start native SYNTHETIC demo' 'make smoke              - strict functional API checks, existing stack' 'make benchmark          - warmup + constant-arrival-rate + report' 'make sweep              - independent RPS steps; stops at first failure' 'make stress             - shorter stress steps up to 600 RPS' 'make soak RATE=150      - 45 minute mixed soak (choose measured rate first)' 'make cold               - RESTART one API; record readiness + first forecast' 'make scale-check        - two replicas + direct cross-replica session checks' 'make observability      - optional Prometheus / Grafana / postgres-exporter' 'make test-devops        - offline Python/Node tests' 'make microbench         - host Go CPU/allocation benchmarks, not HTTP RPS' 'make profile            - internal 30s CPU pprof, enable pprof explicitly'
init:
	python3 scripts/init_env.py
up: init
	docker compose up -d --build --wait --wait-timeout 300 --scale api=$(REPLICAS)
	docker compose up -d --no-deps --force-recreate gateway
	python3 scripts/wait-ready.py
down:
	docker compose down --remove-orphans
smoke: init
	python3 scripts/benchmark.py --script smoke --duration 30s --warmup 0s --replicas $(REPLICAS) --no-start
benchmark: init
	python3 scripts/benchmark.py --script $(BENCH_SCRIPT) --rate $(RATE) --duration $(DURATION) --warmup $(WARMUP) --replicas $(REPLICAS)
sweep: init
	python3 scripts/sweep.py --rates $(RATES) --script $(BENCH_SCRIPT) --duration $(DURATION) --warmup $(WARMUP) --replicas $(REPLICAS)
stress: init
	python3 scripts/sweep.py --rates 100,200,300,400,500,600 --script mixed-workload --duration 2m --warmup 30s --replicas $(REPLICAS)
soak: init
	python3 scripts/benchmark.py --script mixed-workload --rate $(RATE) --duration 45m --warmup 2m --replicas $(REPLICAS)
cold:
	python3 scripts/cold.py
scale-check:
	python3 scripts/scale_check.py
observability: init
	docker compose -f compose.yaml -f compose.observability.yaml up -d --build --wait --wait-timeout 300

microbench:
	out="benchmarks/results/microbench-$$(date -u +%Y%m%dT%H%M%SZ)"; mkdir -p "$$out"; cd backend; go test -run '^$$' -bench 'Benchmark(Calculate|Scenario|Cache|CSV|Observe)' -benchmem -count=5 ./internal/engine ./internal/httpapi ./internal/telemetry | tee "../$$out/go-bench.txt"
test-devops:
	python3 -m unittest discover -s scripts/tests -v
	node --test loadtest/tests/*.test.mjs
	@for file in loadtest/*.js loadtest/lib/*.js; do node --input-type=module --check < "$$file"; done
profile:
	python3 scripts/profile.py --kind cpu
reset-demo:
	@test "$(CONFIRM)" = DELETE_DEMO_DATA || (echo 'Destructive: use make reset-demo CONFIRM=DELETE_DEMO_DATA only for the disposable demo'; exit 1)
	docker compose down -v --remove-orphans
