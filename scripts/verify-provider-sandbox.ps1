[CmdletBinding()]
param([switch]$CheckStripeBalance,[switch]$CheckGoogleDiscovery,[switch]$CheckGmailProfile)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'provider-sandbox-policy.ps1')
function Read-Provider([string]$Uri,[hashtable]$Headers) {
 try { return Invoke-RestMethod -Method Get -Uri $Uri -Headers $Headers -TimeoutSec 20 -ErrorAction Stop }
 catch { throw 'Provider read-only check failed; verify sandbox credentials, permissions and connectivity. Response details omitted.' }
}
if($CheckStripeBalance) {
 Assert-StripeSandboxKey $env:STRIPE_SANDBOX_KEY
 $balance=Read-Provider 'https://api.stripe.com/v1/balance' @{Authorization=('Bearer '+$env:STRIPE_SANDBOX_KEY)}
 Assert-StripeSandboxBalance $balance
 Write-Host 'Stripe test-mode authentication and balance shape passed.'
}
if($CheckGoogleDiscovery) {
 $discovery=Read-Provider 'https://accounts.google.com/.well-known/openid-configuration' @{}
 Assert-GoogleDiscovery $discovery
 Write-Host 'Google issuer and authorization/token endpoints passed.'
}
if($CheckGmailProfile) {
 if([string]::IsNullOrWhiteSpace($env:GMAIL_SANDBOX_ACCESS_TOKEN) -or [string]::IsNullOrWhiteSpace($env:GMAIL_SANDBOX_EXPECTED_EMAIL)) { throw 'Dedicated Gmail test-account token and expected identity are required.' }
 $profile=Read-Provider 'https://gmail.googleapis.com/gmail/v1/users/me/profile' @{Authorization=('Bearer '+$env:GMAIL_SANDBOX_ACCESS_TOKEN)}
 Assert-GmailSandboxProfile $profile $env:GMAIL_SANDBOX_EXPECTED_EMAIL
 Write-Host 'Dedicated Gmail account authentication and identity passed.'
}
if(!$CheckStripeBalance -and !$CheckGoogleDiscovery -and !$CheckGmailProfile) { Write-Host 'No external checks requested. Select explicit read-only provider checks.' }
