function Assert-ProductionEnvironment($Environment) {
    if ([string]$Environment.DEV_MODE -cne 'false') { throw 'Production requires DEV_MODE=false.' }
    if (![string]::IsNullOrWhiteSpace([string]$Environment.EMAIL_DEV_MODE_ADDR)) { throw 'Production requires EMAIL_DEV_MODE_ADDR to be unset or empty.' }
    if ([string]$Environment.STRIPE_TEST_MODE -cne 'false') { throw 'Production requires STRIPE_TEST_MODE=false; use a separate test deployment.' }
}

function Assert-ReleaseDigest {
    param([string]$Image)
    if ($Image -cnotmatch '^[a-zA-Z0-9._:/-]+@sha256:[a-f0-9]{64}$') {
        throw 'Release image must be a registry reference pinned to a lowercase sha256 digest.'
    }
}

function Assert-ReleaseRevision {
    param([string]$Revision)
    if ($Revision -cnotmatch '^[a-f0-9]{40}$') { throw 'Source revision must be a full lowercase Git SHA.' }
}

function Get-ReleaseHealth {
    param([string]$Container, [string]$Image, [string]$Revision)
    Assert-ReleaseDigest $Image
    Assert-ReleaseRevision $Revision
    if ($Container -notmatch '^[a-zA-Z0-9][a-zA-Z0-9_.-]*$') { throw 'A literal container name is required.' }
    $configuredImage = & docker inspect --format '{{.Config.Image}}' $Container
    if ($LASTEXITCODE -ne 0) { throw 'Release container could not be inspected.' }
    $labelsJSON = & docker inspect --format '{{json .Config.Labels}}' $Container
    if ($LASTEXITCODE -ne 0) { throw 'Release revision could not be inspected.' }
    $actualRevision = ($labelsJSON | ConvertFrom-Json).'org.opencontainers.image.revision'
    # Restrict inspection to state; never print container environment or secrets.
    $stateJSON = & docker inspect --format '{{json .State}}' $Container
    if ($LASTEXITCODE -ne 0) { throw 'Release health could not be inspected.' }
    $state = $stateJSON | ConvertFrom-Json
    $healthy = $configuredImage -ceq $Image -and $actualRevision -ceq $Revision -and $state.Status -eq 'running' -and $state.Health.Status -eq 'healthy'
    [pscustomobject]@{container=$Container; healthy=$healthy; state=$state.Status; health=$state.Health.Status; imageMatches=($configuredImage -ceq $Image); revisionMatches=($actualRevision -ceq $Revision)}
}
