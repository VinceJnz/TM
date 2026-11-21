[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$Image,
    [Parameter(Mandatory=$true)][string]$SourceRevision,
    [switch]$Apply
)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'release-policy.ps1')
Assert-ReleaseDigest $Image
Assert-ReleaseRevision $SourceRevision
if (!$Apply) { Write-Host 'Release reference and revision validated; no deployment performed.'; return }

# Operators must separately authorize deployment and apply approved migrations.
& docker pull $Image
if ($LASTEXITCODE -ne 0) { throw 'Release image pull failed.' }
$labelsJSON = & docker image inspect --format '{{json .Config.Labels}}' $Image
if ($LASTEXITCODE -ne 0) { throw 'Release image could not be inspected.' }
$revision = ($labelsJSON | ConvertFrom-Json).'org.opencontainers.image.revision'
if ($revision -cne $SourceRevision) { throw 'Release source revision does not match the approved image.' }
$previousImage=$env:TM_RELEASE_API_IMAGE
try {
    $env:TM_RELEASE_API_IMAGE=$Image
    $root=Split-Path -Parent $PSScriptRoot
    & docker compose --project-directory $root -f (Join-Path $root 'docker-compose.yml') -f (Join-Path $root 'docker-compose.release.yml') --profile prod config --quiet
    if ($LASTEXITCODE -ne 0) { throw 'Release configuration validation failed.' }
    & docker compose --project-directory $root -f (Join-Path $root 'docker-compose.yml') -f (Join-Path $root 'docker-compose.release.yml') --profile prod up -d --no-build apiserver
    if ($LASTEXITCODE -ne 0) { throw 'Release deployment failed.' }
} finally { $env:TM_RELEASE_API_IMAGE=$previousImage }
