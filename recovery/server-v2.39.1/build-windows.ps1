$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go build -trimpath -buildvcs=false -ldflags='-s -w -H=windowsgui' -o AlphaWorkRecoveryTest.exe .
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed' }
    Get-FileHash -Algorithm SHA256 .\AlphaWorkRecoveryTest.exe
} finally { Pop-Location }
