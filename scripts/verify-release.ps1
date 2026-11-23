[CmdletBinding()]
param()

$ErrorActionPreference = "Stop"

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$apiPath = (Resolve-Path (Join-Path $repositoryRoot "api-server")).Path

function Invoke-Docker {
	param([string[]]$Arguments)

	& docker @Arguments
	if ($LASTEXITCODE -ne 0) {
		throw "Docker command failed: docker $($Arguments -join ' ')"
	}
}

Write-Host "Checking release identity and health policy..."
& (Join-Path $PSScriptRoot "test-release-policy.ps1")
& (Join-Path $PSScriptRoot "test-build-policy.ps1")
& (Join-Path $PSScriptRoot "test-release-health.ps1")
& (Join-Path $PSScriptRoot "test-provider-sandbox-policy.ps1")
& (Join-Path $PSScriptRoot "test-production-config.ps1")
Write-Host "Checking generated API contracts..."
Invoke-Docker @("run","--rm","-v","${repositoryRoot}:/repo:ro","-w","/repo/api-server","golang:1.26.11-alpine@sha256:ecb67f987aa1a57794c91d19ade7ae0209f2f0f5496aaef9e6ae85ddf92d02ab","go","run","./cmd/contracts","-check")
Write-Host "Building the TinyGo release client (TinyGo 0.40.0)..."
Invoke-Docker @(
	"build",
	"-f", (Join-Path $repositoryRoot "Dockerfile.client1"),
	"--build-arg", "CLIENT_BUILDER=tinygo-builder",
	"-t", "tm-client1-wasm:tinygo",
	$repositoryRoot
)

Write-Host "Checking the standard Go frontend build..."
Invoke-Docker @(
	"build", "-f", (Join-Path $repositoryRoot "Dockerfile.client1"),
	"--build-arg", "CLIENT_BUILDER=go-builder", "-t", "tm-client1-wasm:go-verify", $repositoryRoot
)

Write-Host "Running frontend WASM transport tests..."
$clientPath = (Resolve-Path (Join-Path $repositoryRoot "client1")).Path
Invoke-Docker @("build", "-f", (Join-Path $repositoryRoot "scripts/Dockerfile.smoke"), "--target", "wasm-tests", "-t", "tm-wasm-tests:verify", $repositoryRoot)
Invoke-Docker @("run", "--rm", "-v", "${clientPath}:/app:ro", "tm-wasm-tests:verify")


Write-Host "Building the production API image without credentials..."
Invoke-Docker @(
	"build", "-f", (Join-Path $repositoryRoot "Dockerfile.apibuild"),
	"--target", "final", "--build-arg", "CLIENT_IMAGE=tm-client1-wasm:tinygo",
	"-t", "tm-apiserver:verify", $repositoryRoot
)

Write-Host "Checking image identity, credentials, and WASM artifacts..."
$image = Invoke-Docker @("image", "inspect", "tm-apiserver:verify") | ConvertFrom-Json
if ($image[0].Config.Labels.'org.tripmanager.client.image' -cne "tm-client1-wasm:tinygo") {
	throw "API image did not record the selected client artifact."
}
Invoke-Docker @(
	"run", "--rm", "--entrypoint", "sh", "tm-apiserver:verify", "-ec",
	'id -u | grep -qx 10001; test ! -e /app/.env; test -s /app/static/main.wasm; test -s /app/static/wasm_exec.js; if find /etc/certs -type f | grep -q .; then exit 1; fi'
)

& (Join-Path $PSScriptRoot "verify-runtime.ps1") -Image "tm-apiserver:verify"
Write-Host "Release verification passed."
