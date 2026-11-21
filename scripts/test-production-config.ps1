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
$empty=[IO.Path]::GetTempFileName();$previous=@{}
$values=@{TM_API_ENV_FILE=$empty;TM_API_DB_PASSWORD='synthetic-api';TM_DB_PASSWORD='synthetic-db';TM_EMAIL_REDIRECT_ALL='false';TM_EMAIL_REDIRECT_TO='synthetic@example.test';TM_STRIPE_TEST_MODE='false'}
try {
 foreach($name in $values.Keys){$previous[$name]=[Environment]::GetEnvironmentVariable($name,'Process');[Environment]::SetEnvironmentVariable($name,$values[$name],'Process')}
 $root=Split-Path -Parent $PSScriptRoot
 # Includes the auto-loaded override: --profile prod must still disable DEV_MODE.
 $text=& docker compose --env-file $empty --project-directory $root --profile prod config --format json
 if($LASTEXITCODE -ne 0){throw 'Default Compose failed'}
 $base=($text|ConvertFrom-Json).services.apiserver.environment
 if($base.DEV_MODE -cne 'false'){throw 'Default override enabled DEV_MODE'}
 $text=& docker compose --env-file $empty --project-directory $root -f (Join-Path $root 'docker-compose.yml') -f (Join-Path $root 'docker-compose.dev.yml') --profile prod config --format json
 if($LASTEXITCODE -ne 0){throw 'Explicit test Compose failed'}
 $test=($text|ConvertFrom-Json).services.apiserver.environment
 if($test.DEV_MODE -cne 'true' -or $test.EMAIL_REDIRECT_ALL -cne 'true' -or $test.STRIPE_TEST_MODE -cne 'true' -or $test.EMAIL_REDIRECT_TO -cne 'synthetic@example.test'){throw 'Test safety flags missing'}
 Write-Host 'Production/test configuration separation passed.'
} finally {
 foreach($name in $previous.Keys){[Environment]::SetEnvironmentVariable($name,$previous[$name],'Process')}
 Remove-Item -LiteralPath $empty
}
