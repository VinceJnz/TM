$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'release-policy.ps1')
$name='tm-health-policy-'+[guid]::NewGuid().ToString('N').Substring(0,12)
$image='alpine:latest@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf'
$revision='b'*40
try {
    & docker run -d --name $name --label "org.opencontainers.image.revision=$revision" --health-cmd 'test ! -e /tmp/unhealthy' --health-interval 1s --health-timeout 1s --health-retries 1 $image sh -c 'while :; do sleep 1; done' | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Synthetic health container failed to start.' }
    $ready=$false
    for ($attempt=0;$attempt -lt 30;$attempt++) {
        if ((Get-ReleaseHealth $name $image $revision).healthy) { $ready=$true;break }
        Start-Sleep -Milliseconds 250
    }
    if (!$ready) { throw 'Healthy synthetic release was not accepted.' }
    & docker exec $name touch /tmp/unhealthy
    if ($LASTEXITCODE -ne 0) { throw 'Could not inject synthetic health failure.' }
    $failed=$false
    for ($attempt=0;$attempt -lt 30;$attempt++) {
        if (!(Get-ReleaseHealth $name $image $revision).healthy) { $failed=$true;break }
        Start-Sleep -Milliseconds 250
    }
    if (!$failed) { throw 'Health failure was not detected.' }
    & docker stop --timeout 2 $name | Out-Null
    if ($LASTEXITCODE -ne 0 -or (Get-ReleaseHealth $name $image $revision).healthy) { throw 'Stopped release was accepted.' }
    Write-Host 'Real Docker health and release identity checks passed.'
} finally {
    & docker rm -f $name 2>$null | Out-Null
}

# Validate the actual release override without loading workstation credentials.
$emptyFile=[IO.Path]::GetTempFileName()
$previous=@{}
$testValues=@{TM_API_ENV_FILE=$emptyFile;TM_API_DB_PASSWORD='synthetic-api';TM_DB_PASSWORD='synthetic-db';TM_EMAIL_REDIRECT_ALL='false';TM_EMAIL_REDIRECT_TO='';TM_STRIPE_TEST_MODE='false';TM_RELEASE_API_IMAGE=('registry.example.test/tm@sha256:'+('a'*64))}
try {
    foreach ($key in $testValues.Keys) { $previous[$key]=[Environment]::GetEnvironmentVariable($key,'Process');[Environment]::SetEnvironmentVariable($key,$testValues[$key],'Process') }
    $root=Split-Path -Parent $PSScriptRoot
    $configuration = & docker compose --env-file $emptyFile --project-directory $root -f (Join-Path $root 'docker-compose.yml') -f (Join-Path $root 'docker-compose.release.yml') --profile prod config --format json
    if ($LASTEXITCODE -ne 0) { throw 'Synthetic release configuration failed.' }
    $api=($configuration | ConvertFrom-Json).services.apiserver
    Assert-ProductionEnvironment $api.environment
    if ($api.image -cne $testValues.TM_RELEASE_API_IMAGE -or $null -ne $api.build) { throw 'Release override did not preserve the approved image or disable builds.' }
    Write-Host 'Release Compose override validated with synthetic configuration.'
} finally {
    foreach ($key in $testValues.Keys) { [Environment]::SetEnvironmentVariable($key,$previous[$key],'Process') }
    Remove-Item -LiteralPath $emptyFile
}
