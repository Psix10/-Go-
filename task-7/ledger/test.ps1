$ErrorActionPreference = "Stop"

Push-Location $PSScriptRoot

try {
    $env:LEDGER_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_ledger_test?sslmode=disable"
    $env:API_TEST_DATABASE_URL = "postgres://postgres:postgres@127.0.0.1:5444/cashapp_api_test?sslmode=disable"

    $env:TEST_REDIS_ADDR = "127.0.0.1:6379"
    $env:TEST_REDIS_PASSWORD = ""

    Write-Host "Running tests..."
    go test ./... -count=1

    if ($LASTEXITCODE -ne 0) {
        throw "Tests failed."
    }

    Write-Host "Running go vet..."
    go vet ./...

    if ($LASTEXITCODE -ne 0) {
        throw "go vet failed."
    }

    Write-Host "All checks passed."
}
finally {
    Pop-Location
}