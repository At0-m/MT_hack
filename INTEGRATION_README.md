# TramFlow: integrated backend + original frontend

Start here, not from the older branch-specific README files. The source integration is implemented; the independent verification scope is recorded in `INTEGRATION_REPORT.md`. Full Docker/browser/native ONNX testing must still be run on your machine or in the included CI.

## 1. Keep the original fonts, then launch

Extract this project into a NEW directory. Do not overwrite an active working deployment or delete its database volume.

The original interface styles, artwork, map and animations are retained. Font binaries are not redistributed in this package. Restore the six exact original fonts from the frontend ZIP you already have; the helper checks SHA-256 and does not download or substitute fonts:

```bash
cd MT_hack-integrated
python3 scripts/restore_fonts.py ../MT_hack-feature-front_fed.zip
COMPOSE_PROJECT_NAME=tramflow-integrated make up
```

Requirements: Docker Engine + Docker Compose v2, Python 3.10+, make; internet for initial image/dependency downloads and the optional map basemap. The Docker build installs frontend dependencies. Native ONNX runtime uses linux/amd64 as in the original backend branch.

Open **http://localhost:8080**. Login and password are `API_USER` and `API_PASSWORD` in the newly generated private root `.env`. They are not hardcoded or included in this package. The frontend uses the backend cookie session; refresh and logout use the real API.

Default startup is an explicitly **synthetic October 2026 demo**, with 10 artificial routes, prepared stop outputs and a tiny test ONNX, not a trained demand model. A badge is visible. This is useful for checking integration and must not be presented as real model quality or real Moscow geometry.

If port 8080 is occupied, stop the old stack WITHOUT `-v`, or choose another port and matching origin:

```bash
COMPOSE_PROJECT_NAME=tramflow-integrated HTTP_PORT=8088 CORS_ORIGIN=http://localhost:8088 make up
```

Keep the same COMPOSE_PROJECT_NAME/port/origin environment for subsequent commands. For public access explicitly configure BIND_ADDRESS, CORS_ORIGIN and TLS; COOKIE_SECURE must be true behind HTTPS. Do not disable origin validation to make deployment work.

Existing active snapshots are preserved. To get the new synthetic stop fixture, use a new Compose project as above; do not erase a real database. Migration 007 is applied by the publisher before API readiness.

## 2. Use the actual ML delivery instead of the demo

Neither input archive contained trained ONNX artifacts or a real prepared data bundle. Integration cannot manufacture them.

Place the real route model files in `backend/artifacts/` and the validated prepared bundle in `data/runtime-bundle.json`, following `backend/docs/ML_HANDOFF.md` and `integration/STOP_MODEL_HANDOFF.md`.

```bash
python3 scripts/start_real.py --bundle data/runtime-bundle.json
```

This uses `compose.yaml + compose.real.yaml`, not the synthetic seed command. It refuses synthetic data, checks publication CAS, does not replace immutable IDs and preserves the previous active snapshot on failed publication. For subsequent maintenance keep `COMPOSE_FILE=compose.yaml:compose.real.yaml` (colon on Linux/macOS).

Stop integration accepts real offline stop-model outputs, not a second request-time stop ONNX. Without stop outputs, route forecasts still work; stop values stay unavailable rather than becoming copied route values. See the exact stop grid, identity, units and limits in the handoff file.

## 3. What is connected

- Browser -> same-origin gateway -> Go API; static frontend is served by a separate non-root container.
- Real login/session/logout; no browser password/session-token storage.
- Route catalog, geometry and forecast snapshot come from the backend.
- Correct calendar month; week and custom use daily frames; custom is <=31 days, without silent clipping.
- The original service-day wheel 06:00-01:00 remains. The tools menu also offers all 24 calendar hours. At the end of coverage, all 24 valid hours of the selected day are shown instead of requesting an unavailable next day. Moving the wheel itself performs no forecast request.
- Fleet and weather/event/season coefficients use server-side scenarios. Omitted neutral factors and normalized timestamps do not cause false response rejection.
- Stop map columns and charts use independent server values keyed by route/direction/stop occurrence/time. Decorative bars remain frontend graphics, not extra predictions.
- Summary is loaded in the general analytics panel, below the original visual content. It uses the exact confirmed calculation and labels its full calendar period.
- CSV exports the exact confirmed calendar calculation; the button tooltip shows the actual period. Route-only columns are unchanged; stop-enabled exports add explicit spatial identity columns.
- Existing metrics/pprof/health/cross-replica scripts now have their backend endpoints/binaries. Metrics and optional pprof are private on port 9090, never routed through the public gateway.

## 4. Immediate verification on your machine

```bash
# After make up, inspect readiness and logs first:
docker compose ps -a
curl -f http://localhost:8080/health/ready

# Browser tests against the running backend (not the frontend mock):
corepack enable
corepack prepare pnpm@10.12.1 --activate
pnpm --dir frontend install --frozen-lockfile
(cd frontend && pnpm exec playwright install chromium)
python3 scripts/browser_check.py
```

Use the same COMPOSE_PROJECT_NAME environment when inspecting a separately named stack. Do not run another make up without the same project name and collide on its public port.

The browser test covers login, refresh persistence, local wheel, fleet scenario, summary identity, CSV, logout and month/daily framing. Its source is included; a successful local/CI run is still required before submission. A screenshot/trace is captured on failures.

Dependency-free DevOps tests: `make test-devops`. Full backend/native/PostGIS tests are in the updated backend CI. No RPS, p95, memory or model-quality numbers have been invented for this integration.

## 5. Fast troubleshooting

`seed exited 1`: inspect `docker compose logs seed db`; do not bypass publication validation. Missing features, incompatible ONNX signatures, wrong version/CAS or incomplete stop grids must be fixed in the bundle.

`403 ORIGIN_FORBIDDEN`: open exactly CORS_ORIGIN including scheme/host/port, or configure it correctly. `localhost` and `127.0.0.1` are different origins.

No map basemap: a local fallback keeps the supplied route geometry visible. It cannot generate roads or buildings when the tile provider is unreachable.

Grey/no stop columns: confirm `stop_forecasts`, route-specific `prediction_scopes` and the published stop grid. Do not replace missing stop outputs with route averages in the frontend.

Old demo still active: expected, it is not silently overwritten. Use a fresh project or publish a new version intentionally.

Missing fonts: rerun restore_fonts.py with the original frontend ZIP. Do not substitute fonts just to make the design approximately match.
# Изменения для критерия коэффициентов

Автоматический пересчёт поправок и воспроизводимый импорт 15-минутных CSV описаны в [docs/CRITERION_COEFFICIENTS.md](docs/CRITERION_COEFFICIENTS.md). Там же результаты сборки, тестов и порядок демонстрации. Основной запуск объединённого проекта описан ниже.

