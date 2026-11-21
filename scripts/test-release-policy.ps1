$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'release-policy.ps1')
$digest='registry.example.test/tm@sha256:'+('a'*64)
$revision='b'*40
$global:tmPolicyFixtureDigest=$digest
$global:tmPolicyFixtureRevision=$revision
Assert-ReleaseDigest $digest
Assert-ReleaseRevision $revision
foreach ($invalid in @('tm:latest','tm@sha256:abc',('tm@sha256:'+('A'*64)),"$digest extra",'')) {
    $rejected=$false
    try { Assert-ReleaseDigest $invalid } catch { $rejected=$true }
    if (!$rejected) { throw 'Mutable or invalid image was accepted.' }
}
$global:tmPolicyMockHealth='healthy'
$global:tmPolicyMockOperationsFailure=$false
function docker {
    param([Parameter(ValueFromRemainingArguments=$true)]$Arguments)
    $global:LASTEXITCODE=0
    if ($Arguments[0] -eq 'exec') {
        if ($global:tmPolicyMockOperationsFailure) { $global:LASTEXITCODE=1 }
        return '{"pending_refunds":0,"alerts":[]}'
    }
    switch ($Arguments[2]) {
        '{{.Config.Image}}' { return $global:tmPolicyFixtureDigest }
        '{{json .Config.Labels}}' { return ('{"org.opencontainers.image.revision":"'+$global:tmPolicyFixtureRevision+'"}') }
        '{{json .State}}' { return ('{"Status":"running","Health":{"Status":"'+$global:tmPolicyMockHealth+'"}}') }
        default { throw 'Unexpected Docker invocation.' }
    }
}
if (!(Get-ReleaseHealth 'synthetic-release' $digest $revision).healthy) { throw 'Healthy matching release rejected.' }
$global:tmPolicyMockHealth='unhealthy'
if ((Get-ReleaseHealth 'synthetic-release' $digest $revision).healthy) { throw 'Unhealthy container accepted.' }
$global:tmPolicyMockHealth='healthy'
if ((Get-ReleaseHealth 'synthetic-release' ('other.example/tm@sha256:'+('c'*64)) $revision).healthy) { throw 'Wrong image accepted.' }
if ((Get-ReleaseHealth 'synthetic-release' $digest ('c'*40)).healthy) { throw 'Wrong revision accepted.' }
Write-Host 'Release digest, identity and health policy tests passed.'
& (Join-Path $PSScriptRoot 'check-release-health.ps1') -Container synthetic-release -Image $digest -SourceRevision $revision -CheckOperations | Out-Null
$global:tmPolicyMockOperationsFailure=$true
$rejected=$false
try { & (Join-Path $PSScriptRoot 'check-release-health.ps1') -Container synthetic-release -Image $digest -SourceRevision $revision -CheckOperations | Out-Null } catch { $rejected=$true }
if (!$rejected) { throw 'Operational failure did not propagate to the monitoring runner.' }
Write-Host 'Operational queue alert propagation passed.'
Remove-Variable tmPolicyFixtureDigest,tmPolicyFixtureRevision,tmPolicyMockHealth,tmPolicyMockOperationsFailure -Scope Global
