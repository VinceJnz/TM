[CmdletBinding()]
param(
    [Parameter(Mandatory=$true)][string]$Container,
    [Parameter(Mandatory=$true)][string]$Image,
    [Parameter(Mandatory=$true)][string]$SourceRevision,
    [switch]$CheckOperations
)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'release-policy.ps1')
$health=Get-ReleaseHealth -Container $Container -Image $Image -Revision $SourceRevision
$health | ConvertTo-Json -Compress
if (!$health.healthy) { throw 'Release health or identity check failed; alert the operator and follow the recovery runbook.' }
if ($CheckOperations) {
    & docker exec $Container /app/tm-operations-check
    if ($LASTEXITCODE -ne 0) { throw 'Operational queue thresholds exceeded or snapshot unavailable; alert the service owner.' }
}
