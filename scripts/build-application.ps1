[CmdletBinding()]
param(
 [ValidateSet('go-builder','tinygo-builder')][string]$ClientBuilder='go-builder',
 [ValidateSet('final','interactive','debug')][string]$ApiTarget='final',
 [string]$ClientImage='',
 [switch]$RequireClean
)
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
if([string]::IsNullOrWhiteSpace($ClientImage)){
 $ClientImage=if($ClientBuilder -eq 'go-builder'){'tm-client1-wasm:go-dev'}else{'tm-client1-wasm:tinygo-dev'}
}
$revision=& git -C $root -c "safe.directory=$root" rev-parse HEAD
if($LASTEXITCODE -ne 0){throw 'Could not identify the source revision.'}
. (Join-Path $PSScriptRoot 'build-policy.ps1')
$changes=& git -C $root -c "safe.directory=$root" status --porcelain --untracked-files=normal
if($LASTEXITCODE -ne 0){throw 'Could not check the working tree.'}
$revision=Get-BuildRevision $revision ([bool]$changes) $RequireClean.IsPresent
# Build only. Do not start containers, migrate data, or load credentials into layers.
& docker build -f (Join-Path $root 'Dockerfile.client1') --build-arg "CLIENT_BUILDER=$ClientBuilder" --build-arg "SOURCE_REVISION=$revision" -t $ClientImage $root
if($LASTEXITCODE -ne 0){throw 'Frontend artifact build failed; API build was not started.'}
$profile='prod';$service='apiserver'
if($ApiTarget -ne 'final'){$profile=$ApiTarget;$service="apiserver-$ApiTarget"}
& docker compose --env-file (Join-Path $root 'api-server/.env') --project-directory $root -f (Join-Path $root 'docker-compose.yml') --profile $profile build --build-arg "CLIENT_IMAGE=$ClientImage" --build-arg "CLIENT_BUILDER=$ClientBuilder" --build-arg "SOURCE_REVISION=$revision" $service
if($LASTEXITCODE -ne 0){throw 'API image assembly failed.'}
Write-Host "Application build passed: $ClientBuilder frontend, $ApiTarget API target, revision $revision. No containers were started."
