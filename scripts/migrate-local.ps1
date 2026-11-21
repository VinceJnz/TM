[CmdletBinding()]
param(
    [string]$DatabaseUser = 'myuser',
    [string]$DatabaseName = 'mydatabase',
    [ValidateRange(1, 65535)]
    [int]$Port = 5432,
    [string]$RuntimeRole = 'api_user'
)

$ErrorActionPreference = 'Stop'
$apiPath = Join-Path (Split-Path -Parent $PSScriptRoot) 'api-server'
$password = Read-Host "PostgreSQL password for $DatabaseUser" -AsSecureString
$credential = [pscredential]::new($DatabaseUser, $password)
$previousDataSource = $env:DATA_SOURCE

Push-Location -LiteralPath $apiPath
try {
    $source = [UriBuilder]::new('postgres', '127.0.0.1', $Port)
    $source.UserName = [Uri]::EscapeDataString($DatabaseUser)
    $source.Password = [Uri]::EscapeDataString($credential.GetNetworkCredential().Password)
    $source.Path = '/' + [Uri]::EscapeDataString($DatabaseName)
    $source.Query = 'sslmode=disable'
    $env:DATA_SOURCE = $source.Uri.AbsoluteUri

    & go run ./cmd/migrate -grant-role $RuntimeRole
    if ($LASTEXITCODE -ne 0) {
        throw "Migration command failed with exit code $LASTEXITCODE."
    }
} finally {
    $env:DATA_SOURCE = $previousDataSource
    $password.Dispose()
    Pop-Location
}
