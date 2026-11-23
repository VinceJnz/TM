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

# Validate the release override with a synthetic api-server/.env.
$root=Split-Path -Parent $PSScriptRoot
$fixture=Join-Path ([IO.Path]::GetTempPath()) ("tm-release-config-"+[guid]::NewGuid().ToString('N'))
$files=@('docker-compose.yml','docker-compose.release.yml')
$apiDirectory=Join-Path $fixture 'api-server'
$apiEnv=Join-Path $apiDirectory '.env'
$releaseImage='registry.example.test/tm@sha256:'+('a'*64)
try {
    New-Item -ItemType Directory -Path $apiDirectory -Force | Out-Null
    foreach ($file in $files) { Copy-Item -LiteralPath (Join-Path $root $file) -Destination $fixture }
    [IO.File]::WriteAllLines($apiEnv,[string[]]@('TM_API_DB_PASSWORD=synthetic-api','TM_DB_PASSWORD=synthetic-db',"TM_RELEASE_API_IMAGE=$releaseImage",'DEV_MODE=false','EMAIL_REDIRECT_ALL=false','STRIPE_TEST_MODE=false'))
    $configuration = & docker compose --env-file $apiEnv --project-directory $fixture -f (Join-Path $fixture $files[0]) -f (Join-Path $fixture $files[1]) --profile prod config --format json
    if ($LASTEXITCODE -ne 0) { throw 'Synthetic release configuration failed.' }
    $api=($configuration | ConvertFrom-Json).services.apiserver
    Assert-ProductionEnvironment $api.environment
    if ($api.image -cne $releaseImage -or $null -ne $api.build) { throw 'Release override did not preserve the approved image or disable builds.' }
    Write-Host 'Release Compose override validated with synthetic configuration.'
} finally {
    foreach ($file in $files) { Remove-Item -LiteralPath (Join-Path $fixture $file) -ErrorAction SilentlyContinue }
    Remove-Item -LiteralPath $apiEnv,$apiDirectory,$fixture -ErrorAction SilentlyContinue
}
