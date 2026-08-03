[CmdletBinding(DefaultParameterSetName = 'Toggle')]
param(
    [Parameter(Mandatory, ParameterSetName = 'Set')]
    [ValidateSet('On', 'Off')]
    [string]$Set,

    [Parameter(Mandatory, ParameterSetName = 'Status')]
    [switch]$Status,

    [Parameter(ParameterSetName = 'Toggle')]
    [switch]$Toggle
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$runtimeDirectory = Join-Path $projectRoot 'runtime'
$modePath = Join-Path $runtimeDirectory 'mode'
$envPath = Join-Path $projectRoot '.env'
$mutex = New-Object System.Threading.Mutex($false, 'Local\Sub2ApiFastToggle')
$lockTaken = $false

function Get-ConfiguredValue {
    param(
        [Parameter(Mandatory)][string]$Name,
        [Parameter(Mandatory)][string]$Default
    )

    if (-not (Test-Path -LiteralPath $envPath)) {
        return $Default
    }
    $pattern = '^\s*' + [regex]::Escape($Name) + '\s*=\s*(.*)\s*$'
    $match = Get-Content -LiteralPath $envPath | Select-String -Pattern $pattern | Select-Object -Last 1
    if ($null -eq $match) {
        return $Default
    }
    return $match.Matches[0].Groups[1].Value.Trim().Trim('"').Trim("'")
}

function Get-FileMode {
    if (-not (Test-Path -LiteralPath $modePath)) {
        return 'off'
    }
    $value = (Get-Content -LiteralPath $modePath -Raw).Trim().ToLowerInvariant()
    if ($value -eq 'on') {
        return 'on'
    }
    return 'off'
}

function Set-FileMode {
    param([Parameter(Mandatory)][ValidateSet('on', 'off')][string]$Mode)

    if (-not (Test-Path -LiteralPath $runtimeDirectory)) {
        New-Item -ItemType Directory -Path $runtimeDirectory | Out-Null
    }

    $temporaryPath = Join-Path $runtimeDirectory ("mode.{0}.tmp" -f $PID)
    $backupPath = Join-Path $runtimeDirectory ("mode.{0}.bak" -f $PID)
    $encoding = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($temporaryPath, $Mode + [Environment]::NewLine, $encoding)

    try {
        if (Test-Path -LiteralPath $modePath) {
            try {
                [System.IO.File]::Replace($temporaryPath, $modePath, $backupPath, $true)
            }
            catch [System.IO.IOException] {
                Move-Item -LiteralPath $temporaryPath -Destination $modePath -Force
            }
            catch [System.PlatformNotSupportedException] {
                Move-Item -LiteralPath $temporaryPath -Destination $modePath -Force
            }
        }
        else {
            Move-Item -LiteralPath $temporaryPath -Destination $modePath
        }
    }
    finally {
        if (Test-Path -LiteralPath $temporaryPath) {
            Remove-Item -LiteralPath $temporaryPath -Force
        }
        if (Test-Path -LiteralPath $backupPath) {
            Remove-Item -LiteralPath $backupPath -Force
        }
    }
}

function Get-ProxyMode {
    $port = Get-ConfiguredValue -Name 'FAST_PROXY_PORT' -Default '18081'
    $uri = "http://127.0.0.1:$port/_sub2api-fast/status"
    try {
        return Invoke-RestMethod -UseBasicParsing -Uri $uri -TimeoutSec 3
    }
    catch {
        return $null
    }
}

function Write-Status {
    param([Parameter(Mandatory)][string]$DesiredMode)

    $runtime = Get-ProxyMode
    if ($DesiredMode -eq 'on') {
        Write-Host 'Sub2API Fast mode: ON' -ForegroundColor Green
    }
    else {
        Write-Host 'Sub2API Fast mode: OFF' -ForegroundColor Yellow
    }

    if ($null -eq $runtime) {
        Write-Host 'Proxy status: unavailable; the saved mode will apply when the proxy is running.' -ForegroundColor DarkYellow
        return
    }
    if (-not $runtime.valid -or $runtime.mode -ne $DesiredMode) {
        throw "Proxy mode mismatch: saved=$DesiredMode runtime=$($runtime.mode) valid=$($runtime.valid)"
    }
    Write-Host "Proxy status: verified ($($runtime.mode))"
    Write-Host 'The next request uses the selected mode; active streams are unchanged.'
}

try {
    $lockTaken = $mutex.WaitOne([TimeSpan]::FromSeconds(10))
    if (-not $lockTaken) {
        throw 'Another Fast toggle operation is still running.'
    }

    $currentMode = Get-FileMode
    if ($Status) {
        Write-Status -DesiredMode $currentMode
        exit 0
    }

    if ($PSCmdlet.ParameterSetName -eq 'Set') {
        $targetMode = $Set.ToLowerInvariant()
    }
    elseif ($currentMode -eq 'on') {
        $targetMode = 'off'
    }
    else {
        $targetMode = 'on'
    }

    Set-FileMode -Mode $targetMode
    Write-Status -DesiredMode $targetMode
}
catch {
    Write-Host "Fast toggle failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
finally {
    if ($lockTaken) {
        $mutex.ReleaseMutex()
    }
    $mutex.Dispose()
}
