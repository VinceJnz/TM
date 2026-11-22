$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'build-policy.ps1')
$revision='a'*40
if((Get-BuildRevision $revision $false) -cne $revision){throw 'Clean revision changed.'}
if((Get-BuildRevision $revision $true) -cne "$revision-dirty"){throw 'Dirty build mislabeled.'}
$rejected=$false
try{Get-BuildRevision $revision $true $true | Out-Null}catch{$rejected=$true}
if(!$rejected){throw 'Dirty release build accepted.'}
Write-Host 'Build revision policy passed.'
