param(
    [switch]$Integration
)

$ErrorActionPreference = "Stop"

$previousGoWork = $env:GOWORK

Push-Location $PSScriptRoot

try {
    $env:GOWORK = "off"

    if ($Integration -and
        [string]::IsNullOrWhiteSpace($env:LEDGER_TEST_DATABASE_URL)) {
        throw "Set LEDGER_TEST_DATABASE_URL to the dedicated test database before running with -Integration."
    }

    Write-Host "Running Ledger unit tests..."
    go test ./... -count=1

    if ($LASTEXITCODE -ne 0) {
        throw "Ledger unit tests failed."
    }

    Write-Host "Running go vet..."
    go vet ./...

    if ($LASTEXITCODE -ne 0) {
        throw "go vet failed."
    }

    Write-Host "Building Ledger packages..."
    go build ./...

    if ($LASTEXITCODE -ne 0) {
        throw "Ledger build failed."
    }

    if ($Integration) {
        Write-Host "Running go vet for integration tests..."
        go vet -tags=integration ./integration

        if ($LASTEXITCODE -ne 0) {
            throw "Integration go vet failed."
        }

        Write-Host "Running Ledger integration tests..."
        go test -tags=integration ./integration -count=1 -parallel=1

        if ($LASTEXITCODE -ne 0) {
            throw "Ledger integration tests failed."
        }
    }

    Write-Host "All requested Ledger checks passed."
}
finally {
    $env:GOWORK = $previousGoWork
    Pop-Location
}