param(
    [string]$BaseUrl = "http://localhost:8080",
    [string]$ProjectId = "1",
    [string]$PublicKey = "0123456789abcdef0123456789abcdef",
    [string]$WebhookUrl = ""
)

$ErrorActionPreference = "Stop"

function Invoke-Json {
    param(
        [string]$Method,
        [string]$Url,
        [object]$Body = $null,
        [hashtable]$Headers = @{}
    )

    $params = @{
        Method = $Method
        Uri = $Url
        Headers = $Headers
    }
    if ($null -ne $Body) {
        $params.ContentType = "application/json"
        $params.Body = ($Body | ConvertTo-Json -Depth 20)
    }
    Invoke-RestMethod @params
}

function Wait-Issues {
    param(
        [string]$Url,
        [int]$TimeoutSeconds = 30
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $issues = Invoke-Json -Method GET -Url $Url
        if (@($issues.items).Count -gt 0) {
            return $issues
        }
        Start-Sleep -Seconds 2
    }

    throw "Smoke test did not observe any unresolved issue within $TimeoutSeconds seconds."
}

Write-Host "Checking API health..."
Invoke-Json -Method GET -Url "$BaseUrl/healthz" | Out-Null
Invoke-Json -Method GET -Url "$BaseUrl/readyz" | Out-Null

if ($WebhookUrl -ne "") {
    Write-Host "Creating frequency webhook alert..."
    Invoke-Json -Method POST -Url "$BaseUrl/api/projects/$ProjectId/alerts/webhook" -Body @{
        name = "Smoke frequency alert"
        event_type = "frequency"
        webhook_url = $WebhookUrl
        min_level = "error"
        threshold_count = 2
        window_seconds = 300
        cooldown_seconds = 300
    } | Out-Null
}

Write-Host "Sending two matching events..."
for ($i = 1; $i -le 2; $i++) {
    $eventId = [guid]::NewGuid().ToString("N")
    Invoke-Json -Method POST -Url "$BaseUrl/api/$ProjectId/envelope/" -Headers @{
        "X-Sentry-Key" = $PublicKey
        "X-SDK-Name" = "smoke-sdk"
        "X-SDK-Version" = "0.1.0"
    } -Body @{
        event_id = $eventId
        timestamp = (Get-Date).ToUniversalTime().ToString("o")
        platform = "javascript"
        runtime = @{
            name = "browser"
            version = "smoke"
        }
        sdk = @{
            name = "smoke-sdk"
            version = "0.1.0"
        }
        level = "error"
        message = "Smoke test failure user 123456"
        exception = @{
            type = "SmokeError"
            value = "Smoke test failure user 123456"
            stacktrace = @()
        }
        release = "smoke"
        environment = "local"
        tags = @{
            smoke = "true"
        }
        user = @{
            id = "smoke-user"
            access_token = "should-be-filtered"
        }
    } | Out-Null
}

Write-Host "Waiting for workers..."
$issues = Wait-Issues -Url "$BaseUrl/api/projects/$ProjectId/issues?status=unresolved&limit=10"

Write-Host "Querying issues, events, stats, alerts..."
$events = Invoke-Json -Method GET -Url "$BaseUrl/api/projects/$ProjectId/events?limit=10"
$trend = Invoke-Json -Method GET -Url "$BaseUrl/api/projects/$ProjectId/stats/trend"
$alerts = Invoke-Json -Method GET -Url "$BaseUrl/api/projects/$ProjectId/alerts"
$deliveries = Invoke-Json -Method GET -Url "$BaseUrl/api/projects/$ProjectId/alert-deliveries?limit=10"

[pscustomobject]@{
    issue_count = @($issues.items).Count
    event_count = @($events.items).Count
    trend_points = @($trend.items).Count
    alert_count = @($alerts.items).Count
    delivery_count = @($deliveries.items).Count
}
