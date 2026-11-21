$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'release-policy.ps1')
$digest='registry.example.test/tm@sha256:'+('a'*64)
$revision='b'*40
$script:fixtureRevision=$revision
Assert-ReleaseDigest $digest
Assert-ReleaseRevision $revision
foreach ($invalid in @('tm:latest','tm@sha256:abc',('tm@sha256:'+('A'*64)),"$digest extra",'')) {
    $rejected=$false
    try { Assert-ReleaseDigest $invalid } catch { $rejected=$true }
    if (!$rejected) { throw 'Mutable or invalid image was accepted.' }
}
$script:mockHealth='healthy'
function docker {
    param([Parameter(ValueFromRemainingArguments=$true)]$Arguments)
    $global:LASTEXITCODE=0
    switch ($Arguments[2]) {
        '{{.Config.Image}}' { return $digest }
        '{{json .Config.Labels}}' { return ('{"org.opencontainers.image.revision":"'+$script:fixtureRevision+'"}') }
        '{{json .State}}' { return ('{"Status":"running","Health":{"Status":"'+$script:mockHealth+'"}}') }
        default { throw 'Unexpected Docker invocation.' }
    }
}
if (!(Get-ReleaseHealth 'synthetic-release' $digest $revision).healthy) { throw 'Healthy matching release rejected.' }
$script:mockHealth='unhealthy'
if ((Get-ReleaseHealth 'synthetic-release' $digest $revision).healthy) { throw 'Unhealthy container accepted.' }
$script:mockHealth='healthy'
if ((Get-ReleaseHealth 'synthetic-release' ('other.example/tm@sha256:'+('c'*64)) $revision).healthy) { throw 'Wrong image accepted.' }
if ((Get-ReleaseHealth 'synthetic-release' $digest ('c'*40)).healthy) { throw 'Wrong revision accepted.' }
Write-Host 'Release digest, identity and health policy tests passed.'
