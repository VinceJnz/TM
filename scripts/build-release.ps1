[CmdletBinding()]
param(
    [ValidatePattern('^[a-zA-Z0-9._:/-]+@sha256:[a-f0-9]{64}$')]
    [string]$ClientImage,
    [ValidateSet('go-builder','tinygo-builder')]
    [string]$ClientBuilder = 'go-builder',
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-f0-9]{40}$')]
    [string]$SourceRevision,
    [Parameter(Mandatory = $true)]
    [string]$Image,
    [switch]$ValidateOnly
)
$ErrorActionPreference = 'Stop'
$repositoryRoot = Split-Path -Parent $PSScriptRoot
if ($ValidateOnly) { Write-Host 'Release inputs validated.'; return }
$headRevision = & git -C $repositoryRoot rev-parse HEAD
if ($LASTEXITCODE -ne 0 -or $headRevision -cne $SourceRevision) {
    throw 'Release source revision does not match the checked-out commit.'
}
$changes = & git -C $repositoryRoot status --porcelain --untracked-files=normal
if ($LASTEXITCODE -ne 0 -or $changes) { throw 'Release builds require a clean working tree.' }
$buildArgs = @('--build-arg', "CLIENT_BUILDER=$ClientBuilder", '--build-arg', "SOURCE_REVISION=$SourceRevision")
if ($ClientImage) {
    & docker pull $ClientImage
    if ($LASTEXITCODE -ne 0) { throw 'Prebuilt client artifact pull failed.' }
    $clientLabelsJson = & docker image inspect --format '{{json .Config.Labels}}' $ClientImage
    if ($LASTEXITCODE -ne 0) { throw 'Could not inspect the prebuilt client artifact.' }
    $clientLabels = $clientLabelsJson | ConvertFrom-Json
    if ($clientLabels.'org.opencontainers.image.revision' -cne $SourceRevision -or
        $clientLabels.'org.tripmanager.client.builder' -cne $ClientBuilder) {
        throw 'Prebuilt client source revision or compiler does not match the requested release.'
    }
    $buildArgs += @('--build-arg', "CLIENT_IMAGE=$ClientImage")
}
& docker build -f (Join-Path $repositoryRoot 'Dockerfile.apibuild') --target final @buildArgs -t $Image $repositoryRoot
if ($LASTEXITCODE -ne 0) { throw 'Release image build failed' }
$identity = & docker image inspect --format '{{.Id}}' $Image
if ($LASTEXITCODE -ne 0) { throw 'Could not inspect release identity' }
Write-Host "Built release image: $identity"
Write-Host 'Publish through your approved registry process and deploy the resulting registry digest.'
