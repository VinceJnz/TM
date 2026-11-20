[CmdletBinding()]
param()
$ErrorActionPreference = "Stop"
$repositoryRoot = Split-Path -Parent $PSScriptRoot
$image = "golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab"
foreach ($module in @("api-server", "client1")) {
    $modulePath = (Resolve-Path (Join-Path $repositoryRoot $module)).Path
    $command = 'go mod verify && go install golang.org/x/vuln/cmd/govulncheck@v1.10.0 && /go/bin/govulncheck ./...'
    if ($module -eq "client1") {
        $command = 'go mod verify && go install golang.org/x/vuln/cmd/govulncheck@v1.10.0 && GOOS=js GOARCH=wasm /go/bin/govulncheck ./...'
    }
    Write-Host "Auditing $module dependencies..."
    & docker run --rm -v "${modulePath}:/app:ro" -v "tm-codex-go-mod:/go/pkg/mod" -v "tm-codex-go-cache:/root/.cache/go-build" -w /app $image sh -c $command
    if ($LASTEXITCODE -ne 0) { throw "Dependency audit failed for $module" }
}
