param(
    [string]$BaseUrl = "http://localhost:8080",
    [switch]$SkipBuild,
    [switch]$UseBuildCache,
    [switch]$NoForceRecreate,
    [switch]$RecreateVolumes,
    [switch]$SkipSmoke,
    [int]$HealthTimeoutSeconds = 120
)

$ErrorActionPreference = "Stop"

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Action
    )

    Write-Host ""
    Write-Host "==> $Name" -ForegroundColor Cyan
    & $Action
}

function Test-Command {
    param([string]$Name)
    if ($null -eq (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Command '$Name' was not found."
    }
}

function Wait-Ready {
    param(
        [string]$Url,
        [int]$TimeoutSeconds
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastError = ""
    while ((Get-Date) -lt $deadline) {
        try {
            $response = Invoke-RestMethod -Uri "$Url/readyz" -TimeoutSec 5
            if ($response.status -eq "ready") {
                return
            }
            $lastError = "readyz returned status '$($response.status)'"
        } catch {
            $lastError = $_.Exception.Message
        }
        Start-Sleep -Seconds 3
    }

    throw "Service did not become ready within $TimeoutSeconds seconds. Last error: $lastError"
}

Invoke-Step "Checking Docker" {
    Test-Command "docker"
    docker version --format "Client={{.Client.Version}} Server={{.Server.Version}}" | Write-Host
}

if ($RecreateVolumes) {
    Invoke-Step "Stopping stack and removing volumes" {
        docker compose down -v
    }
}

if (-not $SkipBuild) {
    if ($UseBuildCache) {
        Invoke-Step "Building Docker images" {
            docker compose build
        }
    } else {
        Invoke-Step "Building Docker images without cache" {
            docker compose build --no-cache
        }
    }
}

Invoke-Step "Starting stack" {
    if ($NoForceRecreate) {
        docker compose up -d
    } else {
        docker compose up -d --force-recreate api worker-normalize worker-grouping worker-event-writer worker-alert
    }
}

Invoke-Step "Waiting for API readiness" {
    Wait-Ready -Url $BaseUrl -TimeoutSeconds $HealthTimeoutSeconds
}

Invoke-Step "Current container status" {
    docker compose ps
}

if (-not $SkipSmoke) {
    Invoke-Step "Running smoke test" {
        & "$PSScriptRoot\smoke.ps1" -BaseUrl $BaseUrl
    }
}

Write-Host ""
Write-Host "Deployment completed: $BaseUrl" -ForegroundColor Green
