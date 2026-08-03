[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$envPath = Join-Path $projectRoot '.env'
$exampleEnvPath = Join-Path $projectRoot '.env.example'
$runtimeDirectory = Join-Path $projectRoot 'runtime'
$modePath = Join-Path $runtimeDirectory 'mode'

function Get-DockerExecutable {
    $command = Get-Command docker.exe -ErrorAction SilentlyContinue
    if ($null -ne $command) {
        return $command.Source
    }
    $defaultPath = Join-Path $env:ProgramFiles 'Docker\Docker\resources\bin\docker.exe'
    if (Test-Path -LiteralPath $defaultPath) {
        return $defaultPath
    }
    throw 'Docker Desktop CLI was not found.'
}

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

try {
    $docker = Get-DockerExecutable
    if (-not (Test-Path -LiteralPath $envPath)) {
        Copy-Item -LiteralPath $exampleEnvPath -Destination $envPath
    }
    if (-not (Test-Path -LiteralPath $runtimeDirectory)) {
        New-Item -ItemType Directory -Path $runtimeDirectory | Out-Null
    }
    if (-not (Test-Path -LiteralPath $modePath)) {
        $encoding = New-Object System.Text.UTF8Encoding($false)
        [System.IO.File]::WriteAllText($modePath, "on`n", $encoding)
    }

    & $docker compose --project-directory $projectRoot -f (Join-Path $projectRoot 'docker-compose.yml') config --quiet
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Compose configuration validation failed.'
    }
    & $docker compose --project-directory $projectRoot -f (Join-Path $projectRoot 'docker-compose.yml') up -d --build --wait
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Compose could not start the Fast proxy.'
    }

    & (Join-Path $PSScriptRoot 'Toggle-Fast.ps1') -Status
    if ($LASTEXITCODE -ne 0) {
        throw 'The Fast proxy started, but its mode could not be verified.'
    }

    Write-Host ''
    Write-Host 'Installation complete.' -ForegroundColor Green
    $port = Get-ConfiguredValue -Name 'FAST_PROXY_PORT' -Default '18081'
    Write-Host "Use this API base URL in your client: http://localhost:$port/v1"
    Write-Host 'Keep Sub2API account management on its original port, normally http://localhost:8081'
}
catch {
    Write-Host "Installation failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
