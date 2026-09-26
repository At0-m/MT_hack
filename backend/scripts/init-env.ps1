$ErrorActionPreference = 'Stop'
$envPath = Join-Path (Split-Path $PSScriptRoot -Parent) '.env'
if (Test-Path -LiteralPath $envPath) { Write-Host '.env already exists; preserved.'; exit 0 }
function New-Secret {
    $bytes = New-Object byte[] 24
    [System.Security.Cryptography.RandomNumberGenerator]::Fill($bytes)
    return [Convert]::ToHexString($bytes).ToLowerInvariant()
}
$lines = @(
    "DB_PASSWORD=$(New-Secret)",
    "API_DB_PASSWORD=$(New-Secret)",
    'API_USER=dispatcher',
    "API_PASSWORD=$(New-Secret)",
    'CORS_ORIGIN=http://localhost:5173'
)
[IO.File]::WriteAllLines($envPath, $lines, [Text.UTF8Encoding]::new($false))
Write-Host 'Created .env with random local credentials. Do not commit it.'
