$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'provider-sandbox-policy.ps1')
Assert-StripeSandboxKey 'sk_test_synthetic'
Assert-StripeSandboxKey 'rk_test_synthetic'
Assert-StripeSandboxBalance ([pscustomobject]@{object='balance';livemode=$false})
Assert-GmailSandboxProfile ([pscustomobject]@{emailAddress='synthetic@example.test'}) 'synthetic@example.test'
Assert-GoogleDiscovery ([pscustomobject]@{issuer='https://accounts.google.com';authorization_endpoint='https://accounts.google.com/o/oauth2/v2/auth';token_endpoint='https://oauth2.googleapis.com/token'})
foreach($check in @(
 {Assert-StripeSandboxKey 'sk_live_synthetic'},
 {Assert-StripeSandboxKey 'pk_test_synthetic'},
 {Assert-StripeSandboxBalance ([pscustomobject]@{object='balance';livemode=$true})},
 {Assert-StripeSandboxBalance ([pscustomobject]@{object='balance'})},
 {Assert-GmailSandboxProfile ([pscustomobject]@{emailAddress='other@example.test'}) 'synthetic@example.test'},
 {Assert-GoogleDiscovery ([pscustomobject]@{issuer='https://attacker.test'})}
)) {
 $rejected=$false;try { & $check } catch {$rejected=$true}
 if(!$rejected){throw 'Unsafe or incomplete provider response was accepted.'}
}
Write-Host 'Provider sandbox policy tests passed without external calls.'
