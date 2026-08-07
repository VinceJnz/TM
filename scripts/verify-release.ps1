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

Write-Host "Running API tests in Linux Go 1.24..."
Invoke-Docker @(
	"run", "--rm",
	"-v", "${apiPath}:/app:ro",
	"-w", "/app",
	"golang:1.24.0-alpine",
	"sh", "-c", "go test ./..."
)

Write-Host "Release verification passed."
