$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot 'release-policy.ps1')
$good=@{DEV_MODE='false';EMAIL_DEBUG_ADDR='';EMAIL_REDIRECT_ALL='false';EMAIL_REDIRECT_TO='';STRIPE_TEST_MODE='false'}
Assert-ProductionEnvironment $good
foreach($entry in @(@('DEV_MODE','true'),@('EMAIL_DEBUG_ADDR','synthetic@example.test'),@('EMAIL_REDIRECT_ALL','true'),@('EMAIL_REDIRECT_TO','synthetic@example.test'),@('STRIPE_TEST_MODE','true'))) {
 $bad=$good.Clone();$bad[$entry[0]]=$entry[1];$rejected=$false
 try { Assert-ProductionEnvironment $bad } catch {$rejected=$true}
 if(!$rejected){throw 'Unsafe production configuration accepted.'}
}
foreach($flag in @('EMAIL_REDIRECT_ALL','STRIPE_TEST_MODE')) {
 foreach($value in @('TRUE','1','t','invalid','')) {
  $bad=$good.Clone();$bad[$flag]=$value;$rejected=$false
  try { Assert-ProductionEnvironment $bad } catch {$rejected=$true}
  if(!$rejected){throw 'Ambiguous or enabled production test flag accepted.'}
 }
}
$root=Split-Path -Parent $PSScriptRoot
$fixture=Join-Path ([IO.Path]::GetTempPath()) ("tm-config-"+[guid]::NewGuid().ToString('N'))
$files=@('docker-compose.yml','docker-compose.override.yml','docker-compose.dev.yml','docker-compose.release.yml')
$apiDirectory=Join-Path $fixture 'api-server'
$apiEnv=Join-Path $apiDirectory '.env'
$composeEnv=Join-Path $fixture '.env'
try {
 New-Item -ItemType Directory -Path $apiDirectory -Force | Out-Null
 foreach ($file in $files) { Copy-Item -LiteralPath (Join-Path $root $file) -Destination $fixture }
 [IO.File]::WriteAllLines($composeEnv,[string[]]@('TM_API_DB_PASSWORD=synthetic-api','TM_DB_PASSWORD=synthetic-db',('TM_RELEASE_API_IMAGE=registry.example.test/tm@sha256:'+('a'*64)), 'TM_DEV_MODE=false','TM_STRIPE_TEST_MODE=false','TM_EMAIL_REDIRECT_ALL=false','TM_EMAIL_REDIRECT_TO=wrong@example.test'))
 foreach($mode in 'true','false') {
  $recipient=if ($mode -eq 'true') {'synthetic@example.test'} else {''}
  [IO.File]::WriteAllLines($apiEnv,[string[]]@('TM_API_DB_PASSWORD=synthetic-api','TM_DB_PASSWORD=synthetic-db',('TM_RELEASE_API_IMAGE=registry.example.test/tm@sha256:'+('a'*64)),"DEV_MODE=$mode","STRIPE_TEST_MODE=$mode","EMAIL_REDIRECT_ALL=$mode","EMAIL_REDIRECT_TO=$recipient",'EMAIL_DEBUG_ADDR='))
  foreach($overlay in '','docker-compose.dev.yml','docker-compose.release.yml') {
   $arguments=@('--env-file',$apiEnv,'--project-directory',$fixture)
   if ($overlay) { $arguments+=@('-f',(Join-Path $fixture 'docker-compose.yml'),'-f',(Join-Path $fixture $overlay)) }
   $text=& docker compose @arguments --profile prod --profile interactive --profile debug config --format json
   if($LASTEXITCODE -ne 0){throw 'Synthetic Compose configuration failed'}
   $services=($text|ConvertFrom-Json).services
   foreach($name in 'apiserver','apiserver-interactive','apiserver-debug') {
    $environment=$services.$name.environment
    if ($environment.TM_DB_PASSWORD) { throw "Database owner password entered $name" }
    if($environment.DEV_MODE -cne $mode -or $environment.STRIPE_TEST_MODE -cne $mode -or $environment.EMAIL_REDIRECT_ALL -cne $mode -or [string]$environment.EMAIL_REDIRECT_TO -cne $recipient) {
     throw "$name did not use its provider settings from api-server/.env"
    }
   }
   if ($overlay -eq 'docker-compose.release.yml' -and $mode -eq 'false') { Assert-ProductionEnvironment $services.apiserver.environment }
  }
 }
 Write-Host 'All Compose profiles and overlays honor api-server/.env.'
} finally {
 foreach ($file in $files) { Remove-Item -LiteralPath (Join-Path $fixture $file) -ErrorAction SilentlyContinue }
 Remove-Item -LiteralPath $apiEnv,$composeEnv,$apiDirectory,$fixture -ErrorAction SilentlyContinue
}
