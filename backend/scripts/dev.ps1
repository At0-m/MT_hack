$ErrorActionPreference = 'Stop'
Push-Location (Split-Path $PSScriptRoot -Parent)
try {
    & "$PSScriptRoot/init-env.ps1"
    if (-not (Test-Path -LiteralPath 'testdata/demo-bundle.json')) {
        go run ./cmd/fixtures --onnx
        if ($LASTEXITCODE -ne 0) { throw 'Fixture generation failed' }
    }
    docker compose up -d db
    if ($LASTEXITCODE -ne 0) { throw 'Database startup failed' }
    $envFile = Get-Content -LiteralPath '.env' | ConvertFrom-StringData
    docker compose run --rm publisher --migrate --create-user $envFile.API_USER
    if ($LASTEXITCODE -ne 0) { throw 'Migration failed' }
    $active = docker compose exec -T db psql -U tramflow_publisher -d tramflow -Atc 'SELECT snapshot_id FROM active_forecast_snapshot WHERE slot=1'
    if ($LASTEXITCODE -ne 0) { throw 'Active snapshot check failed' }
    if (-not $active) {
        docker compose run --rm publisher
        if ($LASTEXITCODE -ne 0) { throw 'Publication failed' }
    }
    docker compose up -d --build api
    if ($LASTEXITCODE -ne 0) { throw 'API startup failed' }
    Write-Host 'API: http://localhost:8080. Credentials are in .env.'
} finally { Pop-Location }
