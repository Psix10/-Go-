$ErrorActionPreference = "Stop"

Push-Location $PSScriptRoot

try {
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

    Write-Host "Building Gateway..."
    go build .

    if ($LASTEXITCODE -ne 0) {
        throw "Gateway build failed."
    }

    Write-Host "All checks passed."
}
finally {
    Pop-Location
}