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
Invoke-Docker @(
	"run", "--rm", "-v", "${clientPath}:/app:ro", "-w", "/app", "golang:1.23.4-alpine",
	"sh", "-c", 'apk add --no-cache nodejs bash >/dev/null && GOOS=js GOARCH=wasm go test -exec /usr/local/go/misc/wasm/go_js_wasm_exec ./app/httpProcessor'
)

Write-Host "Building the production API image without credentials..."
Invoke-Docker @(
	"build", "-f", (Join-Path $repositoryRoot "Dockerfile.apibuild"),
	"--target", "final", "--build-arg", "CLIENT_IMAGE=tm-client1-wasm:tinygo",
	"-t", "tm-apiserver:verify", $repositoryRoot
)

Write-Host "Checking image identity, credentials, and WASM artifacts..."
Invoke-Docker @(
	"run", "--rm", "--entrypoint", "sh", "tm-apiserver:verify", "-ec",
	'id -u | grep -qx 10001; test ! -e /app/.env; test -s /app/static/main.wasm; test -s /app/static/wasm_exec.js; if find /etc/certs -type f | grep -q .; then exit 1; fi'
)

& (Join-Path $PSScriptRoot "verify-runtime.ps1") -Image "tm-apiserver:verify"
Write-Host "Release verification passed."
