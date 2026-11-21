function Assert-StripeSandboxKey([string]$Key) {
 if ($Key -cnotmatch '^(sk|rk)_test_[A-Za-z0-9]+$') { throw 'A dedicated Stripe test-mode secret/restricted key is required.' }
}
function Assert-StripeSandboxBalance($Balance) {
 if ($Balance.object -cne 'balance' -or $Balance.livemode -isnot [bool] -or $Balance.livemode) { throw 'Stripe did not confirm a test-mode balance.' }
}
function Assert-GmailSandboxProfile($Profile,[string]$ExpectedEmail) {
 if ([string]::IsNullOrWhiteSpace($ExpectedEmail) -or ![string]::Equals([string]$Profile.emailAddress,$ExpectedEmail,[StringComparison]::OrdinalIgnoreCase)) { throw 'Gmail sandbox account identity did not match.' }
}
function Assert-GoogleDiscovery($Discovery) {
 if ($Discovery.issuer -cne 'https://accounts.google.com' -or $Discovery.authorization_endpoint -cne 'https://accounts.google.com/o/oauth2/v2/auth' -or $Discovery.token_endpoint -cne 'https://oauth2.googleapis.com/token') { throw 'Google discovery endpoints differ from the expected provider.' }
}
