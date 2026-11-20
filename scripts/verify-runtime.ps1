[CmdletBinding()]
param([string]$Image = "tm-apiserver:verify", [switch]$SkipAPITests)
$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$taskPrefix = "tm-smoke-" + [guid]::NewGuid().ToString("N").Substring(0,12)
$network = $taskPrefix + "-net"
$database = $taskPrefix + "-db"
$application = $taskPrefix + "-api"
$certificates = $taskPrefix + "-certs"
$adminSource = "postgres://tm_test:synthetic-test-only@database:5432/tm_test?sslmode=disable"
function Invoke-TestDocker([string[]]$Arguments) {
    & docker @Arguments
    if ($LASTEXITCODE -ne 0) { throw "Docker release smoke verification failed" }
}
try {
    Write-Host "Preparing disposable release verification resources..."
    Invoke-TestDocker @("network","create",$network)
    Invoke-TestDocker @("run","-d","--name",$database,"--network",$network,"--network-alias","database","-e","POSTGRES_USER=tm_test","-e","POSTGRES_PASSWORD=synthetic-test-only","-e","POSTGRES_DB=tm_test","postgres:13@sha256:4689940c683801b4ab839ab3b0a0a3555a5fe425371422310944e89eca7d8068")
    $ready = $false
    for ($attempt=0;$attempt -lt 60;$attempt++) {
        & docker exec $database pg_isready -U tm_test -d tm_test *> $null
        if ($LASTEXITCODE -eq 0) { $ready=$true;break }
        Start-Sleep -Milliseconds 250
    }
    if (!$ready) { throw "Disposable database did not become ready" }
    if (!$SkipAPITests) {
        Write-Host "Running Linux API and PostgreSQL integration tests..."
        $apiPath = (Resolve-Path (Join-Path $repositoryRoot "api-server")).Path
        Invoke-TestDocker @("run","--rm","--network",$network,"-e","TM_TEST_DATABASE_URL=$adminSource","-v","${apiPath}:/app:ro","-v","tm-codex-go-mod:/go/pkg/mod","-v","tm-codex-go-cache:/root/.cache/go-build","-w","/app","golang:1.24.0-alpine@sha256:2d40d4fc278dad38be0777d5e2a88a2c6dee51b0b29c97a764fc6c6a11ca893c","sh","-c","go test -count=1 ./... && go vet ./...")
    }
    Invoke-TestDocker @("run","--rm","--network",$network,"-e","DATA_SOURCE=$adminSource","--entrypoint","/app/tm-migrate",$Image)
    Get-Content -Raw (Join-Path $PSScriptRoot "fixtures/release-smoke.sql") | & docker exec -i $database psql -U tm_test -d tm_test -v ON_ERROR_STOP=1
    if ($LASTEXITCODE -ne 0) { throw "Synthetic fixture initialization failed" }
    Invoke-TestDocker @("run","--rm","--network",$network,"-e","DATA_SOURCE=$adminSource","--entrypoint","/app/tm-migrate",$Image,"-grant-role","tm_smoke_api")
    Invoke-TestDocker @("build","-f",(Join-Path $PSScriptRoot "Dockerfile.smoke"),"-t","tm-runtime-checks:local",$repositoryRoot)
    Invoke-TestDocker @("run","--rm","-v","${certificates}:/certs","--entrypoint","sh","tm-runtime-checks:local","-ec",'openssl req -x509 -newkey rsa:2048 -nodes -days 1 -subj /CN=tm-smoke -addext subjectAltName=DNS:tm-smoke -keyout /certs/server.key -out /certs/server.crt >/dev/null 2>&1; chown -R 10001:10001 /certs; chmod 700 /certs; chmod 600 /certs/server.key')
    Invoke-TestDocker @("run","-d","--name",$application,"--network",$network,"--network-alias","tm-smoke","--read-only","--tmpfs","/tmp","--cap-drop","ALL","--security-opt","no-new-privileges:true","-v","${certificates}:/etc/certs/ssl:ro","-e","DATA_SOURCE=postgres://tm_smoke_api:synthetic-api@database:5432/tm_test?sslmode=disable","-e","HOST=tm-smoke","-e","HTTP_PORT=8085","-e","HTTPS_PORT=8443","-e","CORE_ORIGINS=https://tm-smoke:8443","-e","API_PATH_PREFIX=/custom/v9","-e","APP_TITLE=Smoke","-e","CERT_OPTION=0","-e","DEV_MODE=false","-e","SERVER_CERT=/etc/certs/ssl/server.crt","-e","SERVER_KEY=/etc/certs/ssl/server.key",$Image)
    Invoke-TestDocker @("run","--rm","--network",$network,"tm-runtime-checks:local")
    Invoke-TestDocker @("stop","--time","15",$application)
    $exitStatus = & docker inspect --format '{{.State.ExitCode}}' $application
    if ($LASTEXITCODE -ne 0 -or $exitStatus -ne "0") { throw "Application did not shut down gracefully" }
    Write-Host "Packaged runtime and browser verification passed."
} finally {
    # These unique names are created exclusively by this invocation. No host
    # ports, live databases, existing volumes or external accounts are used.
    $ErrorActionPreference = "Continue"
    & docker rm -f -v $application $database 2>$null | Out-Null
    & docker network rm $network 2>$null | Out-Null
    & docker volume rm $certificates 2>$null | Out-Null
}
