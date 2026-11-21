[CmdletBinding()]
param()
$ErrorActionPreference='Stop'
$root=Split-Path -Parent $PSScriptRoot
$prefix='tm-pg-upgrade-'+[guid]::NewGuid().ToString('N').Substring(0,10)
$network=$prefix+'-net'
$source=$prefix+'-source'
$target=$prefix+'-target'
$backup=$prefix+'-backup'
$pg13='postgres:13@sha256:4689940c683801b4ab839ab3b0a0a3555a5fe425371422310944e89eca7d8068'
$pg17='postgres:17@sha256:b60e90042aa6b8dbecd58e224b3e36114dbf8ad754465264c2476ad58a86e45c'
$go='golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab'
function Invoke-UpgradeDocker([string[]]$Arguments){
 & docker @Arguments
 if($LASTEXITCODE -ne 0){throw 'Disposable upgrade verification failed'}
}
try {
 Invoke-UpgradeDocker @('network','create',$network)
 foreach($item in @(@($source,$pg13,'source','postgres'),@($target,$pg17,'target','tm_upgrade_bootstrap'))){
  # The isolated target accepts preserved MD5 and SCRAM verifiers. Production
  # authentication policy must be reviewed separately; never rewrite a verifier.
  Invoke-UpgradeDocker @('run','-d','--name',$item[0],'--network',$network,'--network-alias',$item[2],'-e',"POSTGRES_USER=$($item[3])",'-e','POSTGRES_HOST_AUTH_METHOD=md5','-e','POSTGRES_PASSWORD=synthetic-test-only','-e','POSTGRES_DB=tm_test',$item[1])
  $ready=$false
  for($i=0;$i -lt 60;$i++){
   & docker exec $item[0] pg_isready -h 127.0.0.1 -U $item[3] -d tm_test *> $null
   if($LASTEXITCODE -eq 0){$ready=$true;break}
   Start-Sleep -Milliseconds 250
  }
  if(!$ready){throw 'Disposable PostgreSQL not ready'}
 }
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','DATA_SOURCE=postgres://postgres:synthetic-test-only@source:5432/tm_test?sslmode=disable','-v',"${root}/api-server:/app:ro",'-v','tm-codex-go-mod:/go/pkg/mod','-v','tm-codex-go-cache:/root/.cache/go-build','-w','/app',$go,'go','run','./cmd/migrate')
 $seed="INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1);INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1);INSERT INTO at_payments(booking_id,payment_date,amount) VALUES(1,CURRENT_DATE,12.34);"
 Invoke-UpgradeDocker @('exec',$source,'psql','-U','postgres','-d','tm_test','-v','ON_ERROR_STOP=1','-c',$seed)
 # Distinct credentials make bootstrap-password substitution detectable. The
 # hashes and archive stay inside this invocation's private Docker volume.
 Invoke-UpgradeDocker @('exec',$source,'psql','-U','postgres','-d','tm_test','-v','ON_ERROR_STOP=1','-c',"SET password_encryption='md5';CREATE ROLE tm_upgrade_runtime LOGIN PASSWORD 'synthetic-runtime-only';ALTER ROLE tm_upgrade_runtime SET statement_timeout='12s';")
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','PGPASSWORD=synthetic-test-only','-v',"${backup}:/backup",$pg17,'pg_dumpall','-h','source','-U','postgres','--roles-only','-f','/backup/roles.sql')
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','PGPASSWORD=synthetic-test-only','-v',"${backup}:/backup",$pg17,'psql','-h','target','-U','tm_upgrade_bootstrap','-d','tm_test','-v','ON_ERROR_STOP=1','-f','/backup/roles.sql')
 # A newer pg_dump reads the old server; the archive never leaves this task volume.
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','PGPASSWORD=synthetic-test-only','-v',"${backup}:/backup",$pg17,'pg_dump','-h','source','-U','postgres','-d','tm_test','--no-owner','--no-acl','-Fc','-f','/backup/synthetic.dump')
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','PGPASSWORD=synthetic-test-only','-v',"${backup}:/backup",$pg17,'pg_restore','-h','target','-U','postgres','-d','tm_test','--no-owner','--no-acl','--exit-on-error','/backup/synthetic.dump')
 $query="SELECT (SELECT count(*) FROM st_users)||':'||(SELECT count(*) FROM at_bookings)||':'||(SELECT sum(amount)::text FROM at_payments)||':'||(SELECT count(*) FROM at_financial_audit)||':'||(SELECT count(*) FROM schema_migrations)"
 $before=& docker exec $source psql -U postgres -d tm_test -A -t -c $query
 if($LASTEXITCODE -ne 0){throw 'Source comparison failed'}
 $after=& docker exec $target psql -U postgres -d tm_test -A -t -c $query
 if($LASTEXITCODE -ne 0 -or $before -cne $after){throw 'Restored synthetic records differ'}
 # Compare protected role metadata without printing password hashes. Ownership
 # is deliberately restored to the original owner, not the bootstrap identity.
 Invoke-UpgradeDocker @('exec',$target,'psql','-U','tm_upgrade_bootstrap','-d','tm_test','-v','ON_ERROR_STOP=1','-c','ALTER DATABASE tm_test OWNER TO postgres')
 $roleQuery="SELECT rolname,rolsuper,rolinherit,rolcreaterole,rolcreatedb,rolcanlogin,rolreplication,rolbypassrls,rolconnlimit,rolpassword,rolvaliduntil,(SELECT string_agg(config,',' ORDER BY config) FROM pg_db_role_setting settings CROSS JOIN unnest(settings.setconfig) config WHERE settings.setrole=roles.oid AND settings.setdatabase=0) FROM pg_authid roles WHERE rolname IN ('postgres','tm_upgrade_runtime') ORDER BY rolname"
 $sourceRoles=& docker exec $source psql -U postgres -d tm_test -A -t -c $roleQuery
 if($LASTEXITCODE -ne 0){throw 'Source role inventory failed'}
 $targetRoles=& docker exec $target psql -U tm_upgrade_bootstrap -d tm_test -A -t -c $roleQuery
 if($LASTEXITCODE -ne 0 -or ($sourceRoles -join "`n") -cne ($targetRoles -join "`n")){throw 'Restored role credentials or attributes differ'}
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','DATA_SOURCE=postgres://postgres:synthetic-test-only@target:5432/tm_test?sslmode=disable','-v',"${root}/api-server:/app:ro",'-v','tm-codex-go-mod:/go/pkg/mod','-v','tm-codex-go-cache:/root/.cache/go-build','-w','/app',$go,'go','run','./cmd/migrate','-grant-role','tm_upgrade_runtime')
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','PGPASSWORD=synthetic-runtime-only',$pg17,'psql','-h','target','-U','tm_upgrade_runtime','-d','tm_test','-v','ON_ERROR_STOP=1','-c',"BEGIN;INSERT INTO st_token(user_id,name,token) VALUES(0,'synthetic-oauth','synthetic-state');ROLLBACK;")
 Invoke-UpgradeDocker @('run','--rm','--network',$network,'-e','PGPASSWORD=synthetic-test-only',$pg17,'psql','-h','target','-U','postgres','-d','tm_test','-v','ON_ERROR_STOP=1','-c','SELECT 1')
 $ErrorActionPreference='Continue'
 try {
  & docker run --rm --network $network -e PGPASSWORD=synthetic-test-only $pg17 psql -h target -U tm_upgrade_runtime -d tm_test -c 'SELECT 1' *> $null
  $wrongPasswordStatus=$LASTEXITCODE
 } finally { $ErrorActionPreference='Stop' }
 if($wrongPasswordStatus -eq 0){throw 'Bootstrap password unexpectedly authenticates runtime role'}
 Write-Host 'Synthetic PostgreSQL 13 to 17 data, role credentials, TCP logins, ledger evidence and runtime grants passed.'
} finally {
 $ErrorActionPreference='Continue'
 # Only resources named uniquely by this invocation are removed.
 & docker rm -f -v $source $target 2>$null | Out-Null
 & docker network rm $network 2>$null | Out-Null
 & docker volume rm $backup 2>$null | Out-Null
}
