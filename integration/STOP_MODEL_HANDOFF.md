# Real ML / stop data handoff

This integration does not train a model. No trained model or actual stop observations were present in either supplied archive.

## Route model

Keep the existing backend contract: `backend/docs/ML_HANDOFF.md`. Place immutable ONNX/schema/manifest/golden files under `backend/artifacts/` and a complete publisher bundle in `data/runtime-bundle.json`. Input features, observation cutoff, weather metadata, profiles and geometry must all match the pinned snapshot. Routes and stop occurrences come from that bundle, never from the demo UI.

## Stop model: offline predictions, real runtime storage

The newly connected adapter accepts **independent, precomputed hourly stop-model predictions**. It does NOT execute a second stop ONNX in the request path and does NOT split or repeat route predictions. The ML pipeline runs the stop model offline and publishes its outputs with the route snapshot.

Add to the existing bundle:

```json
{
  "stop_predictions": [{
    "route_id": "route-13",
    "route_pattern_id": "route-13-out",
    "route_stop_id": "route-13-out-002",
    "stop_id": "physical-stop-002",
    "sequence": 2,
    "time": "2026-10-01T00:00:00+03:00",
    "boardings": 42.0,
    "reference": 5.5,
    "typical": 40.0,
    "features_available_at": "2026-09-30T23:00:00+03:00"
  }]
}
```

This snippet is an example of the additional field, NOT a complete publishable bundle and NOT real transport data.

For each stop-enabled route, set `coverage.prediction_scopes=["route","route_stop"]`, `coverage.stop_forecast_status="experimental"` (or the validated status supported by OpenAPI), and both `snapshot.provenance.stop_model_version` / `stop_feature_schema_version`. Set actual route runtime mode to `operational` or `replay`, never relabel synthetic data. Use the permitted values in OpenAPI.

Publication requires exactly one row for every hour and every stop occurrence in that route's coverage. Identity must match the pinned PostGIS geometry, including direction, physical stop and sequence. `features_available_at <= forecast_origin_at`; the timestamp describes feature availability, not when the model finished predicting. Values must be finite and satisfy the existing numeric bounds. Missing/duplicate/out-of-coverage rows reject the candidate before activation.

`reference` is the stop-specific historical reference in **boardings per route vehicle-hour**. `typical` is the expected historical number of stop boardings for that hour. Either profile may be null: boardings still exist, but an unsupported index stays unavailable. Do not supply occupancy as boardings. Stop output sums are NOT automatically constrained to the route forecast; reconcile them in ML only if your model and target definitions justify it.

## Serving and scenarios

Migration 007 creates `prepared_stop_forecasts`. It is versioned by forecast snapshot and read from PostgreSQL, independently of a replica's cache. The same effective window/fleet/factor formulas apply to route and stop metrics. A fleet-only override cannot change either boarding prediction. A zero fleet produces an unavailable load index, not zero load.

Response: `RouteReading.stop_readings[]` in each frame AND totals. Summary and CSV use the same `calculation_id`. Bars around a stop remain decorative copies of one stop value; they are not independent segment measurements. `prediction_method=ml_stop_model` describes ML-supplied output; demo uses `synthetic_mock`.

## Bounded operation

- Public JSON requests: unchanged 64 KiB.
- One route calculation: <=744 hours; <=7440 route-hours in a request.
- One stop response: <=4096 stop/frame cells. Prefer daily resolution for a month.
- One trusted offline bundle: <=128 MiB and <=500000 stop-hour rows; route-feature limits remain unchanged. The publisher container has a 2 GiB limit, separately from API resources.
- CSV: <=8192 data rows and <=4 MiB; never silently truncated.
- Large incoming data must be reduced to the selected routes/coverage or published through a future staged bulk-import extension. Do not increase HTTP response limits to work around an oversized offline dataset.

## Publish

```bash
python3 scripts/start_real.py --bundle data/runtime-bundle.json
```

The script builds the real (non-demo) image, migrates, provisions the configured user, checks `expected_previous_snapshot`, publishes through the existing validator and starts the frontend/gateway/API. Existing snapshots are not overwritten in place. Use new immutable version IDs for changed artifacts/data. Re-running with the already-active snapshot ID does not republish changed bytes.

The script rotates the selected user's password to API_PASSWORD and revokes their old sessions by the existing backend rule. Maintain production credentials outside source control.
