[CmdletBinding()]
param(
    [switch]$RemoveImage
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$composePath = Join-Path $projectRoot 'docker-compose.yml'

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

try {
    $docker = Get-DockerExecutable

    & $docker compose --project-directory $projectRoot -f $composePath down --remove-orphans
    if ($LASTEXITCODE -ne 0) {
        throw 'Docker Compose could not stop the Fast proxy.'
    }
    if ($RemoveImage) {
        & $docker image rm sub2api-fast-toggle:local
    }

    Write-Host 'Sub2API Fast Toggle has been stopped and removed.' -ForegroundColor Green
    Write-Host 'The .env and runtime mode file were preserved.'
}
catch {
    Write-Host "Uninstall failed: $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
