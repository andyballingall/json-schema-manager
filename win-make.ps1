param (
    [Parameter(Mandatory = $true, Position = 0)]
    [ValidateSet("build", "run", "clean", "test", "test-race", "test-cover", "check-coverage", "cover-html", "lint", "fmt", "snapshot", "release-check", "setup")]
    [string]$Target
)

$hasRaceSupport = $true
if ($env:CGO_ENABLED -eq "0" -or ($null -eq (Get-Command gcc -ErrorAction SilentlyContinue))) {
    $hasRaceSupport = $false
}

function Invoke-GoTest {
    param([string[]]$extraArgs)
    $allArgs = @("./...", "-v")
    if ($hasRaceSupport) {
        $allArgs = @("-race") + $allArgs
    }
    else {
        Write-Host "⚠️  Warning: Race detection requires a C compiler (gcc) and CGO_ENABLED=1. Running without -race." -ForegroundColor Yellow
    }
    if ($extraArgs) { $allArgs = $extraArgs + $allArgs }
    go run scripts/tester/main.go @allArgs
}

switch ($Target) {
    "build" {
        go run scripts/build/main.go
    }
    "run" {
        go run cmd/jsm/main.go
    }
    "clean" {
        go run scripts/clean/main.go
        go clean
    }
    "test" {
        go run scripts/tester/main.go ./... -v
    }
    "test-race" {
        Invoke-GoTest
    }
    "test-cover" {
        $coverArgs = @("-count=1", "-coverpkg=./internal/...", "-coverprofile=coverage.out")
        Invoke-GoTest $coverArgs
        if ($LASTEXITCODE -eq 0) {
            go tool cover -func coverage.out
        }
    }
    "check-coverage" {
        & "$PSCommandPath" test-cover
        if ($LASTEXITCODE -eq 0) {
            go run scripts/check_coverage/main.go coverage.out
        }
    }
    "cover-html" {
        $htmlArgs = @("-count=1", "-coverpkg=./internal/...", "-coverprofile=coverage.out")
        Invoke-GoTest $htmlArgs
        if ($LASTEXITCODE -eq 0) {
            go tool cover -html coverage.out
        }
    }
    "lint" {
        go run scripts/lint/main.go
    }
    "fmt" {
        go run scripts/fmt/main.go
    }
    "snapshot" {
        go run scripts/snapshot/main.go
    }
    "release-check" {
        go run scripts/release_check/main.go
    }
    "setup" {
        go run scripts/setup/main.go
    }
}
if ($Target -ne "setup" -and $LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}
