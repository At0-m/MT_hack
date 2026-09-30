export interface paths {
    "/health/live": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * HTTP process liveness
         * @description HTTP process liveness
         */
        get: operations["getLiveness"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/health/ready": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * PostgreSQL, active snapshot, ONNX adapter and initial weather/fallback readiness
         * @description 200 when PostgreSQL is reachable, an active fully validated snapshot is readable, referenced immutable ML artifact is loadable, and model session is ready. Does not require external Open-Meteo or basemap availability per request.
         */
        get: operations["getReadiness"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/login": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Create browser session from login form
         * @description Validates username/password against the configured user store, creates a server-side session in PostgreSQL and sets tramflow_session. Do not return password, password hash or reusable bearer token.
         */
        post: operations["login"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/session": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read current browser session
         * @description Called on application startup/refresh. 200 means the existing HttpOnly cookie maps to an active server-side session; 401 means show the login form.
         */
        get: operations["getAuthSession"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/auth/logout": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Revoke current browser session
         * @description Deletes/revokes the server-side session and expires the cookie. Idempotent for an authenticated request.
         */
        post: operations["logout"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/bootstrap": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Get pinned snapshot, defaults and UI capabilities
         * @description Cache-Control:no-store. Active snapshot and limits come from authoritative PostgreSQL registry. Use returned default_selection; never infer dates or route IDs from examples.
         */
        get: operations["getBootstrap"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/snapshots/{forecast_snapshot_id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read immutable forecast snapshot metadata
         * @description Cache-Control:private,no-cache. Keep previous snapshots at least the advertised retention after deactivation.
         */
        get: operations["getSnapshot"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/routes": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * List the ten supported route catalog entries
         * @description List the ten supported route catalog entries
         */
        get: operations["listRoutes"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/routes/{route_id}": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Get route directions and geographic stop catalog
         * @description Get route directions and geographic stop catalog
         */
        get: operations["getRoute"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/routes/{route_id}/geometry": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read local, versioned route GeoJSON
         * @description Reads versioned route/stop geometry from PostgreSQL/PostGIS and serializes GeoJSON. No Overpass call on this runtime path. Unsupported/missing geometry returns 404 GEOMETRY_UNAVAILABLE.
         */
        get: operations["getRouteGeometry"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/forecasts/query": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Batch forecast route boardings and optional route-stop boardings
         * @description Runs the pinned forecast model(s), formula metrics and aggregation for the exact selection. spatial_detail=route_stop returns first-class stop_readings keyed by route_pattern_id + route_stop_id. The API never fabricates multiple approach samples for visual effects. Day responses return hourly frames; week/month/custom return daily frames.
         */
        post: operations["queryForecast"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/scenarios/evaluate": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Evaluate formula-based fleet/manual-demand scenario for route and optional stop forecasts
         * @description Pure read. Reuses Model A boardings. Changing fleet or manual multipliers never retrains/reruns Model A when pinned baseline is cached. Formulas apply hourly within effective_window, then aggregate. No real dispatch operation. Negative resulting fleet or unknown baseline for a delta ->422.
         */
        post: operations["evaluateScenario"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/weather": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read pinned weather features and provenance
         * @description Return hourly values actually pinned by snapshot, including marked climatology or missing values. Bounds aligned to hours, <=744h and within route coverage. Never proxy arbitrary coordinates or a user-supplied URL.
         */
        get: operations["getWeather"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/model-quality": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read measured or explicitly unmeasured quality records
         * @description Read measured or explicitly unmeasured quality records
         */
        get: operations["getModelQuality"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/data-sources": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        /**
         * Read data provenance and usage restrictions
         * @description Read data provenance and usage restrictions
         */
        get: operations["getDataSources"];
        put?: never;
        post?: never;
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/summaries/query": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Build deterministic text summary for an exact calculation and period
         * @description Synchronous bounded summary. No LLM is required. Backend derives wording from already computed metrics/indicators and must not claim causal influence. Frontend loading state is the in-flight HTTP state; ready/unavailable are response states. expected_calculation_id mismatch ->409.
         */
        post: operations["querySummary"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
    "/api/v1/exports": {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        get?: never;
        put?: never;
        /**
         * Export the exact confirmed calculation as CSV
         * @description Synchronous CSV, not a task. Complete validation and bounded buffering before 200. UTF-8 BOM, semicolon delimiter, decimal point, CRLF, RFC4180 quoting, null as empty cell. Exact columns in docs/CSV.md. Hash is returned in X-Calculation-ID; user export is NOT competition submission.
         */
        post: operations["exportCalculation"];
        delete?: never;
        options?: never;
        head?: never;
        patch?: never;
        trace?: never;
    };
}
export type webhooks = Record<string, never>;
export interface components {
    schemas: {
        /** @description Opaque stable identifier; route numbers are strings, not integers. */
        Id: string;
        /** @description SHA-256 of the canonical computation descriptor; NOT a persisted resource ID. See docs/CALCULATIONS.md. */
        CalculationId: string;
        /** @description Half-open [from,to), RFC3339 with explicit timezone. from < to is runtime validation. Day-resolution selections must use Europe/Moscow local-midnight bounds; day view uses one local day and hour resolution. */
        TimeWindow: {
            /** Format: date-time */
            from: string;
            /** Format: date-time */
            to: string;
        };
        /** @enum {string} */
        ViewMode: "day" | "week" | "month" | "custom";
        /** @enum {string} */
        Resolution: "hour" | "day";
        /** @description View-mode-specific selection. custom never supports hourly resolution; use day view + local wheel for hourly exploration. */
        ForecastSelection: components["schemas"]["DayForecastSelection"] | components["schemas"]["WeekForecastSelection"] | components["schemas"]["MonthForecastSelection"] | components["schemas"]["CustomForecastSelection"];
        ForecastQuery: {
            selection: components["schemas"]["ForecastSelection"];
        };
        /** @description Additional manual multipliers on model boardings, NOT physical weather inputs and NOT causal effects. Missing values default to 1. Precision <=3 decimals. Product must be between 0.25 and 4 inclusive (runtime). */
        Factors: {
            /**
             * Format: double
             * @default 1
             */
            weather: number;
            /**
             * Format: double
             * @default 1
             */
            event: number;
            /**
             * Format: double
             * @default 1
             */
            season: number;
        };
        /** @description Assumed effective fleet in every affected hourly bucket, not departures. 0 is a valid no-supply scenario. */
        AbsoluteFleet: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "absolute";
            /** Format: int32 */
            vehicle_count: number;
        };
        /** @description Add to each hourly baseline fleet. Baseline must be known in every affected cell; result must be 0..200. Values are not clamped. */
        DeltaFleet: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "delta";
            /** Format: int32 */
            vehicle_count_delta: number;
        };
        FleetOverride: components["schemas"]["AbsoluteFleet"] | components["schemas"]["DeltaFleet"];
        /** @description Effective window must be contained in selection.window. Applies to all selection.route_ids, before temporal aggregation. Outside this window use baseline. Scenario is relative to original snapshot, never to a previous scenario. */
        ScenarioOverrides: {
            effective_window: components["schemas"]["TimeWindow"];
            fleet?: components["schemas"]["FleetOverride"];
            factors?: components["schemas"]["Factors"];
        } | {
            effective_window: components["schemas"]["TimeWindow"];
            fleet: components["schemas"]["FleetOverride"];
            factors?: components["schemas"]["Factors"];
        } | {
            effective_window: components["schemas"]["TimeWindow"];
            fleet?: components["schemas"]["FleetOverride"];
            factors: components["schemas"]["Factors"];
        };
        ScenarioQuery: {
            selection: components["schemas"]["ForecastSelection"];
            overrides: components["schemas"]["ScenarioOverrides"];
        };
        ForecastCalculation: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "forecast";
            selection: components["schemas"]["ForecastSelection"];
        };
        ScenarioCalculation: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "scenario";
            selection: components["schemas"]["ForecastSelection"];
            overrides: components["schemas"]["ScenarioOverrides"];
        };
        CalculationDescriptor: components["schemas"]["ForecastCalculation"] | components["schemas"]["ScenarioCalculation"];
        /** @enum {string} */
        UndefinedReason: "SUPPLY_UNKNOWN" | "NO_SUPPLY" | "REFERENCE_MISSING" | "REFERENCE_ZERO" | "BASELINE_ZERO" | "NO_TYPICAL_PROFILE" | "NO_MEASUREMENT" | "INCOMPLETE_COVERAGE";
        AvailableNumber: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "available";
            /** Format: double */
            value: number;
        };
        AvailableInteger: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "available";
            /** Format: int32 */
            value: number;
        };
        UnavailableNumber: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "unavailable";
            /** @enum {number|null} */
            value: null;
            reason: components["schemas"]["UndefinedReason"];
        };
        NumericReading: components["schemas"]["AvailableNumber"] | components["schemas"]["UnavailableNumber"];
        IntegerReading: components["schemas"]["AvailableInteger"] | components["schemas"]["UnavailableNumber"];
        /** @enum {string} */
        LoadLevel: "normal" | "elevated" | "high" | "very_high" | "unavailable";
        /** @description All formulas in docs/CALCULATIONS.md. Load index is relative to an empirical historical reference, NOT cabin occupancy. Ratios are recomputed from aggregate numerators/denominators. required_vehicle_count is max hourly hypothetical requirement at snapshot target_index, NOT a dispatch recommendation. */
        MetricSet: {
            /**
             * Format: double
             * @description Expected count, may be fractional. Sum hourly predictions; do not round before aggregation.
             */
            boardings: number;
            vehicle_hours: components["schemas"]["NumericReading"];
            mean_vehicle_count: components["schemas"]["NumericReading"];
            boardings_per_vehicle_hour: components["schemas"]["NumericReading"];
            reference_boardings: components["schemas"]["NumericReading"];
            load_index: components["schemas"]["NumericReading"];
            load_level: components["schemas"]["LoadLevel"];
            required_vehicle_count: components["schemas"]["IntegerReading"];
            /** @enum {string} */
            fleet_source: "observed_vehicle_profile" | "manual_plan" | "scenario_absolute" | "scenario_delta" | "mixed" | "unavailable";
            fleet_is_proxy: boolean;
        };
        /** @description Derived from the pinned weather snapshot. If any hour is missing, affected aggregate numeric fields are null, not a partial sum. */
        WeatherSummary: {
            /** @enum {string} */
            mode: "forecast" | "climatology" | "mixed" | "missing";
            /** Format: double */
            temperature_mean_c: number | null;
            /** Format: double */
            precipitation_sum_mm: number | null;
            /** Format: int32 */
            forecast_hours: number;
            /** Format: int32 */
            climatology_hours: number;
            /** Format: int32 */
            missing_hours: number;
        };
        /** @description evaluated - baseline. An undefined operand produces null. Load-index delta is an absolute ratio difference, not a relative percent. */
        MetricDelta: {
            /** Format: double */
            boardings: number;
            /** Format: double */
            load_index: number | null;
            /** Format: double */
            vehicle_hours: number | null;
        };
        /** @description No scenario: evaluated equals baseline. relative_to_typical uses evaluated.boardings; denominator 0 or absent -> null. stop_readings are real stop-level forecasts when requested and supported; visual approach effects are frontend-only. */
        RouteReading: {
            route_id: components["schemas"]["Id"];
            typical_boardings: components["schemas"]["NumericReading"];
            /** Format: double */
            relative_to_typical: number | null;
            baseline: components["schemas"]["MetricSet"];
            evaluated: components["schemas"]["MetricSet"];
            delta: components["schemas"]["MetricDelta"];
            weather: components["schemas"]["WeatherSummary"];
            indicators: components["schemas"]["UiIndicator"][];
            /** @description Populated only for spatial_detail=route_stop; empty for route-only calculations. */
            stop_readings: components["schemas"]["StopReading"][];
        };
        /** @description Frames are contiguous, non-overlapping and sorted; routes sorted by route_id. Day view frames are hourly; week/month/custom frames are daily. Stop readings inherit the enclosing frame window. */
        Frame: {
            window: components["schemas"]["TimeWindow"];
            routes: components["schemas"]["RouteReading"][];
        };
        /** @description Stable visualization policy across frames/scenarios. Numeric index is never clamped. Thresholds are product policy, not capacity standards. stop_columns describes frontend rendering only and does not create extra analytical values. */
        VisualizationPolicy: {
            /** @enum {string} */
            metric: "load_index";
            /** Format: double */
            scale_min: number;
            /** Format: double */
            scale_max: number;
            /** @enum {string} */
            overflow: "show_overflow";
            thresholds: number[];
            reference_version: components["schemas"]["Id"];
            label: string;
            stop_columns: components["schemas"]["StopVisualizationPolicy"];
        };
        Notice: {
            code: string;
            message: string;
            /** @enum {string} */
            severity: "info" | "warning";
        };
        Provenance: {
            forecast_snapshot_id: components["schemas"]["Id"];
            /** Format: date-time */
            forecast_origin_at: string;
            /** Format: date-time */
            observations_complete_through: string;
            model_version: components["schemas"]["Id"];
            feature_schema_version: components["schemas"]["Id"];
            history_version: components["schemas"]["Id"];
            weather_snapshot_id: components["schemas"]["Id"];
            supply_profile_version: components["schemas"]["Id"];
            reference_version: components["schemas"]["Id"];
            scenario_policy_version: components["schemas"]["Id"];
            network_version: components["schemas"]["Id"];
            /** @enum {string} */
            runtime_mode: "replay" | "operational" | "synthetic_mock";
            /** @description Optional. Required when any coverage exposes route_stop forecasts. */
            stop_model_version?: components["schemas"]["Id"];
            /** @description Optional. Required when stop_model_version is present. */
            stop_feature_schema_version?: components["schemas"]["Id"];
        };
        /** @description Stateless read result. totals cover the exact request window per route, with correctly recomputed ratios. No cross-route occupancy sum. All numeric outputs finite. Context stays pinned until explicit refresh. */
        CalculationResponse: {
            calculation_id: components["schemas"]["CalculationId"];
            descriptor: components["schemas"]["CalculationDescriptor"];
            provenance: components["schemas"]["Provenance"];
            /** Format: double */
            target_load_index: number;
            frames: components["schemas"]["Frame"][];
            totals: components["schemas"]["RouteReading"][];
            visualization: components["schemas"]["VisualizationPolicy"];
            notices: components["schemas"]["Notice"][];
        };
        RouteCoverage: {
            route_id: components["schemas"]["Id"];
            window: components["schemas"]["TimeWindow"];
            resolutions: components["schemas"]["Resolution"][];
            /** Format: int32 */
            max_lead_hours: number;
            /** @enum {string} */
            status: "validated" | "experimental";
            prediction_scopes: ("route" | "route_stop")[];
            /**
             * @description route_stop may appear in prediction_scopes only when status is experimental or validated.
             * @enum {string}
             */
            stop_forecast_status: "unavailable" | "experimental" | "validated";
        };
        /** @description Immutable data/model/weather/supply tuple. An update creates a NEW ID; old requests never silently switch to current. Coverage is a server declaration, not inferred from dataset filenames. */
        Snapshot: {
            provenance: components["schemas"]["Provenance"];
            /** Format: date-time */
            created_at: string;
            coverage: components["schemas"]["RouteCoverage"][];
            view_modes: components["schemas"]["ViewMode"][];
            /** Format: double */
            reference_quantile: number;
            /** Format: double */
            target_load_index: number;
            /** Format: int32 */
            retention_hours_after_deactivation: number;
            limitations: components["schemas"]["Notice"][];
        };
        Position: {
            /** Format: double */
            longitude: number;
            /** Format: double */
            latitude: number;
        };
        RouteSummary: {
            route_id: components["schemas"]["Id"];
            name: string;
            route_number: string;
            /** @enum {string} */
            geometry_status: "verified" | "approximate" | "unavailable";
            representative_position?: components["schemas"]["Position"];
        };
        /** @description Geographic route-pattern stop. Numeric map bars are delivered with calculation frames and explicitly declare their metric_scope/method; this catalog object itself carries no forecast. */
        RouteStop: {
            route_stop_id: components["schemas"]["Id"];
            stop_id: components["schemas"]["Id"];
            /** Format: int32 */
            sequence: number;
            name: string;
            position: components["schemas"]["Position"];
        };
        /** @description A route direction/variant. Repeated physical stops have distinct route_stop_id/sequence. Numeric route forecasts are NOT split by pattern. */
        RoutePattern: {
            route_pattern_id: components["schemas"]["Id"];
            name: string;
            stops: components["schemas"]["RouteStop"][];
        };
        RouteDetail: {
            route: components["schemas"]["RouteSummary"];
            network_version: components["schemas"]["Id"];
            patterns: components["schemas"]["RoutePattern"][];
            source_id: components["schemas"]["Id"];
            /** Format: date-time */
            geometry_observed_at: string;
            /** @enum {string} */
            historical_match_status: "verified" | "unverified";
            notices: components["schemas"]["Notice"][];
        };
        RouteList: {
            network_version: components["schemas"]["Id"];
            items: components["schemas"]["RouteSummary"][];
        };
        LineString: {
            /** @enum {string} */
            type: "LineString";
            coordinates: number[][];
        };
        MultiLineString: {
            /** @enum {string} */
            type: "MultiLineString";
            coordinates: number[][][];
        };
        Point: {
            /** @enum {string} */
            type: "Point";
            /** @description [longitude,latitude], WGS84; ranges checked by geometry importer. */
            coordinates: number[];
        };
        LineGeometry: components["schemas"]["LineString"] | components["schemas"]["MultiLineString"];
        RouteLineFeature: {
            /** @enum {string} */
            type: "Feature";
            id: components["schemas"]["Id"];
            geometry: components["schemas"]["LineGeometry"];
            properties: {
                /** @enum {string} */
                kind: "route_line";
                route_id: components["schemas"]["Id"];
                route_pattern_id: components["schemas"]["Id"];
            };
        };
        StopPointFeature: {
            /** @enum {string} */
            type: "Feature";
            id: components["schemas"]["Id"];
            geometry: components["schemas"]["Point"];
            properties: {
                /** @enum {string} */
                kind: "stop";
                route_id: components["schemas"]["Id"];
                route_pattern_id: components["schemas"]["Id"];
                route_stop_id: components["schemas"]["Id"];
                stop_id: components["schemas"]["Id"];
                name: string;
                /** Format: int32 */
                sequence: number;
            };
        };
        GeometryFeature: components["schemas"]["RouteLineFeature"] | components["schemas"]["StopPointFeature"];
        /** @description GeoJSON with documented foreign members. RouteStopFeature.id equals route_stop_id; route-line id equals route_pattern_id. Catalog geography only; no forecast values. */
        RouteGeometry: {
            /** @enum {string} */
            type: "FeatureCollection";
            network_version: components["schemas"]["Id"];
            route_id: components["schemas"]["Id"];
            source_id: components["schemas"]["Id"];
            features: components["schemas"]["GeometryFeature"][];
        };
        Limits: {
            /** Format: int32 */
            max_routes: number;
            /** Format: int32 */
            max_window_hours: number;
            /** Format: int32 */
            max_hourly_cells: number;
            /** Format: int32 */
            max_body_bytes: number;
            /** Format: int32 */
            max_export_rows: number;
            /** Format: int32 */
            max_export_bytes: number;
            /** Format: int32 */
            forecast_timeout_ms: number;
            /** Format: int32 */
            max_concurrent_inference: number;
            max_response_cells: number;
            /**
             * Format: int32
             * @description Maximum custom day-resolution span advertised by this deployment. Frontend must read this value; no silent clipping.
             */
            max_custom_days: number;
            /**
             * Format: int32
             * @description Maximum route-stop frame cells in one response.
             */
            max_stop_cells: number;
        };
        /** @description External basemap availability is not guaranteed. Local fallback shows route geography without basemap. Attribution must stay visible. */
        MapConfiguration: {
            /** @enum {string} */
            renderer: "maplibre";
            /** Format: uri */
            style_url: string;
            fallback_style_path: string;
            attribution: string;
            center: components["schemas"]["Position"];
            /** Format: double */
            zoom: number;
        };
        /** @description Capabilities describe actual active-snapshot support. route_stop means first-class stop predictions, not repeated route values. The approach visual is decorative and never an analytical segment forecast. */
        Capabilities: {
            forecast_scopes: ("route" | "route_stop")[];
            scenario_fleet: boolean;
            scenario_factors: ("weather" | "event" | "season")[];
            /** @enum {boolean} */
            physical_weather_overrides: false;
            csv_export: boolean;
            /** @enum {boolean} */
            live_ingestion: false;
            /** @enum {boolean} */
            summary: true;
            /** @enum {boolean} */
            custom_period: true;
            /** @enum {boolean} */
            session_auth: true;
            /** @description True when the active snapshot provides real stop-level forecast artifacts/coverage. */
            stop_forecasts: boolean;
            /**
             * @description v1.2 does not define analytical segment forecasts; the approach ramp is decorative frontend rendering.
             * @enum {boolean}
             */
            segment_forecasts: false;
            /** @enum {boolean} */
            decorative_approach_effect: true;
        };
        Bootstrap: {
            api_version: string;
            /** @enum {string} */
            display_timezone: "Europe/Moscow";
            /** @enum {string} */
            locale: "ru-RU";
            active_snapshot: components["schemas"]["Snapshot"];
            routes: components["schemas"]["RouteSummary"][];
            default_selection: components["schemas"]["ForecastSelection"];
            map: components["schemas"]["MapConfiguration"];
            limits: components["schemas"]["Limits"];
            capabilities: components["schemas"]["Capabilities"];
        };
        /** @description For climatology weather_code and provider_run_at are null. Missing mode has null weather values. available_at is information availability, not download time. Precipitation must be aligned to the interval; provider backward-looking timestamps are normalized by the adapter. */
        WeatherPoint: {
            window: components["schemas"]["TimeWindow"];
            /** @enum {string} */
            mode: "forecast" | "climatology" | "missing";
            /** Format: double */
            temperature_c: number | null;
            /** Format: double */
            precipitation_mm: number | null;
            weather_code: number | null;
            /** Format: date-time */
            provider_run_at: string | null;
            /** Format: date-time */
            available_at: string | null;
            availability_verified: boolean;
        };
        /** @description Pinned weather actually used by the computation. Does not call Open-Meteo synchronously, and is not an arbitrary URL/coordinate proxy. */
        WeatherResponse: {
            weather_snapshot_id: components["schemas"]["Id"];
            forecast_snapshot_id: components["schemas"]["Id"];
            route_id: components["schemas"]["Id"];
            location_id: components["schemas"]["Id"];
            /** @enum {string} */
            provider: "open-meteo" | "synthetic_mock";
            /** Format: date-time */
            retrieved_at: string;
            source_id: components["schemas"]["Id"];
            window: components["schemas"]["TimeWindow"];
            points: components["schemas"]["WeatherPoint"][];
            notices: components["schemas"]["Notice"][];
        };
        Source: {
            source_id: components["schemas"]["Id"];
            name: string;
            /** @enum {string} */
            category: "organizer" | "weather" | "calendar" | "geometry";
            /** Format: uri */
            url?: string;
            retrieval_method?: string;
            version: components["schemas"]["Id"];
            license_note: string;
            /** Format: date-time */
            retrieved_at: string;
            used_for_model: boolean;
            limitations: string[];
        } | unknown | unknown;
        SourceList: {
            forecast_snapshot_id: components["schemas"]["Id"];
            sources: components["schemas"]["Source"][];
        };
        ValidationRecord: components["schemas"]["MeasuredValidationRecord"] | components["schemas"]["UnmeasuredValidationRecord"];
        SourceEffect: components["schemas"]["MeasuredSourceEffect"] | components["schemas"]["UnmeasuredSourceEffect"];
        /** @description Quality records are scoped per prediction level. Competition WAPE is route/hour unless organizers define otherwise; stop-model quality is reported separately. */
        QualityResponse: {
            forecast_snapshot_id: components["schemas"]["Id"];
            /** @enum {string} */
            target: "boardings";
            metrics: components["schemas"]["ValidationRecord"][];
            source_effects: components["schemas"]["SourceEffect"][];
            limitations: string[];
        };
        /** @description Full descriptor is replayed. Expected hash mismatch ->409. Does not depend on a process-local forecast_id or saved evaluation. Synchronous, bounded CSV; all validation happens before sending CSV headers. */
        ExportRequest: {
            calculation: components["schemas"]["CalculationDescriptor"];
            expected_calculation_id: components["schemas"]["CalculationId"];
            /** @enum {string} */
            format: "csv";
        };
        ProblemField: {
            pointer: string;
            code: string;
            message: string;
        };
        /** @description application/problem+json. Stable code for logic; human text in Russian. No raw records, card hashes, internals or secrets. */
        Problem: {
            /** Format: uri-reference */
            type: string;
            title: string;
            /** Format: int32 */
            status: number;
            detail: string;
            code: string;
            request_id: string;
            errors?: components["schemas"]["ProblemField"][];
        };
        Health: {
            /** @enum {string} */
            status: "ok" | "not_ready";
            /** @enum {string} */
            component: "api";
            checks: {
                name: string;
                ok: boolean;
            }[];
        };
        /** @description wape_score=max(0,1-sum_absolute_error/sum_actual). Zero actual sum -> null. not_measured -> scores null, count/sums 0. Exact competition protocol still must be confirmed; supplied headers do not establish it. */
        MeasuredValidationRecord: {
            evaluation_id: components["schemas"]["Id"];
            protocol_id: components["schemas"]["Id"];
            model_version: components["schemas"]["Id"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "measured";
            route_ids: components["schemas"]["Id"][];
            window: components["schemas"]["TimeWindow"];
            /** Format: int32 */
            lead_hours_from: number;
            /** Format: int32 */
            lead_hours_to: number;
            /** Format: int32 */
            sample_count: number;
            /** Format: double */
            sum_actual: number;
            /** Format: double */
            sum_absolute_error: number;
            /** Format: double */
            wape_score: number | null;
            /** Format: double */
            baseline_wape_score: number | null;
            notes: string[];
            /** @enum {string} */
            prediction_scope: "route" | "route_stop";
            /** @enum {string} */
            unit: "boardings_per_route_hour" | "boardings_per_route_stop_hour";
        };
        /** @description wape_score=max(0,1-sum_absolute_error/sum_actual). Zero actual sum -> null. not_measured -> scores null, count/sums 0. Exact competition protocol still must be confirmed; supplied headers do not establish it. */
        UnmeasuredValidationRecord: {
            evaluation_id: components["schemas"]["Id"];
            protocol_id: components["schemas"]["Id"];
            model_version: components["schemas"]["Id"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "not_measured";
            route_ids: components["schemas"]["Id"][];
            window: components["schemas"]["TimeWindow"];
            /** Format: int32 */
            lead_hours_from: number;
            /** Format: int32 */
            lead_hours_to: number;
            /**
             * Format: int32
             * @enum {integer}
             */
            sample_count: 0;
            /**
             * Format: double
             * @enum {number}
             */
            sum_actual: 0;
            /**
             * Format: double
             * @enum {number}
             */
            sum_absolute_error: 0;
            /** @enum {number|null} */
            wape_score: null;
            /** @enum {number|null} */
            baseline_wape_score: null;
            notes: string[];
            /** @enum {string} */
            prediction_scope: "route" | "route_stop";
            /** @enum {string} */
            unit: "boardings_per_route_hour" | "boardings_per_route_stop_hour";
        };
        /** @description Comparison must use same splits, horizon, target and other settings. No invented improvements; not_measured -> numeric values null. */
        MeasuredSourceEffect: {
            source_id: components["schemas"]["Id"];
            protocol_id: components["schemas"]["Id"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "measured";
            /** Format: double */
            score_without: number;
            /** Format: double */
            score_with: number;
            /** Format: double */
            delta: number;
            notes: string;
        };
        /** @description Comparison must use same splits, horizon, target and other settings. No invented improvements; not_measured -> numeric values null. */
        UnmeasuredSourceEffect: {
            source_id: components["schemas"]["Id"];
            protocol_id: components["schemas"]["Id"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "not_measured";
            /** @enum {number|null} */
            score_without: null;
            /** @enum {number|null} */
            score_with: null;
            /** @enum {number|null} */
            delta: null;
            notes: string;
        };
        LoginRequest: {
            username: string;
            /** Format: password */
            password: string;
        };
        UserInfo: {
            user_id: components["schemas"]["Id"];
            username: string;
            display_name: string;
        };
        /** @description Browser refresh keeps the login because the opaque session is carried by the HttpOnly cookie. The cookie is not exposed to JavaScript. */
        AuthSession: {
            /** @enum {boolean} */
            authenticated: true;
            user: components["schemas"]["UserInfo"];
            /** Format: date-time */
            expires_at: string;
        };
        /** @enum {string} */
        IndicatorTone: "neutral" | "info" | "positive" | "warning" | "critical";
        /** @enum {string} */
        IndicatorIcon: "none" | "weather" | "fleet" | "trend" | "peak" | "event" | "calendar";
        /** @description Backend-owned semantic indicator. Frontend maps (key, variant, tone) to agreed design tokens and never parses title/text to choose an icon or infer causality. Backend validates compatible key/variant pairs. Neutral state uses variant=none and icon=none. */
        UiIndicator: {
            /** @enum {string} */
            key: "weather" | "fleet" | "trend" | "peak" | "event" | "calendar";
            tone: components["schemas"]["IndicatorTone"];
            icon: components["schemas"]["IndicatorIcon"];
            title: string;
            text: string;
            variant: components["schemas"]["IndicatorVariant"];
        };
        SummaryFocus: components["schemas"]["OverallSummaryFocus"] | components["schemas"]["RouteSummaryFocus"] | components["schemas"]["RouteStopSummaryFocus"];
        /** @description Replays the exact calculation descriptor and verifies expected_calculation_id. Summary period is exactly calculation.selection.window; no implicit expansion or current-state lookup. */
        SummaryQuery: {
            calculation: components["schemas"]["CalculationDescriptor"];
            expected_calculation_id: components["schemas"]["CalculationId"];
            focus: components["schemas"]["SummaryFocus"];
        };
        SummaryFact: {
            /** @enum {string} */
            key: "peak" | "boardings" | "load" | "fleet" | "scenario_delta" | "weather" | "typical_delta";
            text: string;
        };
        SummaryReady: {
            calculation_id: components["schemas"]["CalculationId"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "ready";
            focus: components["schemas"]["SummaryFocus"];
            window: components["schemas"]["TimeWindow"];
            text: string;
            facts: components["schemas"]["SummaryFact"][];
        };
        SummaryUnavailable: {
            calculation_id: components["schemas"]["CalculationId"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            status: "unavailable";
            focus: components["schemas"]["SummaryFocus"];
            window: components["schemas"]["TimeWindow"];
            /** @enum {string} */
            reason: "INSUFFICIENT_DATA" | "FOCUS_NOT_SUPPORTED" | "NO_TEXT_AVAILABLE";
        };
        SummaryResponse: components["schemas"]["SummaryReady"] | components["schemas"]["SummaryUnavailable"];
        OverallSummaryFocus: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "overall";
        };
        RouteSummaryFocus: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "route";
            route_id: components["schemas"]["Id"];
        };
        RouteStopSummaryFocus: {
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            kind: "route_stop";
            route_id: components["schemas"]["Id"];
            route_pattern_id: components["schemas"]["Id"];
            route_stop_id: components["schemas"]["Id"];
        };
        /**
         * @description route_stop requests real stop-level forecasts. Backend returns 422 when the selected snapshot does not provide stop forecasts.
         * @enum {string}
         */
        SpatialDetail: "route" | "route_stop";
        /** @description Exactly one Europe/Moscow local calendar day; backend returns hourly frames. */
        DayForecastSelection: {
            forecast_snapshot_id: components["schemas"]["Id"];
            route_ids: components["schemas"]["Id"][];
            window: components["schemas"]["TimeWindow"];
            spatial_detail: components["schemas"]["SpatialDetail"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            view_mode: "day";
            /** @enum {string} */
            resolution: "hour";
        };
        /** @description Exactly seven consecutive Europe/Moscow local calendar days; backend returns one frame per day. */
        WeekForecastSelection: {
            forecast_snapshot_id: components["schemas"]["Id"];
            route_ids: components["schemas"]["Id"][];
            window: components["schemas"]["TimeWindow"];
            spatial_detail: components["schemas"]["SpatialDetail"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            view_mode: "week";
            /** @enum {string} */
            resolution: "day";
        };
        /** @description Exactly one Europe/Moscow local calendar month; backend returns one frame per calendar day. */
        MonthForecastSelection: {
            forecast_snapshot_id: components["schemas"]["Id"];
            route_ids: components["schemas"]["Id"][];
            window: components["schemas"]["TimeWindow"];
            spatial_detail: components["schemas"]["SpatialDetail"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            view_mode: "month";
            /** @enum {string} */
            resolution: "day";
        };
        /** @description Arbitrary local-date range at day resolution only. Bounds are Europe/Moscow local midnights, may cross months/years/leap day, and duration must not exceed limits.max_custom_days. */
        CustomForecastSelection: {
            forecast_snapshot_id: components["schemas"]["Id"];
            route_ids: components["schemas"]["Id"][];
            window: components["schemas"]["TimeWindow"];
            spatial_detail: components["schemas"]["SpatialDetail"];
            /**
             * @description discriminator enum property added by openapi-typescript
             * @enum {string}
             */
            view_mode: "custom";
            /** @enum {string} */
            resolution: "day";
        };
        /**
         * @description ml_stop_model means a real separately versioned stop-level ML prediction; synthetic_mock is development only.
         * @enum {string}
         */
        StopPredictionMethod: "ml_stop_model" | "synthetic_mock";
        /** @description First-class stop-level forecast for the enclosing frame. It is not a copy of the route value. Geometry is joined by route_stop_id using RouteGeometry. Stop load_index is a normalized boarding-intensity metric, not cabin occupancy. */
        StopReading: {
            route_id: components["schemas"]["Id"];
            route_pattern_id: components["schemas"]["Id"];
            route_stop_id: components["schemas"]["Id"];
            stop_id: components["schemas"]["Id"];
            /** Format: int32 */
            sequence: number;
            prediction_method: components["schemas"]["StopPredictionMethod"];
            typical_boardings: components["schemas"]["NumericReading"];
            /** Format: double */
            relative_to_typical: number | null;
            baseline: components["schemas"]["MetricSet"];
            evaluated: components["schemas"]["MetricSet"];
            delta: components["schemas"]["MetricDelta"];
        };
        /** @description Rendering hint only. Frontend may draw a chain/ramp before the stop using the single stop value. It MUST NOT interpret decorative samples as separate segment measurements or forecasts. */
        StopVisualizationPolicy: {
            /** @enum {string} */
            data_anchor: "stop_position";
            /** @enum {string} */
            value_metric: "load_index";
            /** @enum {string} */
            approach_effect: "decorative";
            /** Format: double */
            approach_length_m: number;
            /** @enum {string} */
            profile: "ramp_to_stop";
        };
        /**
         * @description Exact semantic variant used by frontend to choose the icon/illustration. Text must never be parsed to infer this state.
         * @enum {string}
         */
        IndicatorVariant: "none" | "clear" | "cloudy" | "rain" | "snow" | "heat" | "cold" | "other" | "deficit" | "balanced" | "surplus" | "up" | "down" | "flat" | "peak" | "off_peak" | "weekday" | "weekend" | "holiday" | "active" | "inactive" | "unknown";
    };
    responses: {
        /** @description Invalid JSON, query syntax or schema. */
        Error400: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Authentication required or session expired. */
        Error401: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Unknown identifier. */
        Error404: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Calculation hash mismatch. */
        Error409: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Known snapshot was retired and removed. */
        Error410: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Body/export exceeds limits. */
        Error413: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Valid structure but unsupported semantic combination. */
        Error422: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Admission/rate limit exceeded; Retry-After seconds. */
        Error429: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                /** @description Positive integer delay in seconds. */
                "Retry-After"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Not ready or bounded compute queue exhausted. */
        Error503: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                /** @description Positive integer delay in seconds. */
                "Retry-After"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Calculation timeout; no partial result. */
        Error504: {
            headers: {
                /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                "X-Request-ID"?: string;
                /** @description Per-operation caching policy. */
                "Cache-Control"?: string;
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
        /** @description Unsupported Content-Type; application/json required. */
        Error415: {
            headers: {
                [name: string]: unknown;
            };
            content: {
                "application/problem+json": components["schemas"]["Problem"];
            };
        };
    };
    parameters: never;
    requestBodies: never;
    headers: never;
    pathItems: never;
}
export type $defs = Record<string, never>;
export interface operations {
    getLiveness: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Health"];
                };
            };
        };
    };
    getReadiness: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Health"];
                };
            };
            503: components["responses"]["Error503"];
        };
    };
    login: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["LoginRequest"];
            };
        };
        responses: {
            /** @description Authenticated. Session cookie is set. */
            200: {
                headers: {
                    /** @description tramflow_session=<opaque>; Path=/; HttpOnly; SameSite=Strict; Secure in production */
                    "Set-Cookie"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuthSession"];
                };
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            413: components["responses"]["Error413"];
            415: components["responses"]["Error415"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
        };
    };
    getAuthSession: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Authenticated session. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["AuthSession"];
                };
            };
            401: components["responses"]["Error401"];
            503: components["responses"]["Error503"];
        };
    };
    logout: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Session revoked. */
            204: {
                headers: {
                    /** @description Expire tramflow_session cookie immediately. */
                    "Set-Cookie"?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            401: components["responses"]["Error401"];
            503: components["responses"]["Error503"];
        };
    };
    getBootstrap: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Bootstrap"];
                };
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    getSnapshot: {
        parameters: {
            query?: never;
            header?: {
                "If-None-Match"?: string;
            };
            path: {
                forecast_snapshot_id: components["schemas"]["Id"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["Snapshot"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    listRoutes: {
        parameters: {
            query: {
                network_version: components["schemas"]["Id"];
            };
            header?: {
                "If-None-Match"?: string;
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["RouteList"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    getRoute: {
        parameters: {
            query: {
                network_version: components["schemas"]["Id"];
            };
            header?: {
                "If-None-Match"?: string;
            };
            path: {
                route_id: components["schemas"]["Id"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["RouteDetail"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    getRouteGeometry: {
        parameters: {
            query: {
                network_version: components["schemas"]["Id"];
            };
            header?: {
                "If-None-Match"?: string;
            };
            path: {
                route_id: components["schemas"]["Id"];
            };
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["RouteGeometry"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    queryForecast: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ForecastQuery"];
            };
        };
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["CalculationResponse"];
                };
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            413: components["responses"]["Error413"];
            415: components["responses"]["Error415"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    evaluateScenario: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ScenarioQuery"];
            };
        };
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["CalculationResponse"];
                };
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            413: components["responses"]["Error413"];
            415: components["responses"]["Error415"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    getWeather: {
        parameters: {
            query: {
                forecast_snapshot_id: components["schemas"]["Id"];
                route_id: components["schemas"]["Id"];
                from: string;
                to: string;
            };
            header?: {
                "If-None-Match"?: string;
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["WeatherResponse"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    getModelQuality: {
        parameters: {
            query: {
                forecast_snapshot_id: components["schemas"]["Id"];
            };
            header?: {
                "If-None-Match"?: string;
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["QualityResponse"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    getDataSources: {
        parameters: {
            query: {
                forecast_snapshot_id: components["schemas"]["Id"];
            };
            header?: {
                "If-None-Match"?: string;
            };
            path?: never;
            cookie?: never;
        };
        requestBody?: never;
        responses: {
            /** @description Successful response. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["SourceList"];
                };
            };
            /** @description Not modified. Body omitted. Authentication still checked. */
            304: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description Version/content validator. */
                    ETag?: string;
                    [name: string]: unknown;
                };
                content?: never;
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            410: components["responses"]["Error410"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    querySummary: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["SummaryQuery"];
            };
        };
        responses: {
            /** @description Summary ready or explicitly unavailable. */
            200: {
                headers: {
                    [name: string]: unknown;
                };
                content: {
                    "application/json": components["schemas"]["SummaryResponse"];
                };
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            409: components["responses"]["Error409"];
            410: components["responses"]["Error410"];
            413: components["responses"]["Error413"];
            415: components["responses"]["Error415"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
    exportCalculation: {
        parameters: {
            query?: never;
            header?: never;
            path?: never;
            cookie?: never;
        };
        requestBody: {
            content: {
                "application/json": components["schemas"]["ExportRequest"];
            };
        };
        responses: {
            /** @description Complete CSV attachment. No polling or follow-up GET. */
            200: {
                headers: {
                    /** @description Request correlation ID. Echo a valid incoming ID, or create one. */
                    "X-Request-ID"?: string;
                    /** @description Per-operation caching policy. */
                    "Cache-Control"?: string;
                    /** @description attachment; filename="tramflow-<safe-id>.csv" */
                    "Content-Disposition"?: string;
                    "X-Calculation-ID"?: components["schemas"]["CalculationId"];
                    [name: string]: unknown;
                };
                content: {
                    "text/csv": string;
                };
            };
            400: components["responses"]["Error400"];
            401: components["responses"]["Error401"];
            404: components["responses"]["Error404"];
            409: components["responses"]["Error409"];
            410: components["responses"]["Error410"];
            413: components["responses"]["Error413"];
            415: components["responses"]["Error415"];
            422: components["responses"]["Error422"];
            429: components["responses"]["Error429"];
            503: components["responses"]["Error503"];
            504: components["responses"]["Error504"];
        };
    };
}
