[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-zA-Z0-9._:/-]+@sha256:[a-f0-9]{64}$')]
    [string]$ClientImage,
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-f0-9]{40}$')]
    [string]$SourceRevision,
    [Parameter(Mandatory = $true)]
    [string]$Image,
    [switch]$ValidateOnly
)
$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
if ($ValidateOnly) { Write-Host 'Release inputs are immutable and include a source revision.'; return }
& docker build -f (Join-Path $repositoryRoot 'Dockerfile.apibuild') --target final --build-arg "CLIENT_IMAGE=$ClientImage" --build-arg "SOURCE_REVISION=$SourceRevision" -t $Image $repositoryRoot
if ($LASTEXITCODE -ne 0) { throw 'Release image build failed' }
$identity = & docker image inspect --format '{{.Id}}' $Image
if ($LASTEXITCODE -ne 0) { throw 'Could not inspect release identity' }
Write-Host "Built release image: $identity"
Write-Host 'Publish through your approved registry process and deploy the resulting registry digest.'
