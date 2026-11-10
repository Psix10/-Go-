param(
    [switch]$Integration
)

$ErrorActionPreference = "Stop"

$previousGoWork = $env:GOWORK

Push-Location $PSScriptRoot

try {
    $env:GOWORK = "off"

    if ($Integration -and
        [string]::IsNullOrWhiteSpace($env:API_TEST_DATABASE_URL)) {
        throw "Set API_TEST_DATABASE_URL to the dedicated test database before running with -Integration."
    }

    Write-Host "Running Gateway unit tests..."
    go test ./... -count=1

    if ($LASTEXITCODE -ne 0) {
        throw "Gateway unit tests failed."
    }

    Write-Host "Running go vet..."
    go vet ./...

    if ($LASTEXITCODE -ne 0) {
        throw "go vet failed."
    }

    Write-Host "Building Gateway..."
    go build .

    if ($LASTEXITCODE -ne 0) {
        throw "Gateway build failed."
    }

    if ($Integration) {
        Write-Host "Running go vet for integration tests..."
        go vet -tags=integration ./internal/api

        if ($LASTEXITCODE -ne 0) {
            throw "Integration go vet failed."
        }

        Write-Host "Running Gateway integration tests..."
        go test -tags=integration ./internal/api -count=1 -parallel=1

        if ($LASTEXITCODE -ne 0) {
            throw "Gateway integration tests failed."
        }
    }

    Write-Host "All requested Gateway checks passed."
}
finally {
    $env:GOWORK = $previousGoWork
    Pop-Location
}