$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'build-policy.ps1')
$revision='a'*40
if((Get-BuildRevision $revision $false) -cne $revision){throw 'Clean revision changed.'}
if((Get-BuildRevision $revision $true) -cne "$revision-dirty"){throw 'Dirty build mislabeled.'}
$rejected=$false
try{Get-BuildRevision $revision $true $true | Out-Null}catch{$rejected=$true}
if(!$rejected){throw 'Dirty release build accepted.'}
Write-Host 'Build revision policy passed.'

# Both entry points must use the same compiler stages and WASM/runtime pairing.
$root=Split-Path -Parent $PSScriptRoot
$client=[IO.File]::ReadAllText((Join-Path $root 'Dockerfile.client1'))
$api=[IO.File]::ReadAllText((Join-Path $root 'Dockerfile.apibuild'))
$clientStages=$client.Substring($client.IndexOf('ARG CLIENT_BUILDER'),$client.IndexOf('FROM scratch AS artifact')-$client.IndexOf('ARG CLIENT_BUILDER'))
$apiStages=$api.Substring($api.IndexOf('ARG CLIENT_BUILDER'),$api.IndexOf('FROM scratch AS client1-built')-$api.IndexOf('ARG CLIENT_BUILDER'))
function Normalize-Stages([string]$value){return (($value -split '\r?\n' | Where-Object {$_ -notmatch '^#' -and $_.Trim() -ne ''}) -join "`n").Trim()}
if((Normalize-Stages $clientStages) -cne (Normalize-Stages $apiStages)){throw 'Frontend compiler stages drifted between build entry points.'}
if(!$api.Contains('ARG CLIENT_IMAGE=client1-built')){throw 'Default API build depends on an external stale client image.'}
Write-Host 'Self-contained build and compiler pairing policy passed.'
if ($api -notmatch '(?m)^LABEL org\.tripmanager\.client\.image=\$CLIENT_IMAGE\s*$') {
    throw 'API image does not record its selected frontend artifact.'
}
